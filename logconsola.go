// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Códigos ANSI para los niveles del log.
const (
	ansiRojo     = "\x1b[31m"
	ansiAmarillo = "\x1b[33m"
	ansiVerde    = "\x1b[32m"
	ansiCian     = "\x1b[36m"
	ansiGris     = "\x1b[90m"
	ansiEnd      = "\x1b[0m"
)

// consoleHandler escribe logs compactos y, si la salida es una terminal,
// colorea el nivel y el estado HTTP.
type consoleHandler struct {
	mu          *sync.Mutex
	destination io.Writer
	level       slog.Level
	color       bool
	attrs       []slog.Attr
}

// newConsoleHandler construye el handler.
func newConsoleHandler(destination io.Writer, level slog.Level, color bool) slog.Handler {
	return &consoleHandler{mu: &sync.Mutex{}, destination: destination, level: level, color: color}
}

// Enabled implementa slog.Handler.
func (h *consoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle implementa slog.Handler.
//
//nolint:gocritic // slog.Handler exige el valor de slog.Record.
func (h *consoleHandler) Handle(_ context.Context, registration slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var line strings.Builder

	line.WriteString(h.colorize(ansiGris, registration.Time.Format("15:04:05")))
	line.WriteByte(' ')
	line.WriteString(h.levelLabel(registration.Level))
	line.WriteByte(' ')

	atributos := map[string]string{}
	for _, attr := range h.attrs {
		atributos[attr.Key] = attr.Value.String()
	}

	registration.Attrs(func(attr slog.Attr) bool {
		atributos[attr.Key] = attr.Value.String()

		return true
	})

	if registration.Message == "api" {
		h.writeAPI(&line, atributos)
	} else {
		line.WriteString(registration.Message)

		for _, attr := range h.attrs {
			h.writeAttribute(&line, attr)
		}

		registration.Attrs(func(attr slog.Attr) bool {
			h.writeAttribute(&line, attr)

			return true
		})
	}

	line.WriteByte('\n')

	if _, err := io.WriteString(h.destination, line.String()); err != nil {
		return fmt.Errorf("escribir el log: %w", err)
	}

	return nil
}

// WithAttrs implementa slog.Handler.
func (h *consoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	duplicate := *h
	duplicate.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)

	return &duplicate
}

// WithGroup implementa slog.Handler; el formato plano no usa grupos.
func (h *consoleHandler) WithGroup(string) slog.Handler {
	return h
}

// writeAPI formatea una petición HTTP.
func (h *consoleHandler) writeAPI(line *strings.Builder, atributos map[string]string) {
	state, err := strconv.Atoi(atributos["estado"])
	if err != nil {
		state = 0
	}

	line.WriteString(h.colorize(ansiCian, "API"))
	line.WriteString("  ")
	line.WriteString(atributos["metodo"])
	line.WriteByte(' ')
	line.WriteString(atributos["ruta"])
	line.WriteByte(' ')
	line.WriteString(h.stateColor(state, atributos["estado"]))
	line.WriteString("  ")
	line.WriteString(atributos["duracion_ms"])
	line.WriteString("ms")

	if description := atributos["descripcion"]; description != "" {
		line.WriteString("  ")
		line.WriteString(description)
	}

	if reason := atributos["motivo"]; reason != "" {
		line.WriteString("  motivo=")
		line.WriteString(strconv.Quote(reason))
	}

	if vm := atributos["vm"]; vm != "" {
		line.WriteString("  vm=")
		line.WriteString(vm)
	}

	if request := atributos["solicitud"]; request != "" {
		line.WriteString("  req=")
		line.WriteString(request)
	}
}

// writeAttribute agrega «clave=valor» al log general.
func (h *consoleHandler) writeAttribute(line *strings.Builder, attr slog.Attr) {
	value := attr.Value.String()
	if strings.ContainsAny(value, " \t\"") {
		value = strconv.Quote(value)
	}

	line.WriteByte(' ')
	line.WriteString(attr.Key)
	line.WriteByte('=')
	line.WriteString(value)
}

// levelLabel pinta y alinea el nivel.
func (h *consoleHandler) levelLabel(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return h.colorize(ansiRojo, "ERROR")
	case level >= slog.LevelWarn:
		return h.colorize(ansiAmarillo, "WARN ")
	case level >= slog.LevelInfo:
		return h.colorize(ansiVerde, "INFO ")
	default:
		return h.colorize(ansiGris, "DEBUG")
	}
}

// stateColor pinta el código HTTP según su familia.
func (h *consoleHandler) stateColor(code int, text string) string {
	switch {
	case code >= 500:
		return h.colorize(ansiRojo, text)
	case code >= 400:
		return h.colorize(ansiAmarillo, text)
	default:
		return h.colorize(ansiVerde, text)
	}
}

// colorize aplica el color solo si está habilitado.
func (h *consoleHandler) colorize(color, text string) string {
	if !h.color {
		return text
	}

	return color + text + ansiEnd
}

// colorEnabledFor indica si el escritor es una terminal y NO_COLOR no está
// definido.
func colorEnabledFor(destination io.Writer) bool {
	if _, desactivado := os.LookupEnv("NO_COLOR"); desactivado {
		return false
	}

	file, ok := destination.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}
