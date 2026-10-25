// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestConObservabilidadRegistra comprueba el identificador, el estado y
// la máquina en el log.
func TestConObservabilidadRegistra(t *testing.T) {
	t.Parallel()

	var registro bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&registro, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := ConObservabilidad(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
		func() string { return maquinaDebian },
		logger,
	)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, rutaPaquetes, http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get(cabeceraSolicitud) == "" {
		t.Error("falta X-Request-ID en la respuesta")
	}

	salida := registro.String()
	for _, esperado := range []string{"solicitud=", "metodo=POST", "estado=201", "vm=" + maquinaDebian} {
		if !strings.Contains(salida, esperado) {
			t.Errorf("log = %q, want %q", salida, esperado)
		}
	}
}

// TestConObservabilidadOmitaEstaticos comprueba que solo registra /api.
func TestConObservabilidadOmitaEstaticos(t *testing.T) {
	t.Parallel()

	var registro bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&registro, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := ConObservabilidad(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), nil, logger)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/assets/main.js", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if registro.Len() != 0 {
		t.Errorf("log = %q, want vacío para un recurso estático", registro.String())
	}
}
