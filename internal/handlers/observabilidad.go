// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// msgAPI es el mensaje común de las peticiones registradas.
const msgAPI = "api"

// cabeceraSolicitud identifica cada petición en los logs y en la
// respuesta para correlacionar un error del frontend con el servidor.
const cabeceraSolicitud = "X-Request-ID"

// escritorEstado conserva el código y el tamaño de la respuesta.
type escritorEstado struct {
	http.ResponseWriter
	estado int
	bytes  int
}

// WriteHeader registra el primer estado escrito.
func (w *escritorEstado) WriteHeader(codigo int) {
	if w.estado == 0 {
		w.estado = codigo
	}

	w.ResponseWriter.WriteHeader(codigo)
}

// Write registra 200 si el handler no llamó a WriteHeader.
func (w *escritorEstado) Write(datos []byte) (int, error) {
	if w.estado == 0 {
		w.estado = http.StatusOK
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
		}

		if nombre := nombreVM(vm); nombre != "" {
			atributos = append(atributos, "vm", nombre)
		}

		logger.Log(context.Background(), nivelDeEstado(estado), msgAPI, atributos...)
	})
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
