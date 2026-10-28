// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package webui resuelve de dónde se sirve la interfaz web y la publica en
// el enrutador HTTP de la consola.
package webui

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

// Resolve devuelve el sistema de archivos de la aplicación web: el
// directorio indicado por el operador o, si está vacío, la copia embebida
// en el binario.
func Resolve(dir string, embedded fs.FS) (fs.FS, error) {
	if dir != "" {
		return FromDisk(dir)
	}

	dist, err := fs.Sub(embedded, "web/dist")
	if err != nil {
		return nil, fmt.Errorf("preparar la interfaz embebida: %w", err)
	}

	return dist, nil
}

// FromDisk valida el directorio indicado por el operador y lo publica como
// sistema de archivos.
func FromDisk(dir string) (fs.FS, error) {
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

// Mount publica la aplicación compilada con Vite en la raíz y conserva la
// interfaz anterior en /modules/ y /static/ mientras se retira.
func Mount(mux *http.ServeMux, dist, legacy fs.FS) {
	mux.Handle("/modules/", http.FileServerFS(legacy))
	mux.Handle("/static/", http.FileServerFS(legacy))

	// El enrutado por fragmento de URL no necesita reescrituras.
	mux.Handle("/assets/", http.FileServerFS(dist))
	mux.Handle("/", http.FileServerFS(dist))
}
