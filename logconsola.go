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
	ansiFin      = "\x1b[0m"
)

// handlerConsola escribe logs compactos y, si la salida es una terminal,
// colorea el nivel y el estado HTTP.
type handlerConsola struct {
	mu      *sync.Mutex
	destino io.Writer
	nivel   slog.Level
	color   bool
	attrs   []slog.Attr
}

// nuevoHandlerConsola construye el handler.
func nuevoHandlerConsola(destino io.Writer, nivel slog.Level, color bool) slog.Handler {
	return &handlerConsola{mu: &sync.Mutex{}, destino: destino, nivel: nivel, color: color}
}

// Enabled implementa slog.Handler.
func (h *handlerConsola) Enabled(_ context.Context, nivel slog.Level) bool {
	return nivel >= h.nivel
}

// Handle implementa slog.Handler.
//
//nolint:gocritic // slog.Handler exige el valor de slog.Record.
func (h *handlerConsola) Handle(_ context.Context, registro slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var linea strings.Builder

	linea.WriteString(h.pintar(ansiGris, registro.Time.Format("15:04:05")))
	linea.WriteByte(' ')
	linea.WriteString(h.nivelTexto(registro.Level))
	linea.WriteByte(' ')

	atributos := map[string]string{}
	for _, attr := range h.attrs {
		atributos[attr.Key] = attr.Value.String()
	}

	registro.Attrs(func(attr slog.Attr) bool {
		atributos[attr.Key] = attr.Value.String()

		return true
	})

	if registro.Message == "api" {
		h.escribirAPI(&linea, atributos)
	} else {
		linea.WriteString(registro.Message)

		for _, attr := range h.attrs {
			h.escribirAtributo(&linea, attr)
		}

		registro.Attrs(func(attr slog.Attr) bool {
			h.escribirAtributo(&linea, attr)

			return true
		})
	}

	linea.WriteByte('\n')

	if _, err := io.WriteString(h.destino, linea.String()); err != nil {
		return fmt.Errorf("escribir el log: %w", err)
	}

	return nil
}

// WithAttrs implementa slog.Handler.
func (h *handlerConsola) WithAttrs(attrs []slog.Attr) slog.Handler {
	copia := *h
	copia.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)

	return &copia
}

// WithGroup implementa slog.Handler; el formato plano no usa grupos.
func (h *handlerConsola) WithGroup(string) slog.Handler {
	return h
}

// escribirAPI formatea una petición HTTP.
func (h *handlerConsola) escribirAPI(linea *strings.Builder, atributos map[string]string) {
	estado, err := strconv.Atoi(atributos["estado"])
	if err != nil {
		estado = 0
	}

	linea.WriteString(h.pintar(ansiCian, "API"))
	linea.WriteString("  ")
	linea.WriteString(atributos["metodo"])
	linea.WriteByte(' ')
	linea.WriteString(atributos["ruta"])
	linea.WriteByte(' ')
	linea.WriteString(h.colorEstado(estado, atributos["estado"]))
	linea.WriteString("  ")
	linea.WriteString(atributos["duracion_ms"])
	linea.WriteString("ms")

	if vm := atributos["vm"]; vm != "" {
		linea.WriteString("  vm=")
		linea.WriteString(vm)
	}

	if solicitud := atributos["solicitud"]; solicitud != "" {
		linea.WriteString("  req=")
		linea.WriteString(solicitud)
	}
}

// escribirAtributo agrega «clave=valor» al log general.
func (h *handlerConsola) escribirAtributo(linea *strings.Builder, attr slog.Attr) {
	valor := attr.Value.String()
	if strings.ContainsAny(valor, " \t\"") {
		valor = strconv.Quote(valor)
	}

	linea.WriteByte(' ')
	linea.WriteString(attr.Key)
	linea.WriteByte('=')
	linea.WriteString(valor)
}

// nivelTexto pinta y alinea el nivel.
func (h *handlerConsola) nivelTexto(nivel slog.Level) string {
	switch {
	case nivel >= slog.LevelError:
		return h.pintar(ansiRojo, "ERROR")
	case nivel >= slog.LevelWarn:
		return h.pintar(ansiAmarillo, "WARN ")
	case nivel >= slog.LevelInfo:
		return h.pintar(ansiVerde, "INFO ")
	default:
		return h.pintar(ansiGris, "DEBUG")
	}
}

// colorEstado pinta el código HTTP según su familia.
func (h *handlerConsola) colorEstado(codigo int, texto string) string {
	switch {
	case codigo >= 500:
		return h.pintar(ansiRojo, texto)
	case codigo >= 400:
		return h.pintar(ansiAmarillo, texto)
	default:
		return h.pintar(ansiVerde, texto)
	}
}

// pintar aplica el color solo si está habilitado.
func (h *handlerConsola) pintar(color, texto string) string {
	if !h.color {
		return texto
	}

	return color + texto + ansiFin
}

// colorEn indica si el escritor es una terminal y NO_COLOR no está
// definido.
func colorEn(destino io.Writer) bool {
	if _, desactivado := os.LookupEnv("NO_COLOR"); desactivado {
		return false
	}

	archivo, ok := destino.(*os.File)
	if !ok {
		return false
	}

	info, err := archivo.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}
