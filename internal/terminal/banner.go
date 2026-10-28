// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package terminal presenta la consola en la terminal: el banner de
// arranque y el formato compacto de los logs.
package terminal

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

// vmconsArt es el nombre de la consola en arte ASCII, tal como se muestra
// al arrancar.
const vmconsArt = `
 ██╗   ██╗███╗   ███╗ ██████╗ ██████╗ ███╗   ██╗███████╗
 ██║   ██║████╗ ████║██╔════╝██╔═══██╗████╗  ██║██╔════╝
 ██║   ██║██╔████╔██║██║     ██║   ██║██╔██╗ ██║███████╗
 ╚██╗ ██╔╝██║╚██╔╝██║██║     ██║   ██║██║╚██╗██║╚════██║
  ╚████╔╝ ██║ ╚═╝ ██║╚██████╗╚██████╔╝██║ ╚████║███████║
   ╚═══╝  ╚═╝     ╚═╝ ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝╚══════╝`

// Códigos de color ANSI que usa el arranque cuando la salida es una
// terminal. Si no, el banner se imprime sin secuencias de control para
// no ensuciar los archivos de registro.
const (
	colorRed = "\x1b[38;2;192;57;43m"
	colorDim = "\x1b[38;2;102;102;102m"
	colorEnd = "\x1b[0m"
)

// PrintBanner presenta la consola al arrancar: el nombre en arte ASCII, la
// dirección donde escucha y la URL que debe abrir el usuario. Devuelve el
// error de escritura, que el llamador solo registra: no poder pintar el
// banner no impide servir.
func PrintBanner(destination io.Writer, address string) error {
	red, dim, end := "", "", ""

	if ColorEnabled(destination) {
		red, dim, end = colorRed, colorDim, colorEnd
	}

	lines := []string{
		fmt.Sprintf("%s%s%s", red, vmconsArt, end),
		"",
		fmt.Sprintf("  %sconsola de automatización de máquinas virtuales%s", dim, end),
		"",
		fmt.Sprintf("  %sescuchando en%s  %s", dim, end, URL(address)),
		"",
	}

	if _, err := io.WriteString(destination, strings.Join(lines, "\n")+"\n"); err != nil {
		return fmt.Errorf("escribir el banner de arranque: %w", err)
	}

	return nil
}

// URL traduce la dirección de escucha a la URL que se anuncia. Una
// dirección sin host, o con la dirección de cualquier interfaz, se anuncia
// como localhost, que es donde el usuario abre el navegador.
func URL(address string) string {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "http://" + address
	}

	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}

	return "http://" + net.JoinHostPort(host, port)
}

// ColorEnabled informa si el escritor es una terminal y el usuario no pidió
// desactivar el color con NO_COLOR.
func ColorEnabled(destination io.Writer) bool {
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
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
