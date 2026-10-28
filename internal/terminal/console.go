// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package terminal

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
)

// Códigos ANSI para los niveles del log.
const (
	ansiRed    = "\x1b[31m"
	ansiYellow = "\x1b[33m"
	ansiGreen  = "\x1b[32m"
	ansiCyan   = "\x1b[36m"
	ansiGray   = "\x1b[90m"
	ansiEnd    = "\x1b[0m"
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

// NewConsoleHandler construye el handler compacto de la consola.
func NewConsoleHandler(destination io.Writer, level slog.Level, color bool) slog.Handler {
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

	line.WriteString(h.colorize(ansiGray, registration.Time.Format("15:04:05")))
	line.WriteByte(' ')
	line.WriteString(h.levelLabel(registration.Level))
	line.WriteByte(' ')

	attributes := map[string]string{}
	for _, attr := range h.attrs {
		attributes[attr.Key] = attr.Value.String()
	}

	registration.Attrs(func(attr slog.Attr) bool {
		attributes[attr.Key] = attr.Value.String()

		return true
	})

	if registration.Message == "api" {
		h.writeAPI(&line, attributes)
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
func (h *consoleHandler) writeAPI(line *strings.Builder, attributes map[string]string) {
	state, err := strconv.Atoi(attributes["estado"])
	if err != nil {
		state = 0
	}

	line.WriteString(h.colorize(ansiCyan, "API"))
	line.WriteString("  ")
	line.WriteString(attributes["metodo"])
	line.WriteByte(' ')
	line.WriteString(attributes["ruta"])
	line.WriteByte(' ')
	line.WriteString(h.stateColor(state, attributes["estado"]))
	line.WriteString("  ")
	line.WriteString(attributes["duracion_ms"])
	line.WriteString("ms")

	if description := attributes["descripcion"]; description != "" {
		line.WriteString("  ")
		line.WriteString(description)
	}

	if reason := attributes["motivo"]; reason != "" {
		line.WriteString("  motivo=")
		line.WriteString(strconv.Quote(reason))
	}

	if vm := attributes["vm"]; vm != "" {
		line.WriteString("  vm=")
		line.WriteString(vm)
	}

	if request := attributes["solicitud"]; request != "" {
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
		return h.colorize(ansiRed, "ERROR")
	case level >= slog.LevelWarn:
		return h.colorize(ansiYellow, "WARN ")
	case level >= slog.LevelInfo:
		return h.colorize(ansiGreen, "INFO ")
	default:
		return h.colorize(ansiGray, "DEBUG")
	}
}

// stateColor pinta el código HTTP según su familia.
func (h *consoleHandler) stateColor(code int, text string) string {
	switch {
	case code >= 500:
		return h.colorize(ansiRed, text)
	case code >= 400:
		return h.colorize(ansiYellow, text)
	default:
		return h.colorize(ansiGreen, text)
	}
}

// colorize aplica el color solo si está habilitado.
func (h *consoleHandler) colorize(color, text string) string {
	if !h.color {
		return text
	}

	return color + text + ansiEnd
}
