// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package terminal

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

	logger := slog.New(NewConsoleHandler(&output, slog.LevelDebug, false))
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

	logger := slog.New(NewConsoleHandler(&output, slog.LevelDebug, false))
	logger.Error("registrar la máquina", "vm", "mint", "error", "sin acceso")

	line := output.String()
	if !strings.Contains(line, "ERROR") || !strings.Contains(line, "registrar la máquina") {
		t.Errorf("log = %q, want nivel y mensaje", line)
	}

	if !strings.Contains(line, `error="sin acceso"`) {
		t.Errorf("log = %q, want el atributo entre comillas", line)
	}
}

// TestHandlerWithAttrsAndGroups comprueba que los atributos fijos y los
// del registro llegan al destino del log.
func TestHandlerWithAttrsAndGroups(t *testing.T) {
	t.Parallel()

	var destination bytes.Buffer

	handler := NewConsoleHandler(&destination, slog.LevelDebug, false).
		WithAttrs([]slog.Attr{slog.String("clave", "valor")}).
		WithGroup("grupo")

	slog.New(handler).Info("mensaje", "otra", "dato")

	output := destination.String()
	for _, expected := range []string{"mensaje", "clave=valor", "otra=dato"} {
		if !strings.Contains(output, expected) {
			t.Errorf("log = %q, want %q", output, expected)
		}
	}
}

// TestLevelAndStateColor cubre las ramas de color del formato.
func TestLevelAndStateColor(t *testing.T) {
	t.Parallel()

	var destination bytes.Buffer

	handler, ok := NewConsoleHandler(&destination, slog.LevelDebug, true).(*consoleHandler)
	if !ok {
		t.Fatalf("NewConsoleHandler() = %T, want *consoleHandler",
			NewConsoleHandler(&destination, slog.LevelDebug, true))
	}

	if !strings.Contains(handler.levelLabel(slog.LevelWarn), "WARN") {
		t.Error("levelLabel(WARN) no contiene WARN")
	}

	if !strings.Contains(handler.stateColor(503, "503"), "503") {
		t.Error("stateColor(503) no conserva el texto")
	}
}
