// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package main

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"consola-vm/internal/adapters/localproc"
	"consola-vm/internal/services"
)

// TestPoliciesCacheDeclaraValidities comprueba que cada recurso tiene una
// política de caché.
func TestPoliciesCacheDeclaraValidities(t *testing.T) {
	t.Parallel()

	policies := cachePolicies()
	if len(policies) == 0 {
		t.Fatal("politicasCache() = vacío, want al menos una política")
	}

	for _, policy := range policies {
		if !strings.HasPrefix(policy.Prefijo, "/api/") || policy.TTL <= 0 {
			t.Errorf("política inválida: %+v", policy)
		}
	}
}

// TestAddressRespetaEntorno devuelve la dirección configurada.
func TestAddressRespetaEntorno(t *testing.T) {
	t.Setenv(envAddress, "127.0.0.1:9090")

	if address() != "127.0.0.1:9090" {
		t.Errorf("direccion() = %q, want 127.0.0.1:9090", address())
	}

	t.Setenv(envAddress, "")

	if address() != addressProd {
		t.Errorf("direccion() = %q, want %q", address(), addressProd)
	}
}

// TestRutasByOmision comprueba la base, la clave y la llave.
func TestRutasByOmision(t *testing.T) {
	t.Setenv(envDB, "")
	t.Setenv(envSSHKey, "")

	if path := databasePath(); path != filepath.Join("data", "vmcons.db") {
		t.Errorf("rutaBaseDatos() = %q, want data/vmcons.db", path)
	}

	base := filepath.Join(t.TempDir(), "datos", "vmcons.db")
	t.Setenv(envDB, base)

	if path := cipherKeyPath(); path != filepath.Join(filepath.Dir(base), "credenciales.key") {
		t.Errorf("rutaClaveCifrado() = %q, want junto a la base", path)
	}

	t.Setenv(envSSHKey, "/tmp/llave")

	if path := sshKeyPath(); path != "/tmp/llave" {
		t.Errorf("rutaLlave() = %q, want /tmp/llave", path)
	}

	t.Setenv(envSSHKey, "")

	if path := sshKeyPath(); !strings.HasSuffix(path, filepath.Join("vmcons", "id_ed25519")) {
		t.Errorf("rutaLlave() = %q, want la llave en el directorio de configuración", path)
	}
}

// TestUserInvitadoByOmision devuelve root.
func TestUserInvitadoByOmision(t *testing.T) {
	t.Setenv(envGuestUser, "")

	if user := guestUser(); user != "root" {
		t.Errorf("usuarioInvitado() = %q, want root", user)
	}

	t.Setenv(envGuestUser, "jorge")

	if user := guestUser(); user != "jorge" {
		t.Errorf("usuarioInvitado() = %q, want jorge", user)
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

	t.Setenv(envVM, "")

	if target := initialTarget(store); target.ID != "" {
		t.Errorf("objetivoInicial() = %q, want vacío", target.ID)
	}

	if err := store.SaveActive("vm-1"); err != nil {
		t.Fatalf("guardar la máquina: %v", err)
	}

	if target := initialTarget(store); target.ID != "vm-1" {
		t.Errorf("objetivoInicial() = %q, want vm-1", target.ID)
	}

	t.Setenv(envVM, "ubuntu-lab")

	if target := initialTarget(store); target.ID != "ubuntu-lab" {
		t.Errorf("objetivoInicial() = %q, want la variable de entorno", target.ID)
	}
}

// TestColorEnabledRespetaNotColor comprueba la variable de entorno.
func TestColorEnabledRespetaNotColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	if colorEnabled() {
		t.Error("colorHabilitado() = true con NO_COLOR definido")
	}
}

// TestHandlerWithAtributosYGroups comprueba que los atributos fijos y los
// del registro llegan al destino del log.
func TestHandlerWithAtributosYGroups(t *testing.T) {
	t.Parallel()

	var destination bytes.Buffer

	handler := newConsoleHandler(&destination, slog.LevelDebug, false).
		WithAttrs([]slog.Attr{slog.String("clave", "valor")}).
		WithGroup("grupo")

	slog.New(handler).Info("mensaje", "otra", "dato")

	output := destination.String()
	for _, expected := range []string{"mensaje", "clave=valor", "otra=dato"} {
		if !strings.Contains(output, expected) {
			t.Errorf("log = %q, want %q", output, expected)
		}
	}
}

// TestLevelYStatePintanWithColor cubre las ramas de color del formato.
func TestLevelYStatePintanWithColor(t *testing.T) {
	t.Parallel()

	var destination bytes.Buffer

	handler, ok := newConsoleHandler(&destination, slog.LevelDebug, true).(*consoleHandler)
	if !ok {
		t.Fatalf("nuevoHandlerConsola() = %T, want *handlerConsola", newConsoleHandler(&destination, slog.LevelDebug, true))
	}

	if !strings.Contains(handler.levelLabel(slog.LevelWarn), "WARN") {
		t.Error("nivelTexto(WARN) no contiene WARN")
	}

	if !strings.Contains(handler.stateColor(503, "503"), "503") {
		t.Error("colorEstado(503) no conserva el texto")
	}
}

// TestColorInWithDestinationThatNotIsFile devuelve false.
func TestColorInWithDestinationThatNotIsFile(t *testing.T) {
	t.Parallel()

	if colorEnabledFor(&bytes.Buffer{}) {
		t.Error("colorEn(buffer) = true, want false")
	}
}

// TestInterfazWebEmbebida usa la copia del binario.
func TestInterfazWebEmbebida(t *testing.T) {
	t.Setenv(envWeb, "")

	interfaz, err := webInterface()
	if err != nil {
		t.Fatalf("interfazWeb() error = %v, want nil", err)
	}

	if _, err := interfaz.Open("index.html"); err != nil {
		t.Errorf("Open(index.html) error = %v, want nil", err)
	}
}

// TestInterfazInDiscoRejectsFile exige un directorio.
func TestInterfazInDiscoRejectsFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "index.html")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("crear el archivo: %v", err)
	}

	if _, err := webFromDisk(path); err == nil {
		t.Error("interfazEnDisco() error = nil, want el fallo de directorio")
	}
}

// entornoTemp apunta la base y las llaves a un directorio temporal.
func entornoTemp(t *testing.T) {
	t.Helper()

	directory := t.TempDir()
	t.Setenv(envDB, filepath.Join(directory, "vmcons.db"))
	t.Setenv(envSSHKey, filepath.Join(directory, "id_ed25519"))
	t.Setenv(envWeb, "")
	t.Setenv(envVM, "")
}

// TestNewServerArmaServer comprueba la dirección y los tiempos.
func TestNewServerArmaServer(t *testing.T) {
	entornoTemp(t)
	t.Setenv(envAddress, "127.0.0.1:0")

	server, err := newServer()
	if err != nil {
		t.Fatalf("nuevoServidor() error = %v, want nil", err)
	}

	if server.Addr != "127.0.0.1:0" || server.Handler == nil {
		t.Errorf("nuevoServidor() = %+v, want dirección y manejador", server)
	}
}

// TestNewMuxProtegeAPI comprueba el ensamblaje completo.
func TestNewMuxProtegeAPI(t *testing.T) {
	entornoTemp(t)

	handler, err := newMux()
	if err != nil {
		t.Fatalf("nuevoMux() error = %v, want nil", err)
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

// TestNewConsoleArmaServices comprueba la inyección de dependencias.
func TestNewConsoleArmaServices(t *testing.T) {
	entornoTemp(t)

	console := newConsole(localproc.New())
	if console.sessions == nil || console.access == nil || console.registration == nil {
		t.Errorf("nuevaConsola() = %+v, want los servicios armados", console)
	}
}

// TestRunFallWithAddressInvalida devuelve el error del servidor.
func TestRunFallWithAddressInvalida(t *testing.T) {
	entornoTemp(t)
	t.Setenv(envAddress, "256.256.256.256:99999")
	t.Setenv("NO_COLOR", "1")

	if err := run(); err == nil {
		t.Error("ejecutar() error = nil, want el fallo de la dirección")
	}
}

// TestPrintBannerWritesURL captura la salida del banner.
func TestPrintBannerWritesURL(t *testing.T) {
	read, escritura, err := os.Pipe()
	if err != nil {
		t.Fatalf("crear la tubería: %v", err)
	}

	previous := os.Stdout

	os.Stdout = escritura

	t.Setenv("NO_COLOR", "1")

	failure := printBanner("127.0.0.1:8080")

	os.Stdout = previous

	if err := escritura.Close(); err != nil {
		t.Fatalf("cerrar la tubería: %v", err)
	}

	output, err := io.ReadAll(read)
	if err != nil {
		t.Fatalf("leer la salida: %v", err)
	}

	if failure != nil {
		t.Fatalf("imprimirBanner() error = %v, want nil", failure)
	}

	if !strings.Contains(string(output), "http://127.0.0.1:8080") {
		t.Errorf("banner = %q, want la URL de la consola", output)
	}
}
