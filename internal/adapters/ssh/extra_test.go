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

// fileFake simula un archivo temporal con fallos preprogramados.
type fileFake struct {
	chmodErr error
	writeErr error
	closeErr error
	name     string
}

// Chmod devuelve el error preprogramado.
func (a *fileFake) Chmod(fs.FileMode) error { return a.chmodErr }

// WriteString devuelve el error preprogramado.
func (a *fileFake) WriteString(string) (int, error) { return 0, a.writeErr }

// Name devuelve la ruta asociada.
func (a *fileFake) Name() string { return a.name }

// Close devuelve el error preprogramado.
func (a *fileFake) Close() error { return a.closeErr }

// blankFile devuelve la ruta de un archivo temporal del test.
func blankFile(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "pass")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("crear el archivo temporal: %v", err)
	}

	return path
}

// TestPrepareTempErrors deja el sistema limpio ante cada fallo.
func TestPrepareTempErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		file    *fileFake
		message string
	}{
		{
			"chmod",
			&fileFake{chmodErr: errors.New("permisos"), name: blankFile(t)},
			"proteger",
		},
		{
			"escritura",
			&fileFake{writeErr: errors.New("disco lleno"), name: blankFile(t)},
			"guardar",
		},
		{
			"cierre",
			&fileFake{closeErr: errors.New("cerrado"), name: blankFile(t)},
			"cerrar",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := prepareTempFile(testCase.file, "secreto")
			if err == nil || !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("prepararTemporal() error = %v, want el fallo de %s", err, testCase.message)
			}

			if _, err := os.Stat(testCase.file.name); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("el archivo temporal %q quedó en disco", testCase.file.name)
			}
		})
	}
}

// TestPrepareTempSavesPassword escribe el archivo con la clave.
func TestPrepareTempSavesPassword(t *testing.T) {
	t.Parallel()

	file, err := os.CreateTemp(t.TempDir(), "pass-*")
	if err != nil {
		t.Fatalf("crear el archivo temporal: %v", err)
	}

	if err := prepareTempFile(file, "secreto"); err != nil {
		t.Fatalf("prepararTemporal() error = %v, want nil", err)
	}

	content, err := os.ReadFile(file.Name())
	if err != nil || string(content) != "secreto" {
		t.Errorf("contenido = %q, %v, want secreto", content, err)
	}

	info, err := os.Stat(file.Name())
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("permisos = %v, %v, want 0600", info.Mode().Perm(), err)
	}
}

// TestConnectPasswordWithoutFileTemp propaga el fallo de creación.
func TestConnectPasswordWithoutFileTemp(t *testing.T) {
	original := createTempFile

	t.Cleanup(func() { createTempFile = original })

	createTempFile = func() (*os.File, error) {
		return nil, errors.New("sin espacio")
	}

	connector := New(newFakeExecutor())

	if _, err := connector.ConnectPassword(context.Background(), runningTarget(), "jorge", "x"); err == nil {
		t.Error("ConnectPassword() error = nil, want el fallo del archivo temporal")
	}
}

// TestConnectPasswordCreatesYProtegeFile abre una sesión con sshpass y
// borra la contraseña al cerrar.
func TestConnectPasswordCreatesYProtegeFile(t *testing.T) {
	t.Parallel()

	session, err := New(newFakeExecutor()).ConnectPassword(context.Background(), runningTarget(), "jorge", "secreto")
	if err != nil {
		t.Fatalf("ConnectPassword() error = %v, want nil", err)
	}

	concreta, ok := session.(*Session)
	if !ok {
		t.Fatalf("ConnectPassword() = %T, want *Session", session)
	}

	info, err := os.Stat(concreta.passwordFile)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("permisos del temporal = %v, %v, want 0600", info.Mode().Perm(), err)
	}

	if err := session.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}

	if _, err := os.Stat(concreta.passwordFile); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("el archivo temporal %q no se eliminó", concreta.passwordFile)
	}
}

// TestCloseTempWithFailureOfCierre elimina el archivo igualmente.
func TestCloseTempWithFailureOfCierre(t *testing.T) {
	t.Parallel()

	path := blankFile(t)
	closeTempFile(&fileFake{closeErr: errors.New("cerrado"), name: path})

	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("el archivo temporal %q no se eliminó", path)
	}
}

// TestDeleteTempIgnoresAbsence no falla si el archivo ya no está.
func TestDeleteTempIgnoresAbsence(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "inexistente")
	removeTempFile(path)

	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Stat(%q) error = %v, want la ausencia del archivo", path, err)
	}
}

// TestProbeWithEjecutorCaido propaga el fallo de autenticación.
func TestProbeWithEjecutorCaido(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.errs = map[string]error{binarySSH: errors.New("sin conexión")}

	connector := New(fake, WithKey(writeTempKey(t)))

	if err := connector.Probe(context.Background(), runningTarget()); err == nil {
		t.Error("Probe() error = nil, want el fallo de ssh")
	}
}

// TestConnectWithProbeFailed no abre sesión sin autenticación.
func TestConnectWithProbeFailed(t *testing.T) {
	t.Parallel()

	connector := New(newFakeExecutor(), WithKey(writeTempKey(t)))

	if _, err := connector.Connect(context.Background(), domain.Target{ID: "vm1"}); err == nil {
		t.Error("Connect() error = nil, want el fallo de Probe")
	}
}

// TestConnectOpensSession con una llave disponible.
func TestConnectOpensSession(t *testing.T) {
	t.Parallel()

	connector := New(newFakeExecutor(), WithKey(writeTempKey(t)))

	session, err := connector.Connect(context.Background(), runningTarget())
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	if session == nil {
		t.Fatal("Connect() = nil, want una sesión")
	}
}

// TestPutPropagatesFailureOfSCP devuelve el error de la transferencia.
func TestPutPropagatesFailureOfSCP(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.errs = map[string]error{binarySCP: &domain.ExitError{Code: 1, Command: "scp"}}

	session, err := New(fake, WithKey(writeTempKey(t))).Connect(context.Background(), runningTarget())
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	if err := session.Put(context.Background(), "/tmp/a", "/tmp/b"); err == nil {
		t.Error("Put() error = nil, want el fallo de scp")
	}
}

// fakeConFalloAutentica la primera vez y falla en las siguientes órdenes.
type fakeWithFailure struct {
	llamadas int
}

// Execute autentica la primera orden ssh y falla en las posteriores.
func (f *fakeWithFailure) Execute(_ context.Context, program string, _ ...string) (domain.Result, error) {
	if program != binarySSH {
		return domain.Result{}, nil
	}

	f.llamadas++
	if f.llamadas == 1 {
		return domain.Result{}, nil
	}

	return domain.Result{}, &domain.ExitError{Code: 1, Command: binarySSH}
}

// TestPutPropagatesFailureOfChmod devuelve el error al ajustar los permisos.
func TestPutPropagatesFailureOfChmod(t *testing.T) {
	t.Parallel()

	session, err := New(&fakeWithFailure{}, WithKey(writeTempKey(t))).Connect(context.Background(), runningTarget())
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	if err := session.Put(context.Background(), "/tmp/a", "/tmp/b"); err == nil {
		t.Error("Put() error = nil, want el fallo de chmod")
	}
}

// TestCloseWithoutFileTemp no hace nada en una sesión con llave.
func TestCloseWithoutFileTemp(t *testing.T) {
	t.Parallel()

	if err := (&Session{}).Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

// TestCloseWithFileAusente devuelve el error al eliminar.
func TestCloseWithFileAusente(t *testing.T) {
	t.Parallel()

	session := &Session{passwordFile: filepath.Join(t.TempDir(), "inexistente")}

	if err := session.Close(); err == nil {
		t.Error("Close() error = nil, want el fallo al eliminar")
	}
}

// TestErrorSanitizedKeepsOtherErrors deja intacto un error que no
// proviene de un código de salida.
func TestErrorSanitizedKeepsOtherErrors(t *testing.T) {
	t.Parallel()

	original := errors.New("sin conexión")

	if err := sanitizeError(original, "ssh root@10.0.2.15"); !errors.Is(err, original) {
		t.Errorf("errorSanitizado() = %v, want el error original", err)
	}
}

// TestProbeWithAddressValidatesYLLave comprueba el camino correcto.
func TestProbeWithAddressValidatesYLLave(t *testing.T) {
	t.Parallel()

	connector := New(newFakeExecutor(), WithKey(writeTempKey(t)))

	if err := connector.Probe(context.Background(), domain.Target{
		ID:      "vm1",
		Address: netip.MustParseAddr("10.0.2.15"),
	}); err != nil {
		t.Errorf("Probe() error = %v, want nil", err)
	}
}

// TestConnectPasswordWithFileNotEscribible propaga el fallo al guardar la
// contraseña.
func TestConnectPasswordWithFileNotEscribible(t *testing.T) {
	original := createTempFile

	t.Cleanup(func() { createTempFile = original })

	path := filepath.Join(t.TempDir(), "pass")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("crear el archivo: %v", err)
	}

	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatalf("quitar la escritura: %v", err)
	}

	createTempFile = func() (*os.File, error) {
		return os.Open(path) // #nosec G304 -- ruta de la prueba.
	}

	if _, err := New(newFakeExecutor()).ConnectPassword(
		context.Background(), runningTarget(), "jorge", "secreto",
	); err == nil {
		t.Error("ConnectPassword() error = nil, want el fallo al escribir la contraseña")
	}
}

// TestDeleteTempWithFailureOfBorrado conserva el archivo cuando el
// sistema no permite eliminarlo, sin interrumpir el flujo.
func TestDeleteTempWithFailureOfBorrado(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "pass")

	// Un directorio no vacío hace que os.Remove falle de forma
	// determinista, aunque la prueba se ejecute como root.
	if err := os.MkdirAll(filepath.Join(path, "contenido"), 0o700); err != nil {
		t.Fatalf("crear el directorio: %v", err)
	}

	if err := os.WriteFile(filepath.Join(path, "contenido", "archivo"), []byte("x"), 0o600); err != nil {
		t.Fatalf("crear el archivo interno: %v", err)
	}

	removeTempFile(path)

	if _, err := os.Stat(path); err != nil {
		t.Errorf("el archivo que no se pudo borrar desapareció: %v", err)
	}
}
