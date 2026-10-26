// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"consola-vm/internal/services"
)

// msgAPI es el mensaje común de las peticiones registradas.
const msgAPI = "api"

// headerRequest identifica cada petición en los logs y en la
// respuesta para correlacionar un error del frontend con el servidor.
const headerRequest = "X-Request-ID"

// maxBodyLog acota el cuerpo de error que se usa para explicar el
// motivo en el log.
const maxBodyLog = 512

// writerState conserva el código, el tamaño y el cuerpo de error de
// la respuesta.
type writerState struct {
	http.ResponseWriter
	state    int
	bytes    int
	capturar bool
	captures bytes.Buffer
}

// WriteHeader registra el primer estado escrito.
func (w *writerState) WriteHeader(code int) {
	if w.state == 0 {
		w.state = code
		w.capturar = code >= http.StatusBadRequest
	}

	w.ResponseWriter.WriteHeader(code)
}

// Write registra 200 si el handler no llamó a WriteHeader.
func (w *writerState) Write(data []byte) (int, error) {
	if w.state == 0 {
		w.state = http.StatusOK
	}

	if w.capturar && w.captures.Len() < maxBodyLog {
		restante := maxBodyLog - w.captures.Len()

		trozo := data
		if len(trozo) > restante {
			trozo = trozo[:restante]
		}

		_, _ = w.captures.Write(trozo)
	}

	escritos, err := w.ResponseWriter.Write(data)
	w.bytes += escritos

	if err != nil {
		return escritos, fmt.Errorf("escribir la respuesta: %w", err)
	}

	return escritos, nil
}

// WithObservability registra cada petición de la API con un
// identificador de solicitud, duración, estado y máquina activa.
func WithObservability(next http.Handler, vm func() string, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)

			return
		}

		request := r.Header.Get(headerRequest)
		if request == "" || len(request) > 64 {
			request = newRequestID()
		}

		w.Header().Set(headerRequest, request)

		data := readBody(r)
		completeData(r, data)

		start := time.Now()
		registration := &writerState{ResponseWriter: w}

		next.ServeHTTP(registration, r)

		state := registration.state
		if state == 0 {
			state = http.StatusOK
		}

		path := r.Pattern
		if path == "" {
			path = r.URL.Path
		}

		atributos := []any{
			"solicitud", request,
			"metodo", r.Method,
			"ruta", path,
			"estado", state,
			"duracion_ms", time.Since(start).Milliseconds(),
			"bytes", registration.bytes,
			"descripcion", services.DescribeOperation(r.Method, path, r.URL.Path, data),
		}

		if reason := reasonOf(state, registration.captures.Bytes()); reason != "" {
			atributos = append(atributos, "motivo", reason)
		}

		if name := vmName(vm); name != "" {
			atributos = append(atributos, "vm", name)
		}

		logger.Log(context.Background(), stateLevel(state), msgAPI, atributos...)
	})
}

// reasonOf extrae el mensaje comprensible del cuerpo de error, sin
// incluir el detalle técnico.
func reasonOf(state int, body []byte) string {
	if state < http.StatusBadRequest || len(body) == 0 {
		return ""
	}

	var response struct {
		Message string `json:"mensaje"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return ""
	}

	return truncate(strings.TrimSpace(response.Message), 200)
}

// truncate limita un texto sin partir un carácter UTF-8.
func truncate(text string, limit int) string {
	characters := []rune(text)
	if len(characters) <= limit {
		return text
	}

	return string(characters[:limit]) + "…"
}

// stateLevel clasifica el estado HTTP: 503 es una indisponibilidad
// temporal (advertencia), no un error interno.
func stateLevel(state int) slog.Level {
	switch {
	case state >= http.StatusInternalServerError && state != http.StatusServiceUnavailable:
		return slog.LevelError
	case state >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

// vmName devuelve la máquina activa sin romper si no hay vínculo.
func vmName(vm func() string) string {
	if vm == nil {
		return ""
	}

	return vm()
}

// readAleatorio llena el identificador de solicitud; es una variable para
// poder simular su fallo en las pruebas.
var readAleatorio = rand.Read

// newRequestID genera un identificador corto y aleatorio.
func newRequestID() string {
	data := make([]byte, 8)
	if _, err := readAleatorio(data); err != nil {
		return time.Now().UTC().Format("20060102T150405.000000")
	}

	return hex.EncodeToString(data)
}
