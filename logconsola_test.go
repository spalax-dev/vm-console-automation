// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// TestHandlerFormateaAPI comprueba la línea compacta de una petición.
func TestHandlerFormateaAPI(t *testing.T) {
	t.Parallel()

	var salida bytes.Buffer

	logger := slog.New(nuevoHandlerConsola(&salida, slog.LevelDebug, false))
	logger.Info("api",
		"solicitud", "abc123",
		"metodo", "GET",
		"ruta", "/api/session",
		"estado", 503,
		"duracion_ms", 128,
		"vm", "base",
	)

	linea := salida.String()
	for _, esperado := range []string{"API", "GET /api/session", "503", "128ms", "vm=base", "req=abc123"} {
		if !strings.Contains(linea, esperado) {
			t.Errorf("log = %q, want %q", linea, esperado)
		}
	}
}

// TestHandlerFormateaError comprueba el formato de un error general.
func TestHandlerFormateaError(t *testing.T) {
	t.Parallel()

	var salida bytes.Buffer

	logger := slog.New(nuevoHandlerConsola(&salida, slog.LevelDebug, false))
	logger.Error("registrar la máquina", "vm", "mint", "error", "sin acceso")

	linea := salida.String()
	if !strings.Contains(linea, "ERROR") || !strings.Contains(linea, "registrar la máquina") {
		t.Errorf("log = %q, want nivel y mensaje", linea)
	}

	if !strings.Contains(linea, `error="sin acceso"`) {
		t.Errorf("log = %q, want el atributo entre comillas", linea)
	}
}
