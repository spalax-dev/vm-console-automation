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

// TestConObservabilidadDescribeYExplicaError comprueba que el log
// incluye el resumen de la lectura y el motivo del 503.
func TestConObservabilidadDescribeYExplicaError(t *testing.T) {
	t.Parallel()

	var registro bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&registro, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := ConObservabilidad(
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

	salida := registro.String()
	for _, esperado := range []string{
		"estado=503",
		"Consultar el estado de la sesión",
		"motivo=",
		"No hay una sesión disponible",
	} {
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

// TestConObservabilidadNoMutilaCuerpoDeErrorLargo comprueba que un cuerpo
// de error mayor a 512 bytes no se trunca al entregarse al cliente HTTP.
func TestConObservabilidadNoMutilaCuerpoDeErrorLargo(t *testing.T) {
	t.Parallel()

	var registro bytes.Buffer

	cuerpoLargo := `{"mensaje":"` + strings.Repeat("a", 800) + `"}`
	logger := slog.New(slog.NewTextHandler(&registro, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := ConObservabilidad(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)

			if _, err := w.Write([]byte(cuerpoLargo)); err != nil {
				t.Errorf("escribir cuerpo: %v", err)
			}
		}),
		nil,
		logger,
	)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/test", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != cuerpoLargo {
		t.Errorf("cuerpo entregado = %d bytes, want %d bytes (se mutiló la respuesta)", rec.Body.Len(), len(cuerpoLargo))
	}
}

// TestConObservabilidadConservaLaSolicitudDelCliente comprueba que un
// identificador corto enviado por el cliente se conserva para poder
// correlacionar el error del frontend con el log.
func TestConObservabilidadConservaLaSolicitudDelCliente(t *testing.T) {
	t.Parallel()

	handler := ConObservabilidad(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), nil, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, rutaSesion, http.NoBody)
	req.Header.Set(cabeceraSolicitud, "solicitud-123")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if obtenida := rec.Header().Get(cabeceraSolicitud); obtenida != "solicitud-123" {
		t.Errorf("%s = %q, want la solicitud del cliente", cabeceraSolicitud, obtenida)
	}
}
