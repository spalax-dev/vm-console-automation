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

// TestWithObservabilityRegisters comprueba el identificador, el estado y
// la máquina en el log.
func TestWithObservabilityRegisters(t *testing.T) {
	t.Parallel()

	var registration bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&registration, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := WithObservability(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
		func() string { return testMachine },
		logger,
	)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, pathPackages, http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get(headerRequest) == "" {
		t.Error("falta X-Request-ID en la respuesta")
	}

	output := registration.String()
	for _, expected := range []string{"solicitud=", "metodo=POST", "estado=201", "vm=" + testMachine} {
		if !strings.Contains(output, expected) {
			t.Errorf("log = %q, want %q", output, expected)
		}
	}
}

// TestWithObservabilityDescribeYExplainsError comprueba que el log
// incluye el resumen de la lectura y el motivo del 503.
func TestWithObservabilityDescribeYExplainsError(t *testing.T) {
	t.Parallel()

	var registration bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&registration, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := WithObservability(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)

			if _, err := w.Write([]byte(`{"mensaje":"No hay una sesión disponible con la máquina."}`)); err != nil {
				t.Errorf("escribir el cuerpo de error: %v", err)
			}
		}),
		nil,
		logger,
	)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/session", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	output := registration.String()
	for _, expected := range []string{
		"estado=503",
		"Consultar el estado de la sesión",
		"motivo=",
		"No hay una sesión disponible",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("log = %q, want %q", output, expected)
		}
	}
}

// TestWithObservabilityOmitaEstaticos comprueba que solo registra /api.
func TestWithObservabilityOmitaEstaticos(t *testing.T) {
	t.Parallel()

	var registration bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&registration, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := WithObservability(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), nil, logger)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/assets/main.js", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if registration.Len() != 0 {
		t.Errorf("log = %q, want vacío para un recurso estático", registration.String())
	}
}

// TestWithObservabilityNotMutilatesBodyOfErrorLong comprueba que un cuerpo
// de error mayor a 512 bytes no se trunca al entregarse al cliente HTTP.
func TestWithObservabilityNotMutilatesBodyOfErrorLong(t *testing.T) {
	t.Parallel()

	var registration bytes.Buffer

	bodyLong := `{"mensaje":"` + strings.Repeat("a", 800) + `"}`
	logger := slog.New(slog.NewTextHandler(&registration, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := WithObservability(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)

			if _, err := w.Write([]byte(bodyLong)); err != nil {
				t.Errorf("escribir cuerpo: %v", err)
			}
		}),
		nil,
		logger,
	)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/test", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != bodyLong {
		t.Errorf("cuerpo entregado = %d bytes, want %d bytes (se mutiló la respuesta)", rec.Body.Len(), len(bodyLong))
	}
}

// TestWithObservabilityKeepsRequestOfCliente comprueba que un
// identificador corto enviado por el cliente se conserva para poder
// correlacionar el error del frontend con el log.
func TestWithObservabilityKeepsRequestOfCliente(t *testing.T) {
	t.Parallel()

	handler := WithObservability(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), nil, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, pathSession, http.NoBody)
	req.Header.Set(headerRequest, "solicitud-123")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if obtenida := rec.Header().Get(headerRequest); obtenida != "solicitud-123" {
		t.Errorf("%s = %q, want la solicitud del cliente", headerRequest, obtenida)
	}
}
