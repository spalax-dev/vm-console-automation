// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package main

import (
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
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
	colorRojo  = "\x1b[38;2;192;57;43m"
	colorTenue = "\x1b[38;2;102;102;102m"
	colorEnd   = "\x1b[0m"
)

// printBanner presenta la consola al arrancar: el nombre en arte
// ASCII, la dirección donde escucha, el origen de la interfaz web y la
// máquina configurada. Devuelve el error de escritura, que el llamador
// solo registra: no poder pintar el banner no impide servir.
//
// direccion es la dirección de escucha del servidor HTTP; origen, el
// texto que describe de dónde se sirve la interfaz; maquina, el nombre o
// UUID configurado, que puede estar vacío.
func printBanner(address string) error {
	rojo, tenue, end := "", "", ""

	if colorEnabled() {
		rojo, tenue, end = colorRojo, colorTenue, colorEnd
	}

	lines := []string{
		fmt.Sprintf("%s%s%s", rojo, vmconsArt, end),
		"",
		fmt.Sprintf("  %sconsola de automatización de máquinas virtuales%s", tenue, end),
		"",
		fmt.Sprintf("  %sescuchando en%s  %s", tenue, end, consoleURL(address)),
		"",
	}

	if _, err := os.Stdout.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
		return fmt.Errorf("escribir el banner de arranque: %w", err)
	}

	return nil
}

// consoleURL traduce la dirección de escucha a la URL que se anuncia.
// Una dirección sin host, o con la dirección de cualquier interfaz, se
// anuncia como localhost, que es donde el usuario abre el navegador.
func consoleURL(address string) string {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "http://" + address
	}

	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}

	return "http://" + net.JoinHostPort(host, port)
}

// webInterface devuelve el sistema de archivos de la aplicación web: el
// directorio indicado por VMCONS_WEB o, si la variable falta, la copia
// embebida en el binario.
func webInterface() (fs.FS, error) {
	if dir := os.Getenv(envWeb); dir != "" {
		return webFromDisk(dir)
	}

	dist, err := fs.Sub(embeddedWeb, "web/dist")
	if err != nil {
		return nil, fmt.Errorf("preparar la interfaz embebida: %w", err)
	}

	return dist, nil
}

// webFromDisk valida el directorio indicado por el operador y lo
// publica como sistema de archivos.
func webFromDisk(dir string) (fs.FS, error) {
	// #nosec G703 -- la ruta la indica el operador en VMCONS_WEB.
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("abrir la interfaz web en %q: %w", dir, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("la interfaz web en %q no es un directorio", dir)
	}

	// #nosec G703 -- la ruta la indica el operador en VMCONS_WEB.
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		return nil, fmt.Errorf("la interfaz web en %q no contiene index.html: %w", dir, err)
	}

	return os.DirFS(dir), nil
}

// colorEnabled informa si la salida estándar es una terminal y el
// usuario no pidió desactivar el color con NO_COLOR.
func colorEnabled() bool {
	if _, desactivado := os.LookupEnv("NO_COLOR"); desactivado {
		return false
	}

	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}
