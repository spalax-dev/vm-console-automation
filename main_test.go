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

// TestPoliticasCacheDeclaraVigencias comprueba que cada recurso tiene una
// política de caché.
func TestPoliticasCacheDeclaraVigencias(t *testing.T) {
	t.Parallel()

	politicas := politicasCache()
	if len(politicas) == 0 {
		t.Fatal("politicasCache() = vacío, want al menos una política")
	}

	for _, politica := range politicas {
		if !strings.HasPrefix(politica.Prefijo, "/api/") || politica.TTL <= 0 {
			t.Errorf("política inválida: %+v", politica)
		}
	}
}

// TestDireccionRespetaElEntorno devuelve la dirección configurada.
func TestDireccionRespetaElEntorno(t *testing.T) {
	t.Setenv(envDireccion, "127.0.0.1:9090")

	if direccion() != "127.0.0.1:9090" {
		t.Errorf("direccion() = %q, want 127.0.0.1:9090", direccion())
	}

	t.Setenv(envDireccion, "")

	if direccion() != direccionProd {
		t.Errorf("direccion() = %q, want %q", direccion(), direccionProd)
	}
}

// TestRutasPorOmision comprueba la base, la clave y la llave.
func TestRutasPorOmision(t *testing.T) {
	t.Setenv(envDB, "")
	t.Setenv(envSSHKey, "")

	if ruta := rutaBaseDatos(); ruta != filepath.Join("data", "vmcons.db") {
		t.Errorf("rutaBaseDatos() = %q, want data/vmcons.db", ruta)
	}

	base := filepath.Join(t.TempDir(), "datos", "vmcons.db")
	t.Setenv(envDB, base)

	if ruta := rutaClaveCifrado(); ruta != filepath.Join(filepath.Dir(base), "credenciales.key") {
		t.Errorf("rutaClaveCifrado() = %q, want junto a la base", ruta)
	}

	t.Setenv(envSSHKey, "/tmp/llave")

	if ruta := rutaLlave(); ruta != "/tmp/llave" {
		t.Errorf("rutaLlave() = %q, want /tmp/llave", ruta)
	}

	t.Setenv(envSSHKey, "")

	if ruta := rutaLlave(); !strings.HasSuffix(ruta, filepath.Join("vmcons", "id_ed25519")) {
		t.Errorf("rutaLlave() = %q, want la llave en el directorio de configuración", ruta)
	}
}

// TestUsuarioInvitadoPorOmision devuelve root.
func TestUsuarioInvitadoPorOmision(t *testing.T) {
	t.Setenv(envGuestUser, "")

	if usuario := usuarioInvitado(); usuario != "root" {
		t.Errorf("usuarioInvitado() = %q, want root", usuario)
	}

	t.Setenv(envGuestUser, "jorge")

	if usuario := usuarioInvitado(); usuario != "jorge" {
		t.Errorf("usuarioInvitado() = %q, want jorge", usuario)
	}
}

// TestObjetivoInicialRestauraLaMaquina usa la máquina persistida.
func TestObjetivoInicialRestauraLaMaquina(t *testing.T) {
	directorio := t.TempDir()

	almacen, err := services.NuevoAlmacen(
		filepath.Join(directorio, "vmcons.db"), filepath.Join(directorio, "credenciales.key"))
	if err != nil {
		t.Fatalf("abrir el almacén: %v", err)
	}

	t.Setenv(envVM, "")

	if objetivo := objetivoInicial(almacen); objetivo.ID != "" {
		t.Errorf("objetivoInicial() = %q, want vacío", objetivo.ID)
	}

	if err := almacen.GuardarActiva("debian-1"); err != nil {
		t.Fatalf("guardar la máquina: %v", err)
	}

	if objetivo := objetivoInicial(almacen); objetivo.ID != "debian-1" {
		t.Errorf("objetivoInicial() = %q, want debian-1", objetivo.ID)
	}

	t.Setenv(envVM, "ubuntu-lab")

	if objetivo := objetivoInicial(almacen); objetivo.ID != "ubuntu-lab" {
		t.Errorf("objetivoInicial() = %q, want la variable de entorno", objetivo.ID)
	}
}

// TestColorHabilitadoRespetaNoColor comprueba la variable de entorno.
func TestColorHabilitadoRespetaNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	if colorHabilitado() {
		t.Error("colorHabilitado() = true con NO_COLOR definido")
	}
}

// TestHandlerConAtributosYGrupos comprueba que los atributos fijos y los
// del registro llegan al destino del log.
func TestHandlerConAtributosYGrupos(t *testing.T) {
	t.Parallel()

	var destino bytes.Buffer

	manejador := nuevoHandlerConsola(&destino, slog.LevelDebug, false).
		WithAttrs([]slog.Attr{slog.String("clave", "valor")}).
		WithGroup("grupo")

	slog.New(manejador).Info("mensaje", "otra", "dato")

	salida := destino.String()
	for _, esperado := range []string{"mensaje", "clave=valor", "otra=dato"} {
		if !strings.Contains(salida, esperado) {
			t.Errorf("log = %q, want %q", salida, esperado)
		}
	}
}

// TestNivelYEstadoPintanConColor cubre las ramas de color del formato.
func TestNivelYEstadoPintanConColor(t *testing.T) {
	t.Parallel()

	var destino bytes.Buffer

	manejador, ok := nuevoHandlerConsola(&destino, slog.LevelDebug, true).(*handlerConsola)
	if !ok {
		t.Fatalf("nuevoHandlerConsola() = %T, want *handlerConsola", nuevoHandlerConsola(&destino, slog.LevelDebug, true))
	}

	if !strings.Contains(manejador.nivelTexto(slog.LevelWarn), "WARN") {
		t.Error("nivelTexto(WARN) no contiene WARN")
	}

	if !strings.Contains(manejador.colorEstado(503, "503"), "503") {
		t.Error("colorEstado(503) no conserva el texto")
	}
}

// TestColorEnConDestinoQueNoEsArchivo devuelve false.
func TestColorEnConDestinoQueNoEsArchivo(t *testing.T) {
	t.Parallel()

	if colorEn(&bytes.Buffer{}) {
		t.Error("colorEn(buffer) = true, want false")
	}
}

// TestInterfazWebEmbebida usa la copia del binario.
func TestInterfazWebEmbebida(t *testing.T) {
	t.Setenv(envWeb, "")

	interfaz, err := interfazWeb()
	if err != nil {
		t.Fatalf("interfazWeb() error = %v, want nil", err)
	}

	if _, err := interfaz.Open("index.html"); err != nil {
		t.Errorf("Open(index.html) error = %v, want nil", err)
	}
}

// TestInterfazEnDiscoRechazaUnArchivo exige un directorio.
func TestInterfazEnDiscoRechazaUnArchivo(t *testing.T) {
	t.Parallel()

	ruta := filepath.Join(t.TempDir(), "index.html")
	if err := os.WriteFile(ruta, []byte("x"), 0o600); err != nil {
		t.Fatalf("crear el archivo: %v", err)
	}

	if _, err := interfazEnDisco(ruta); err == nil {
		t.Error("interfazEnDisco() error = nil, want el fallo de directorio")
	}
}

// entornoTemporal apunta la base y las llaves a un directorio temporal.
func entornoTemporal(t *testing.T) {
	t.Helper()

	directorio := t.TempDir()
	t.Setenv(envDB, filepath.Join(directorio, "vmcons.db"))
	t.Setenv(envSSHKey, filepath.Join(directorio, "id_ed25519"))
	t.Setenv(envWeb, "")
	t.Setenv(envVM, "")
}

// TestNuevoServidorArmaElServidor comprueba la dirección y los tiempos.
func TestNuevoServidorArmaElServidor(t *testing.T) {
	entornoTemporal(t)
	t.Setenv(envDireccion, "127.0.0.1:0")

	servidor, err := nuevoServidor()
	if err != nil {
		t.Fatalf("nuevoServidor() error = %v, want nil", err)
	}

	if servidor.Addr != "127.0.0.1:0" || servidor.Handler == nil {
		t.Errorf("nuevoServidor() = %+v, want dirección y manejador", servidor)
	}
}

// TestNuevoMuxProtegeLaAPI comprueba el ensamblaje completo.
func TestNuevoMuxProtegeLaAPI(t *testing.T) {
	entornoTemporal(t)

	manejador, err := nuevoMux()
	if err != nil {
		t.Fatalf("nuevoMux() error = %v, want nil", err)
	}

	rec := httptest.NewRecorder()
	manejador.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/api/auth/estado", http.NoBody))

	if rec.Code != http.StatusOK {
		t.Errorf("estado = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	manejador.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/api/packages", http.NoBody))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("packages = %d, want 401 sin sesión", rec.Code)
	}
}

// TestNuevaConsolaArmaLosServicios comprueba la inyección de dependencias.
func TestNuevaConsolaArmaLosServicios(t *testing.T) {
	entornoTemporal(t)

	consola := nuevaConsola(localproc.New())
	if consola.sesiones == nil || consola.acceso == nil || consola.registro == nil {
		t.Errorf("nuevaConsola() = %+v, want los servicios armados", consola)
	}
}

// TestEjecutarFallConDireccionInvalida devuelve el error del servidor.
func TestEjecutarFallConDireccionInvalida(t *testing.T) {
	entornoTemporal(t)
	t.Setenv(envDireccion, "256.256.256.256:99999")
	t.Setenv("NO_COLOR", "1")

	if err := ejecutar(); err == nil {
		t.Error("ejecutar() error = nil, want el fallo de la dirección")
	}
}

// TestImprimirBannerEscribeLaURL captura la salida del banner.
func TestImprimirBannerEscribeLaURL(t *testing.T) {
	lectura, escritura, err := os.Pipe()
	if err != nil {
		t.Fatalf("crear la tubería: %v", err)
	}

	anterior := os.Stdout

	os.Stdout = escritura

	t.Setenv("NO_COLOR", "1")

	fallo := imprimirBanner("127.0.0.1:8080")

	os.Stdout = anterior

	if err := escritura.Close(); err != nil {
		t.Fatalf("cerrar la tubería: %v", err)
	}

	salida, err := io.ReadAll(lectura)
	if err != nil {
		t.Fatalf("leer la salida: %v", err)
	}

	if fallo != nil {
		t.Fatalf("imprimirBanner() error = %v, want nil", fallo)
	}

	if !strings.Contains(string(salida), "http://127.0.0.1:8080") {
		t.Errorf("banner = %q, want la URL de la consola", salida)
	}
}
