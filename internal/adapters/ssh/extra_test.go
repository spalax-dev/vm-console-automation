// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package ssh

import (
	"context"
	"errors"
	"io/fs"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"consola-vm/internal/core/domain"
)

// archivoFalso simula un archivo temporal con fallos preprogramados.
type archivoFalso struct {
	chmodErr error
	writeErr error
	closeErr error
	nombre   string
}

// Chmod devuelve el error preprogramado.
func (a *archivoFalso) Chmod(fs.FileMode) error { return a.chmodErr }

// WriteString devuelve el error preprogramado.
func (a *archivoFalso) WriteString(string) (int, error) { return 0, a.writeErr }

// Name devuelve la ruta asociada.
func (a *archivoFalso) Name() string { return a.nombre }

// Close devuelve el error preprogramado.
func (a *archivoFalso) Close() error { return a.closeErr }

// archivoEnBlanco devuelve la ruta de un archivo temporal del test.
func archivoEnBlanco(t *testing.T) string {
	t.Helper()

	ruta := filepath.Join(t.TempDir(), "pass")
	if err := os.WriteFile(ruta, nil, 0o600); err != nil {
		t.Fatalf("crear el archivo temporal: %v", err)
	}

	return ruta
}

// TestPrepararTemporalErrores deja el sistema limpio ante cada fallo.
func TestPrepararTemporalErrores(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre  string
		archivo *archivoFalso
		mensaje string
	}{
		{
			"chmod",
			&archivoFalso{chmodErr: errors.New("permisos"), nombre: archivoEnBlanco(t)},
			"proteger",
		},
		{
			"escritura",
			&archivoFalso{writeErr: errors.New("disco lleno"), nombre: archivoEnBlanco(t)},
			"guardar",
		},
		{
			"cierre",
			&archivoFalso{closeErr: errors.New("cerrado"), nombre: archivoEnBlanco(t)},
			"cerrar",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			err := prepararTemporal(caso.archivo, "secreto")
			if err == nil || !strings.Contains(err.Error(), caso.mensaje) {
				t.Fatalf("prepararTemporal() error = %v, want el fallo de %s", err, caso.mensaje)
			}

			if _, err := os.Stat(caso.archivo.nombre); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("el archivo temporal %q quedó en disco", caso.archivo.nombre)
			}
		})
	}
}

// TestPrepararTemporalGuardaLaContraseña escribe el archivo con la clave.
func TestPrepararTemporalGuardaLaContraseña(t *testing.T) {
	t.Parallel()

	archivo, err := os.CreateTemp(t.TempDir(), "pass-*")
	if err != nil {
		t.Fatalf("crear el archivo temporal: %v", err)
	}

	if err := prepararTemporal(archivo, "secreto"); err != nil {
		t.Fatalf("prepararTemporal() error = %v, want nil", err)
	}

	contenido, err := os.ReadFile(archivo.Name())
	if err != nil || string(contenido) != "secreto" {
		t.Errorf("contenido = %q, %v, want secreto", contenido, err)
	}

	info, err := os.Stat(archivo.Name())
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("permisos = %v, %v, want 0600", info.Mode().Perm(), err)
	}
}

// TestConnectPasswordSinArchivoTemporal propaga el fallo de creación.
func TestConnectPasswordSinArchivoTemporal(t *testing.T) {
	original := crearTemporal

	t.Cleanup(func() { crearTemporal = original })

	crearTemporal = func() (*os.File, error) {
		return nil, errors.New("sin espacio")
	}

	conector := New(newFakeExecutor())

	if _, err := conector.ConnectPassword(context.Background(), runningTarget(), "jorge", "x"); err == nil {
		t.Error("ConnectPassword() error = nil, want el fallo del archivo temporal")
	}
}

// TestConnectPasswordCreaYProtegeElArchivo abre una sesión con sshpass y
// borra la contraseña al cerrar.
func TestConnectPasswordCreaYProtegeElArchivo(t *testing.T) {
	t.Parallel()

	sesion, err := New(newFakeExecutor()).ConnectPassword(context.Background(), runningTarget(), "jorge", "secreto")
	if err != nil {
		t.Fatalf("ConnectPassword() error = %v, want nil", err)
	}

	concreta, ok := sesion.(*Session)
	if !ok {
		t.Fatalf("ConnectPassword() = %T, want *Session", sesion)
	}

	info, err := os.Stat(concreta.passwordFile)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("permisos del temporal = %v, %v, want 0600", info.Mode().Perm(), err)
	}

	if err := sesion.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}

	if _, err := os.Stat(concreta.passwordFile); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("el archivo temporal %q no se eliminó", concreta.passwordFile)
	}
}

// TestCerrarTemporalConFalloDeCierre elimina el archivo igualmente.
func TestCerrarTemporalConFalloDeCierre(t *testing.T) {
	t.Parallel()

	ruta := archivoEnBlanco(t)
	cerrarTemporal(&archivoFalso{closeErr: errors.New("cerrado"), nombre: ruta})

	if _, err := os.Stat(ruta); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("el archivo temporal %q no se eliminó", ruta)
	}
}

// TestEliminarTemporalIgnoraLaAusencia no falla si el archivo ya no está.
func TestEliminarTemporalIgnoraLaAusencia(t *testing.T) {
	t.Parallel()

	eliminarTemporal(filepath.Join(t.TempDir(), "inexistente"))
}

// TestProbeConEjecutorCaido propaga el fallo de autenticación.
func TestProbeConEjecutorCaido(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.errs = map[string]error{binarySSH: errors.New("sin conexión")}

	conector := New(fake, WithKey(writeTempKey(t)))

	if err := conector.Probe(context.Background(), runningTarget()); err == nil {
		t.Error("Probe() error = nil, want el fallo de ssh")
	}
}

// TestConnectConProbeFallido no abre sesión sin autenticación.
func TestConnectConProbeFallido(t *testing.T) {
	t.Parallel()

	conector := New(newFakeExecutor(), WithKey(writeTempKey(t)))

	if _, err := conector.Connect(context.Background(), domain.Target{ID: "debian13"}); err == nil {
		t.Error("Connect() error = nil, want el fallo de Probe")
	}
}

// TestConnectAbreLaSesion con una llave disponible.
func TestConnectAbreLaSesion(t *testing.T) {
	t.Parallel()

	conector := New(newFakeExecutor(), WithKey(writeTempKey(t)))

	sesion, err := conector.Connect(context.Background(), runningTarget())
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	if sesion == nil {
		t.Fatal("Connect() = nil, want una sesión")
	}
}

// TestPutPropagaElFalloDeSCP devuelve el error de la transferencia.
func TestPutPropagaElFalloDeSCP(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.errs = map[string]error{binarySCP: &domain.ExitError{Code: 1, Command: "scp"}}

	sesion, err := New(fake, WithKey(writeTempKey(t))).Connect(context.Background(), runningTarget())
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	if err := sesion.Put(context.Background(), "/tmp/a", "/tmp/b"); err == nil {
		t.Error("Put() error = nil, want el fallo de scp")
	}
}

// fakeConFalloAutentica la primera vez y falla en las siguientes órdenes.
type fakeConFallo struct {
	llamadas int
}

// Execute autentica la primera orden ssh y falla en las posteriores.
func (f *fakeConFallo) Execute(_ context.Context, program string, _ ...string) (domain.Result, error) {
	if program != binarySSH {
		return domain.Result{}, nil
	}

	f.llamadas++
	if f.llamadas == 1 {
		return domain.Result{}, nil
	}

	return domain.Result{}, &domain.ExitError{Code: 1, Command: binarySSH}
}

// TestPutPropagaElFalloDeChmod devuelve el error al ajustar los permisos.
func TestPutPropagaElFalloDeChmod(t *testing.T) {
	t.Parallel()

	sesion, err := New(&fakeConFallo{}, WithKey(writeTempKey(t))).Connect(context.Background(), runningTarget())
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	if err := sesion.Put(context.Background(), "/tmp/a", "/tmp/b"); err == nil {
		t.Error("Put() error = nil, want el fallo de chmod")
	}
}

// TestCloseSinArchivoTemporal no hace nada en una sesión con llave.
func TestCloseSinArchivoTemporal(t *testing.T) {
	t.Parallel()

	if err := (&Session{}).Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

// TestCloseConArchivoAusente devuelve el error al eliminar.
func TestCloseConArchivoAusente(t *testing.T) {
	t.Parallel()

	sesion := &Session{passwordFile: filepath.Join(t.TempDir(), "inexistente")}

	if err := sesion.Close(); err == nil {
		t.Error("Close() error = nil, want el fallo al eliminar")
	}
}

// TestErrorSanitizadoConservaLosDemasErrores deja intacto un error que no
// proviene de un código de salida.
func TestErrorSanitizadoConservaLosDemasErrores(t *testing.T) {
	t.Parallel()

	original := errors.New("sin conexión")

	if err := errorSanitizado(original, "ssh root@10.0.2.15"); !errors.Is(err, original) {
		t.Errorf("errorSanitizado() = %v, want el error original", err)
	}
}

// TestProbeConDireccionValidaYLLave comprueba el camino correcto.
func TestProbeConDireccionValidaYLLave(t *testing.T) {
	t.Parallel()

	conector := New(newFakeExecutor(), WithKey(writeTempKey(t)))

	if err := conector.Probe(context.Background(), domain.Target{
		ID:      "debian13",
		Address: netip.MustParseAddr("10.0.2.15"),
	}); err != nil {
		t.Errorf("Probe() error = %v, want nil", err)
	}
}

// TestConnectPasswordConArchivoNoEscribible propaga el fallo al guardar la
// contraseña.
func TestConnectPasswordConArchivoNoEscribible(t *testing.T) {
	original := crearTemporal

	t.Cleanup(func() { crearTemporal = original })

	ruta := filepath.Join(t.TempDir(), "pass")
	if err := os.WriteFile(ruta, nil, 0o600); err != nil {
		t.Fatalf("crear el archivo: %v", err)
	}

	if err := os.Chmod(ruta, 0o400); err != nil {
		t.Fatalf("quitar la escritura: %v", err)
	}

	crearTemporal = func() (*os.File, error) {
		return os.Open(ruta) // #nosec G304 -- ruta de la prueba.
	}

	if _, err := New(newFakeExecutor()).ConnectPassword(
		context.Background(), runningTarget(), "jorge", "secreto",
	); err == nil {
		t.Error("ConnectPassword() error = nil, want el fallo al escribir la contraseña")
	}
}

// TestEliminarTemporalConPermisosDenegados solo advierte cuando no puede
// borrar el archivo.
func TestEliminarTemporalConPermisosDenegados(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("ejecutando como root: los permisos no impiden borrar")
	}

	directorio := t.TempDir()
	ruta := filepath.Join(directorio, "pass")

	if err := os.WriteFile(ruta, nil, 0o600); err != nil {
		t.Fatalf("crear el archivo: %v", err)
	}

	if err := os.Chmod(directorio, 0o500); err != nil { // #nosec G302 -- permisos de la prueba.
		t.Fatalf("restringir el directorio: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chmod(directorio, 0o700); err != nil { // #nosec G302 -- permisos de la prueba.
			t.Logf("restaurar permisos: %v", err)
		}
	})

	eliminarTemporal(ruta)
}
