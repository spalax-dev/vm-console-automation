// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"consola-vm/internal/adapters/localproc"
	"consola-vm/internal/config"
	"consola-vm/internal/services"
)

// pageTests es el index.html mínimo que usan las pruebas del servidor.
const pageTests = "<html>ok</html>"

// TestPoliciesCacheDeclaraValidities comprueba que cada recurso tiene una
// política de caché.
func TestPoliciesCacheDeclaraValidities(t *testing.T) {
	t.Parallel()

	policies := cachePolicies()
	if len(policies) == 0 {
		t.Fatal("cachePolicies() = vacío, want al menos una política")
	}

	for _, policy := range policies {
		if !strings.HasPrefix(policy.Prefijo, "/api/") || policy.TTL <= 0 {
			t.Errorf("política inválida: %+v", policy)
		}
	}
}

// TestTargetInicialRestoresMachine usa la máquina persistida.
func TestTargetInicialRestoresMachine(t *testing.T) {
	directory := t.TempDir()

	store, err := services.NewStore(
		filepath.Join(directory, "vmcons.db"), filepath.Join(directory, "credenciales.key"))
	if err != nil {
		t.Fatalf("abrir el almacén: %v", err)
	}

	t.Setenv(config.EnvVM, "")

	if target := initialTarget(store); target.ID != "" {
		t.Errorf("initialTarget() = %q, want vacío", target.ID)
	}

	if err := store.SaveActive("vm-1"); err != nil {
		t.Fatalf("guardar la máquina: %v", err)
	}

	if target := initialTarget(store); target.ID != "vm-1" {
		t.Errorf("initialTarget() = %q, want vm-1", target.ID)
	}

	t.Setenv(config.EnvVM, "ubuntu-lab")

	if target := initialTarget(store); target.ID != "ubuntu-lab" {
		t.Errorf("initialTarget() = %q, want la variable de entorno", target.ID)
	}
}

// tempEnv apunta la base y las llaves a un directorio temporal.
func tempEnv(t *testing.T) {
	t.Helper()

	directory := t.TempDir()
	t.Setenv(config.EnvDB, filepath.Join(directory, "vmcons.db"))
	t.Setenv(config.EnvSSHKey, filepath.Join(directory, "id_ed25519"))
	t.Setenv(config.EnvWeb, "")
	t.Setenv(config.EnvVM, "")
}

// TestNewServerArmaServer comprueba la dirección y los tiempos.
func TestNewServerArmaServer(t *testing.T) {
	tempEnv(t)
	t.Setenv(config.EnvAddress, "127.0.0.1:0")

	server, err := newServer()
	if err != nil {
		t.Fatalf("newServer() error = %v, want nil", err)
	}

	if server.Addr != "127.0.0.1:0" || server.Handler == nil {
		t.Errorf("newServer() = %+v, want dirección y manejador", server)
	}
}

// TestNewMuxProtegeAPI comprueba el ensamblaje completo.
func TestNewMuxProtegeAPI(t *testing.T) {
	tempEnv(t)

	handler, err := newMux()
	if err != nil {
		t.Fatalf("newMux() error = %v, want nil", err)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/api/auth/estado", http.NoBody))

	if rec.Code != http.StatusOK {
		t.Errorf("estado = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/api/packages", http.NoBody))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("packages = %d, want 401 sin sesión", rec.Code)
	}
}

// TestServeWebUsesEnvWeb comprueba que VMCONS_WEB manda sobre la copia
// embebida y que la interfaz queda publicada.
func TestServeWebUsesEnvWeb(t *testing.T) {
	directory := t.TempDir()

	if err := os.WriteFile(filepath.Join(directory, "index.html"), []byte(pageTests), 0o600); err != nil {
		t.Fatalf("preparar el index.html: %v", err)
	}

	t.Setenv(config.EnvWeb, directory)

	mux := http.NewServeMux()
	if err := serveWeb(mux); err != nil {
		t.Fatalf("serveWeb() error = %v, want nil", err)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))

	if rec.Code != http.StatusOK || rec.Body.String() != pageTests {
		t.Errorf("GET / = %d %q, want 200 y la página indicada", rec.Code, rec.Body.String())
	}
}

// TestNewConsoleArmaServices comprueba la inyección de dependencias.
func TestNewConsoleArmaServices(t *testing.T) {
	tempEnv(t)

	console := newConsole(localproc.New())
	if console.sessions == nil || console.access == nil || console.registration == nil {
		t.Errorf("newConsole() = %+v, want los servicios armados", console)
	}
}

// TestRunFallWithAddressInvalida devuelve el error del servidor.
func TestRunFallWithAddressInvalida(t *testing.T) {
	tempEnv(t)
	t.Setenv(config.EnvAddress, "256.256.256.256:99999")
	t.Setenv("NO_COLOR", "1")

	if err := run(); err == nil {
		t.Error("run() error = nil, want el fallo de la dirección")
	}
}
