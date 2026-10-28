// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package webui

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// pageTests es el index.html mínimo que usan las pruebas.
const pageTests = "<html>ok</html>"

// TestResolveEmbedded usa la copia incluida en el binario.
func TestResolveEmbedded(t *testing.T) {
	t.Parallel()

	embedded := fstest.MapFS{
		"web/dist/index.html": &fstest.MapFile{Data: []byte(pageTests)},
	}

	interfaceFS, err := Resolve("", embedded)
	if err != nil {
		t.Fatalf("Resolve() error = %v, want nil", err)
	}

	if _, err := fs.ReadFile(interfaceFS, "index.html"); err != nil {
		t.Errorf("ReadFile(index.html) error = %v, want nil", err)
	}
}

// TestResolveDisk prefiere el directorio indicado por el operador.
func TestResolveDisk(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "index.html"), []byte(pageTests), 0o600); err != nil {
		t.Fatalf("preparar el index.html: %v", err)
	}

	interfaceFS, err := Resolve(directory, fstest.MapFS{})
	if err != nil {
		t.Fatalf("Resolve() error = %v, want nil", err)
	}

	if _, err := fs.ReadFile(interfaceFS, "index.html"); err != nil {
		t.Errorf("ReadFile(index.html) error = %v, want nil", err)
	}
}

// TestResolveRejectsDirectoryWithoutIndex exige una aplicación compilada.
func TestResolveRejectsDirectoryWithoutIndex(t *testing.T) {
	t.Parallel()

	if _, err := Resolve(t.TempDir(), fstest.MapFS{}); err == nil {
		t.Error("Resolve() error = nil, want el fallo por falta de index.html")
	}
}

// TestFromDiskRejectsFile exige un directorio.
func TestFromDiskRejectsFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "index.html")
	if err := os.WriteFile(path, []byte(pageTests), 0o600); err != nil {
		t.Fatalf("crear el archivo: %v", err)
	}

	if _, err := FromDisk(path); err == nil {
		t.Error("FromDisk() error = nil, want el fallo de directorio")
	}
}

// TestMountServesInterface publica la interfaz compilada y la anterior.
func TestMountServesInterface(t *testing.T) {
	t.Parallel()

	dist := fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte(pageTests)},
		"assets/app.js": &fstest.MapFile{Data: []byte("app")},
	}
	legacy := fstest.MapFS{
		"modules/index.html": &fstest.MapFile{Data: []byte("legacy")},
	}

	mux := http.NewServeMux()
	Mount(mux, dist, legacy)

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))

	if recorder.Code != http.StatusOK || recorder.Body.String() != pageTests {
		t.Errorf("GET / = %d %q, want 200 y la interfaz compilada", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/modules/", http.NoBody))

	if recorder.Body.String() != "legacy" {
		t.Errorf("GET /modules/ = %q, want la interfaz anterior", recorder.Body.String())
	}
}
