// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Valores repetidos en las pruebas del arranque.
const (
	paginaPrueba  = "<html>ok</html>"
	directorioWeb = "/opt/vmcons/web"
)

// TestURLConsola comprueba la dirección que se anuncia al arrancar.
func TestURLConsola(t *testing.T) {
	t.Parallel()

	casos := map[string]string{
		":8080":           "http://localhost:8080",
		"0.0.0.0:8080":    "http://localhost:8080",
		"127.0.0.1:9000":  "http://127.0.0.1:9000",
		"192.168.1.10:80": "http://192.168.1.10:80",
		"[::1]:8080":      "http://[::1]:8080",
		"localhost:3000":  "http://localhost:3000",
	}

	for entrada, esperado := range casos {
		if obtenido := urlConsola(entrada); obtenido != esperado {
			t.Errorf("urlConsola(%q) = %q, want %q", entrada, obtenido, esperado)
		}
	}
}

// TestInterfazWebDesdeElEntorno comprueba que VMCONS_WEB manda sobre la
// copia embebida.
func TestInterfazWebDesdeElEntorno(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(paginaPrueba), 0o600); err != nil {
		t.Fatalf("preparar el index.html: %v", err)
	}

	t.Setenv(envWeb, dir)

	interfaz, err := interfazWeb()
	if err != nil {
		t.Fatalf("interfazWeb() error = %v, want nil", err)
	}

	contenido, err := fs.ReadFile(interfaz, "index.html")
	if err != nil {
		t.Fatalf("leer index.html: %v", err)
	}

	if string(contenido) != paginaPrueba {
		t.Errorf("index.html = %q, want %q", contenido, paginaPrueba)
	}
}

// TestInterfazWebRechazaUnaRutaInexistente evita arrancar con una
// interfaz que no existe.
func TestInterfazWebRechazaUnaRutaInexistente(t *testing.T) {
	t.Setenv(envWeb, filepath.Join(t.TempDir(), "no-existe"))

	if _, err := interfazWeb(); err == nil {
		t.Fatal("interfazWeb() error = nil, want error")
	}
}

// TestInterfazWebRechazaUnDirectorioSinIndex exige que la ruta apunte a
// una aplicación compilada.
func TestInterfazWebRechazaUnDirectorioSinIndex(t *testing.T) {
	t.Setenv(envWeb, t.TempDir())

	if _, err := interfazWeb(); err == nil {
		t.Fatal("interfazWeb() error = nil, want error")
	}
}

// TestOrigenInterfazDistingueDiscoYEmbebida comprueba el texto que se
// muestra en el arranque.
func TestOrigenInterfazDistingueDiscoYEmbebida(t *testing.T) {
	t.Setenv(envWeb, "")

	if origen := origenInterfaz(); !strings.Contains(origen, "embebida") {
		t.Errorf("origenInterfaz() = %q, want que mencione la copia embebida", origen)
	}

	t.Setenv(envWeb, directorioWeb)

	if origen := origenInterfaz(); origen != directorioWeb {
		t.Errorf("origenInterfaz() = %q, want %q", origen, directorioWeb)
	}
}
