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

// cabeceraSolicitud identifica cada petición en los logs y en la
// respuesta para correlacionar un error del frontend con el servidor.
const cabeceraSolicitud = "X-Request-ID"

// maxCuerpoLog acota el cuerpo de error que se usa para explicar el
// motivo en el log.
const maxCuerpoLog = 512

// escritorEstado conserva el código, el tamaño y el cuerpo de error de
// la respuesta.
type escritorEstado struct {
	http.ResponseWriter
	estado   int
	bytes    int
	capturar bool
	captura  bytes.Buffer
}

// WriteHeader registra el primer estado escrito.
func (w *escritorEstado) WriteHeader(codigo int) {
	if w.estado == 0 {
		w.estado = codigo
		w.capturar = codigo >= http.StatusBadRequest
	}

	w.ResponseWriter.WriteHeader(codigo)
}

// Write registra 200 si el handler no llamó a WriteHeader.
func (w *escritorEstado) Write(datos []byte) (int, error) {
	if w.estado == 0 {
		w.estado = http.StatusOK
	}

	if w.capturar && w.captura.Len() < maxCuerpoLog {
		restante := maxCuerpoLog - w.captura.Len()

		trozo := datos
		if len(trozo) > restante {
			trozo = trozo[:restante]
		}

		_, _ = w.captura.Write(trozo)
	}

	escritos, err := w.ResponseWriter.Write(datos)
	w.bytes += escritos

	if err != nil {
		return escritos, fmt.Errorf("escribir la respuesta: %w", err)
	}

	return escritos, nil
}

// ConObservabilidad registra cada petición de la API con un
// identificador de solicitud, duración, estado y máquina activa.
func ConObservabilidad(siguiente http.Handler, vm func() string, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			siguiente.ServeHTTP(w, r)

			return
		}

		solicitud := r.Header.Get(cabeceraSolicitud)
		if solicitud == "" || len(solicitud) > 64 {
			solicitud = nuevaSolicitud()
		}

		w.Header().Set(cabeceraSolicitud, solicitud)

		datos := leerCuerpo(r)
		completarDatos(r, datos)

		inicio := time.Now()
		registro := &escritorEstado{ResponseWriter: w}

		siguiente.ServeHTTP(registro, r)

		estado := registro.estado
		if estado == 0 {
			estado = http.StatusOK
		}

		ruta := r.Pattern
		if ruta == "" {
			ruta = r.URL.Path
		}

		atributos := []any{
			"solicitud", solicitud,
			"metodo", r.Method,
			"ruta", ruta,
			"estado", estado,
			"duracion_ms", time.Since(inicio).Milliseconds(),
			"bytes", registro.bytes,
			"descripcion", services.DescribirOperacion(r.Method, ruta, r.URL.Path, datos),
		}

		if motivo := motivoDe(estado, registro.captura.Bytes()); motivo != "" {
			atributos = append(atributos, "motivo", motivo)
		}

		if nombre := nombreVM(vm); nombre != "" {
			atributos = append(atributos, "vm", nombre)
		}

		logger.Log(context.Background(), nivelDeEstado(estado), msgAPI, atributos...)
	})
}

// motivoDe extrae el mensaje comprensible del cuerpo de error, sin
// incluir el detalle técnico.
func motivoDe(estado int, cuerpo []byte) string {
	if estado < http.StatusBadRequest || len(cuerpo) == 0 {
		return ""
	}

	var respuesta struct {
		Mensaje string `json:"mensaje"`
	}

	if err := json.Unmarshal(cuerpo, &respuesta); err != nil {
		return ""
	}

	return recortar(strings.TrimSpace(respuesta.Mensaje), 200)
}

// recortar limita un texto sin partir un carácter UTF-8.
func recortar(texto string, maximo int) string {
	caracteres := []rune(texto)
	if len(caracteres) <= maximo {
		return texto
	}

	return string(caracteres[:maximo]) + "…"
}

// nivelDeEstado clasifica el estado HTTP: 503 es una indisponibilidad
// temporal (advertencia), no un error interno.
func nivelDeEstado(estado int) slog.Level {
	switch {
	case estado >= http.StatusInternalServerError && estado != http.StatusServiceUnavailable:
		return slog.LevelError
	case estado >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

// nombreVM devuelve la máquina activa sin romper si no hay vínculo.
func nombreVM(vm func() string) string {
	if vm == nil {
		return ""
	}

	return vm()
}

// nuevaSolicitud genera un identificador corto y aleatorio.
func nuevaSolicitud() string {
	datos := make([]byte, 8)
	if _, err := rand.Read(datos); err != nil {
		return time.Now().UTC().Format("20060102T150405.000000")
	}

	return hex.EncodeToString(datos)
}
