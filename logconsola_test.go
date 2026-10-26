// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// TestHandlerFormatsAPI comprueba la línea compacta de una petición.
func TestHandlerFormatsAPI(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	logger := slog.New(newConsoleHandler(&output, slog.LevelDebug, false))
	logger.Info("api",
		"solicitud", "abc123",
		"metodo", "GET",
		"ruta", "/api/session",
		"estado", 503,
		"duracion_ms", 128,
		"descripcion", "Consultar el estado de la sesión",
		"motivo", "No hay una sesión disponible",
		"vm", "base",
	)

	line := output.String()
	for _, expected := range []string{
		"API", "GET /api/session", "503", "128ms",
		"Consultar el estado de la sesión", `motivo="No hay una sesión disponible"`,
		"vm=base", "req=abc123",
	} {
		if !strings.Contains(line, expected) {
			t.Errorf("log = %q, want %q", line, expected)
		}
	}
}

// TestHandlerFormatsError comprueba el formato de un error general.
func TestHandlerFormatsError(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	logger := slog.New(newConsoleHandler(&output, slog.LevelDebug, false))
	logger.Error("registrar la máquina", "vm", "mint", "error", "sin acceso")

	line := output.String()
	if !strings.Contains(line, "ERROR") || !strings.Contains(line, "registrar la máquina") {
		t.Errorf("log = %q, want nivel y mensaje", line)
	}

	if !strings.Contains(line, `error="sin acceso"`) {
		t.Errorf("log = %q, want el atributo entre comillas", line)
	}
}
