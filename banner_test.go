// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// Valores repetidos en las pruebas del arranque.
const paginaTests = "<html>ok</html>"

// TestURLConsole comprueba la dirección que se anuncia al arrancar.
func TestURLConsole(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		":8080":           "http://localhost:8080",
		"0.0.0.0:8080":    "http://localhost:8080",
		"127.0.0.1:9000":  "http://127.0.0.1:9000",
		"192.168.1.10:80": "http://192.168.1.10:80",
		"[::1]:8080":      "http://[::1]:8080",
		"localhost:3000":  "http://localhost:3000",
	}

	for entry, expected := range testCases {
		if obtained := consoleURL(entry); obtained != expected {
			t.Errorf("urlConsola(%q) = %q, want %q", entry, obtained, expected)
		}
	}
}

// TestInterfazWebFromEntorno comprueba que VMCONS_WEB manda sobre la
// copia embebida.
func TestInterfazWebFromEntorno(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(paginaTests), 0o600); err != nil {
		t.Fatalf("preparar el index.html: %v", err)
	}

	t.Setenv(envWeb, dir)

	interfaz, err := webInterface()
	if err != nil {
		t.Fatalf("interfazWeb() error = %v, want nil", err)
	}

	content, err := fs.ReadFile(interfaz, "index.html")
	if err != nil {
		t.Fatalf("leer index.html: %v", err)
	}

	if string(content) != paginaTests {
		t.Errorf("index.html = %q, want %q", content, paginaTests)
	}
}

// TestInterfazWebRejectsPathInexistente evita arrancar con una
// interfaz que no existe.
func TestInterfazWebRejectsPathInexistente(t *testing.T) {
	t.Setenv(envWeb, filepath.Join(t.TempDir(), "no-existe"))

	if _, err := webInterface(); err == nil {
		t.Fatal("interfazWeb() error = nil, want error")
	}
}

// TestInterfazWebRejectsDirectoryWithoutIndex exige que la ruta apunte a
// una aplicación compilada.
func TestInterfazWebRejectsDirectoryWithoutIndex(t *testing.T) {
	t.Setenv(envWeb, t.TempDir())

	if _, err := webInterface(); err == nil {
		t.Fatal("interfazWeb() error = nil, want error")
	}
}
