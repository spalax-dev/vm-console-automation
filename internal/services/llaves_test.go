// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"consola-vm/internal/core/domain"
)

const (
	// clavePublica simula la llave pública que genera ssh-keygen.
	keyPublic = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI ejemplo vmcons-root"
	// nombreLlave es el nombre por omisión de la llave.
	nameSSHKey = "id_ed25519"
	// contenidoPrivado simula el contenido de la llave privada.
	contentPrivate = "-----BEGIN OPENSSH PRIVATE KEY-----\nfalso\n-----END OPENSSH PRIVATE KEY-----\n"
)

// stubExecutor registra las órdenes del anfitrión y permite simular sus
// efectos secundarios con onExecute.
type stubExecutor struct {
	calls     [][]string
	result    domain.Result
	err       error
	onExecute func()
}

// Execute registra la orden y devuelve la respuesta preprogramada.
func (e *stubExecutor) Execute(_ context.Context, program string, args ...string) (domain.Result, error) {
	e.calls = append(e.calls, append([]string{program}, args...))

	if e.onExecute != nil {
		e.onExecute()
	}

	return e.result, e.err
}

// newKeysForTest arma el servicio con un ejecutor y una sesión
// falsos sobre una ruta temporal.
func newKeysForTest(t *testing.T, ejecutor *stubExecutor, session *stubSession, openErr error) (*Keys, *stubOpener, string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), nameSSHKey)
	opener := &stubOpener{session: session, err: openErr}

	return NewKeys(KeysConfig{
		Sessions: opener,
		Ejecutor: ejecutor,
		Path:     path,
	}), opener, path
}

// TestKeysGenerateInstallsKey comprueba la generación, instalación y
// configuración de sshd.
func TestKeysGenerateInstallsKey(t *testing.T) {
	t.Parallel()

	ejecutor := &stubExecutor{}
	session := &stubSession{}

	keys, opener, path := newKeysForTest(t, ejecutor, session, nil)
	ejecutor.onExecute = func() {
		if err := os.WriteFile(path, []byte(contentPrivate), 0o600); err != nil {
			t.Fatalf("escribir la llave privada: %v", err)
		}

		if err := os.WriteFile(path+".pub", []byte(keyPublic+"\n"), 0o600); err != nil {
			t.Fatalf(msgWritePub, err)
		}
	}

	result, err := keys.Generate(context.Background())
	if err != nil {
		t.Fatalf("Generar() error = %v, want nil", err)
	}

	if opener.open != 1 {
		t.Errorf(msgOpen, opener.open)
	}

	if len(ejecutor.calls) != 1 {
		t.Fatalf("ssh-keygen se ejecutó %d veces, want 1", len(ejecutor.calls))
	}

	llamada := ejecutor.calls[0]
	if llamada[0] != "ssh-keygen" {
		t.Errorf("programa = %q, want ssh-keygen", llamada[0])
	}

	if !contiene(llamada, "-t") || !contiene(llamada, "ed25519") || !contiene(llamada, path) {
		t.Errorf("argumentos = %v, want tipo ed25519 y ruta %q", llamada, path)
	}

	if len(session.commands) != 1 {
		t.Fatalf(msgCommandUnica, len(session.commands))
	}

	command := session.commands[0].ShellLine()
	if !strings.Contains(command, "authorized_keys") || !strings.Contains(command, "AAAAC3NzaC1lZDI1NTE5") {
		t.Errorf("orden de authorized_keys = %q", command)
	}

	if !strings.Contains(command, "sshd_config.d/99-vmcons.conf") ||
		!strings.Contains(command, "PasswordAuthentication no") {
		t.Errorf("orden de sshd = %q", command)
	}

	if !strings.Contains(command, "openssh-server") || !strings.Contains(command, "enable --now ssh") {
		t.Errorf("orden = %q, want instalar y arrancar SSH", command)
	}

	if !strings.Contains(command, "systemctl reload ssh") || !strings.Contains(command, "systemctl restart ssh") {
		t.Errorf("orden = %q, want recargar la configuración de SSH", command)
	}

	if !strings.Contains(command, "sshd -T") || !strings.Contains(command, "passwordauthentication no") {
		t.Errorf("orden = %q, want verificar la configuración efectiva de SSH", command)
	}

	if !session.closed {
		t.Error(msgSessionOpen)
	}

	if result.Code != 0 {
		t.Errorf(msgCodeCero, result.Code)
	}

	if result.Message == "" {
		t.Error("Mensaje vacío, want el mensaje de éxito")
	}
}

// contiene informa si la lista incluye el valor.
func contiene(values []string, buscado string) bool {
	for _, value := range values {
		if value == buscado {
			return true
		}
	}

	return false
}

// TestKeysGenerateExisting comprueba que no se sobrescribe una llave
// existente.
func TestKeysGenerateExisting(t *testing.T) {
	t.Parallel()

	ejecutor := &stubExecutor{}
	keys, opener, path := newKeysForTest(t, ejecutor, &stubSession{}, nil)

	if err := os.WriteFile(path, []byte(contentPrivate), 0o600); err != nil {
		t.Fatalf("escribir la llave existente: %v", err)
	}

	if _, err := keys.Generate(context.Background()); !errors.Is(err, ErrKeyExists) {
		t.Fatalf("Generar() error = %v, want ErrLlaveExistente", err)
	}

	if len(ejecutor.calls) != 0 {
		t.Error("ssh-keygen se ejecutó con una llave existente")
	}

	if opener.open != 0 {
		t.Error("se abrió una sesión con una llave existente")
	}
}

// TestKeysGenerateErrorOfKeygen comprueba que el fallo de ssh-keygen
// se propaga.
func TestKeysGenerateErrorOfKeygen(t *testing.T) {
	t.Parallel()

	ejecutor := &stubExecutor{err: &domain.ExitError{Code: 1, Command: "ssh-keygen"}}
	session := &stubSession{}

	keys, _, _ := newKeysForTest(t, ejecutor, session, nil)

	if _, err := keys.Generate(context.Background()); err == nil {
		t.Fatal("Generar() error = nil, want el error de ssh-keygen")
	}

	if !session.closed {
		t.Error(msgSessionOpen)
	}
}

// TestKeysGenerateWithoutSession comprueba que se propaga la ausencia de
// sesión.
func TestKeysGenerateWithoutSession(t *testing.T) {
	t.Parallel()

	keys, _, _ := newKeysForTest(t, &stubExecutor{}, nil, ErrSessionUnavailable)

	if _, err := keys.Generate(context.Background()); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("Generar() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestKeysGenerateErrorOfInstallation comprueba que el fallo al instalar
// la clave conserva el detalle técnico.
func TestKeysGenerateErrorOfInstallation(t *testing.T) {
	t.Parallel()

	ejecutor := &stubExecutor{}
	session := &stubSession{
		result: domain.Result{Stderr: "Permission denied", ExitCode: 1},
		err:    &domain.ExitError{Code: 1, Command: "sh"},
	}

	keys, _, path := newKeysForTest(t, ejecutor, session, nil)
	ejecutor.onExecute = func() {
		if err := os.WriteFile(path+".pub", []byte(keyPublic+"\n"), 0o600); err != nil {
			t.Fatalf(msgWritePub, err)
		}
	}

	result, err := keys.Generate(context.Background())
	if err == nil {
		t.Fatal("Generar() error = nil, want el error de instalación")
	}

	if result.Code != 1 {
		t.Errorf("Codigo = %d, want 1", result.Code)
	}

	if result.Stderr != "Permission denied" {
		t.Errorf(msgStderrSession, result.Stderr)
	}
}

// TestKeysDownload devuelve el contenido de la llave privada.
func TestKeysDownload(t *testing.T) {
	t.Parallel()

	keys, _, path := newKeysForTest(t, &stubExecutor{}, &stubSession{}, nil)

	if err := os.WriteFile(path, []byte(contentPrivate), 0o600); err != nil {
		t.Fatalf("escribir la llave: %v", err)
	}

	name, content, err := keys.Download()
	if err != nil {
		t.Fatalf("Descargar() error = %v, want nil", err)
	}

	if name != nameSSHKey {
		t.Errorf("nombre = %q, want id_ed25519", name)
	}

	if string(content) != contentPrivate {
		t.Errorf("contenido = %q, want %q", string(content), contentPrivate)
	}
}

// TestKeysDownloadInexistente devuelve ErrKeyNotFound.
func TestKeysDownloadInexistente(t *testing.T) {
	t.Parallel()

	keys, _, _ := newKeysForTest(t, &stubExecutor{}, &stubSession{}, nil)

	if _, _, err := keys.Download(); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("Descargar() error = %v, want ErrLlaveNoEncontrada", err)
	}
}

// TestKeysGenerateStatError comprueba el fallo al comprobar la ruta de
// la llave.
func TestKeysGenerateStatError(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "archivo")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf(msgPrepareFile, err)
	}

	keys := NewKeys(KeysConfig{
		Sessions: &stubOpener{},
		Ejecutor: &stubExecutor{},
		Path:     filepath.Join(file, "id_ed25519"),
	})

	if _, err := keys.Generate(context.Background()); err == nil {
		t.Fatal("Generar() error = nil, want el error al comprobar la ruta")
	}
}

// TestKeysGenerateErrorOfDirectory comprueba el fallo al crear el
// directorio de llaves cuando el padre es un archivo.
func TestKeysGenerateErrorOfDirectory(t *testing.T) {
	t.Parallel()

	file := fileWithText(t)

	keys := NewKeys(KeysConfig{
		Sessions: &stubOpener{},
		Ejecutor: &stubExecutor{},
		Path:     filepath.Join(file, "sub", "id_ed25519"),
	})

	if _, err := keys.Generate(context.Background()); err == nil {
		t.Fatal("Generar() error = nil, want el error al crear el directorio")
	}
}

// TestKeysGenerateWithoutPublic comprueba el fallo al leer la llave
// pública.
func TestKeysGenerateWithoutPublic(t *testing.T) {
	t.Parallel()

	keys, _, _ := newKeysForTest(t, &stubExecutor{}, &stubSession{}, nil)

	if _, err := keys.Generate(context.Background()); err == nil {
		t.Fatal("Generar() error = nil, want el error al leer la llave pública")
	}
}

// TestKeysDownloadErrorOfRead comprueba el fallo de lectura de la
// llave privada.
func TestKeysDownloadErrorOfRead(t *testing.T) {
	t.Parallel()

	keys, _, path := newKeysForTest(t, &stubExecutor{}, &stubSession{}, nil)

	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf(msgCreateDir, err)
	}

	if _, _, err := keys.Download(); err == nil {
		t.Fatal("Descargar() error = nil, want el error de lectura")
	}
}
