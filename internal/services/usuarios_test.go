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

// outputPasswd simula getent passwd con cuentas del sistema y regulares.
const outputPasswd = "root:x:0:0:root:/root:/bin/bash\n" +
	"jorge:x:1000:1000:Jorge:/home/jorge:/bin/bash\n" +
	"systemd-network:x:998:998::/:/usr/sbin/nologin\n" +
	"andrey:x:1001:1001::/home/andrey:/bin/bash\n"

// outputSudo simula getent group sudo.
const outputSudo = "sudo:x:27:jorge\n"

// newUsersForTest arma el servicio con sesión y ejecutor falsos.
func newUsersForTest(t *testing.T, ejecutor *stubExecutor, session *stubSession, err error) (*Users, *stubOpener, string) {
	t.Helper()

	directory := t.TempDir()
	opener := &stubOpener{session: session, err: err}

	return NewUsers(UsersConfig{
		Sessions:  opener,
		Ejecutor:  ejecutor,
		Directory: directory,
	}), opener, directory
}

// TestUsersListFiltraAccountsRegulares comprueba el listado y las
// marcas de sudo y llaves.
func TestUsersListFiltraAccountsRegulares(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{
		{Stdout: outputPasswd},
		{Stdout: outputSudo},
	}}

	users, _, directory := newUsersForTest(t, &stubExecutor{}, session, nil)

	if err := os.WriteFile(filepath.Join(directory, "andrey_ed25519"), []byte(contentPrivate), 0o600); err != nil {
		t.Fatalf("crear la llave de andrey: %v", err)
	}

	lists, err := users.List(context.Background())
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	if len(lists) != 2 {
		t.Fatalf("Listar() devolvió %d usuarios, want 2", len(lists))
	}

	if lists[0].Name != nameJorge || !lists[0].Sudo || lists[0].UID != 1000 {
		t.Errorf("primer usuario = %+v, want jorge con sudo y UID 1000", lists[0])
	}

	if lists[1].Name != userAndrey || !lists[1].Keys {
		t.Errorf("segundo usuario = %+v, want andrey con llaves", lists[1])
	}
}

// TestUsersGet comprueba la consulta de una cuenta.
func TestUsersGet(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{
		{Stdout: "jorge:x:1000:1000:Jorge:/home/jorge:/bin/bash\n"},
		{Stdout: outputSudo},
	}}

	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	user, err := users.Get(context.Background(), nameJorge)
	if err != nil {
		t.Fatalf(msgGetOK, err)
	}

	if user.Name != nameJorge || !user.Sudo || user.Home != "/home/jorge" {
		t.Errorf("usuario = %+v, want jorge con sudo y home", user)
	}
}

// TestUsersGetInexistente comprueba el error de una cuenta que no
// existe.
func TestUsersGetInexistente(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{{Stdout: ""}}}

	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	if _, err := users.Get(context.Background(), nameJorge); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Obtener() error = %v, want ErrUsuarioNoEncontrado", err)
	}
}

// TestUsersCreateWithKeys comprueba la creación con sudo y llaves.
func TestUsersCreateWithKeys(t *testing.T) {
	t.Parallel()

	ejecutor := &stubExecutor{}
	session := &stubSession{}

	users, opener, directory := newUsersForTest(t, ejecutor, session, nil)
	ejecutor.onExecute = func() {
		if err := os.WriteFile(filepath.Join(directory, "jorge_ed25519.pub"), []byte(keyPublic+"\n"), 0o600); err != nil {
			t.Fatalf(msgWritePub, err)
		}
	}

	result, err := users.Create(context.Background(), nameJorge, true, true)
	if err != nil {
		t.Fatalf("Crear() error = %v, want nil", err)
	}

	if opener.open != 1 {
		t.Errorf(msgOpen, opener.open)
	}

	if len(session.commands) != 3 {
		t.Fatalf("la sesión ejecutó %d órdenes, want 3", len(session.commands))
	}

	creacion := session.commands[1].ShellLine()
	for _, expected := range []string{binUseradd, "-aG sudo", binPasswd + " -l"} {
		if !strings.Contains(creacion, expected) {
			t.Errorf("orden de creación = %q, want %q", creacion, expected)
		}
	}

	installation := session.commands[2].ShellLine()
	if !strings.Contains(installation, "authorized_keys") || !strings.Contains(installation, keyPublic) {
		t.Errorf("orden de llaves = %q, want authorized_keys con la pública", installation)
	}

	if !session.closed {
		t.Error(msgSessionOpen)
	}

	if result.Code != 0 {
		t.Errorf(msgCodeCero, result.Code)
	}
}

// TestUsersCreateExisting comprueba el conflicto de una cuenta
// existente.
func TestUsersCreateExisting(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{
		{Stdout: lineJorge},
	}}

	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	if _, err := users.Create(context.Background(), nameJorge, false, false); !errors.Is(err, ErrUserExists) {
		t.Fatalf("Crear() error = %v, want ErrUsuarioExistente", err)
	}

	if len(session.commands) != 1 {
		t.Errorf("la sesión ejecutó %d órdenes, want solo la consulta", len(session.commands))
	}
}

// TestUsersCreateWithoutKeys comprueba que sin llaves no se genera el
// par ni se bloquea la contraseña.
func TestUsersCreateWithoutKeys(t *testing.T) {
	t.Parallel()

	ejecutor := &stubExecutor{}
	session := &stubSession{}

	users, _, _ := newUsersForTest(t, ejecutor, session, nil)

	if _, err := users.Create(context.Background(), userAndrey, false, false); err != nil {
		t.Fatalf("Crear() error = %v, want nil", err)
	}

	if len(ejecutor.calls) != 0 {
		t.Error("ssh-keygen se ejecutó sin llaves")
	}

	if len(session.commands) != 2 {
		t.Fatalf(msgCommandDos, len(session.commands))
	}

	if command := session.commands[1].ShellLine(); strings.Contains(command, "passwd -l") {
		t.Errorf("orden = %q, no debe bloquear la contraseña", command)
	}
}

// TestUsersDeleteWithFiles comprueba userdel -r y el borrado de
// las llaves del anfitrión.
func TestUsersDeleteWithFiles(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{
		{Stdout: lineJorge},
		{},
	}}

	users, _, directory := newUsersForTest(t, &stubExecutor{}, session, nil)
	path := filepath.Join(directory, fileJorge)

	for _, file := range []string{path, path + ".pub"} {
		if err := os.WriteFile(file, []byte(contentPrivate), 0o600); err != nil {
			t.Fatalf("crear %s: %v", file, err)
		}
	}

	if _, err := users.Delete(context.Background(), nameJorge, true); err != nil {
		t.Fatalf(msgDeleteOK, err)
	}

	if command := session.commands[1].ShellLine(); !strings.Contains(command, "userdel -r") {
		t.Errorf("orden = %q, want userdel -r", command)
	}

	for _, file := range []string{path, path + ".pub"} {
		if _, err := os.Stat(file); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("la llave %s no se borró", file)
		}
	}
}

// TestUsersDeleteInexistente comprueba el error de una cuenta que
// no existe.
func TestUsersDeleteInexistente(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{{Stdout: ""}}}

	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	if _, err := users.Delete(context.Background(), nameJorge, false); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Eliminar() error = %v, want ErrUsuarioNoEncontrado", err)
	}
}

// TestUsersDownloadSSHKey devuelve la llave privada del usuario.
func TestUsersDownloadSSHKey(t *testing.T) {
	t.Parallel()

	users, _, directory := newUsersForTest(t, &stubExecutor{}, &stubSession{}, nil)
	path := filepath.Join(directory, fileJorge)

	if err := os.WriteFile(path, []byte(contentPrivate), 0o600); err != nil {
		t.Fatalf("crear la llave: %v", err)
	}

	name, content, err := users.DownloadKey(nameJorge)
	if err != nil {
		t.Fatalf("DescargarLlave() error = %v, want nil", err)
	}

	if name != fileJorge || string(content) != contentPrivate {
		t.Errorf("llave = %q %q, want el contenido de jorge", name, string(content))
	}
}

// TestUsersDownloadSSHKeyInexistente devuelve
// ErrUserKeyNotFound.
func TestUsersDownloadSSHKeyInexistente(t *testing.T) {
	t.Parallel()

	users, _, _ := newUsersForTest(t, &stubExecutor{}, &stubSession{}, nil)

	if _, _, err := users.DownloadKey(nameJorge); !errors.Is(err, ErrUserKeyNotFound) {
		t.Fatalf("DescargarLlave() error = %v, want ErrLlaveUsuarioNoEncontrada", err)
	}
}

// TestUsersCreateErrorOfCommand comprueba el fallo de useradd con su
// detalle técnico.
func TestUsersCreateErrorOfCommand(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		results: []domain.Result{{Stdout: ""}},
		result:  domain.Result{Stderr: "useradd: cannot lock /etc/passwd", ExitCode: 1},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: "useradd"}},
	}

	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	result, err := users.Create(context.Background(), nameJorge, false, false)
	if err == nil {
		t.Fatal("Crear() error = nil, want el error de useradd")
	}

	if result.Code != 1 || result.Stderr == "" {
		t.Errorf(msgResultTechnical, result)
	}
}

// TestUsersListWithoutSession comprueba que se propaga la ausencia de
// sesión.
func TestUsersListWithoutSession(t *testing.T) {
	t.Parallel()

	users, _, _ := newUsersForTest(t, &stubExecutor{}, nil, ErrSessionUnavailable)

	if _, err := users.List(context.Background()); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf(msgListWithoutSession, err)
	}
}

// TestUsersListError comprueba el fallo al listar las cuentas.
func TestUsersListError(t *testing.T) {
	t.Parallel()

	session := &stubSession{err: &domain.ExitError{Code: 1, Command: binGetent}}
	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	if _, err := users.List(context.Background()); err == nil {
		t.Fatal("Listar() error = nil, want el error de getent")
	}
}

// TestUsersGetError comprueba un fallo inesperado al consultar.
func TestUsersGetError(t *testing.T) {
	t.Parallel()

	session := &stubSession{errs: []error{&domain.ExitError{Code: 1, Command: binGetent}}}
	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	if _, err := users.Get(context.Background(), nameJorge); err == nil {
		t.Fatal("Obtener() error = nil, want el error de getent")
	}
}

// TestUsersGetGetentWithoutAccount comprueba el código 2 de getent.
func TestUsersGetGetentWithoutAccount(t *testing.T) {
	t.Parallel()

	session := &stubSession{errs: []error{&domain.ExitError{Code: 2, Command: binGetent}}}
	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	if _, err := users.Get(context.Background(), nameJorge); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Obtener() error = %v, want ErrUsuarioNoEncontrado", err)
	}
}

// TestUsersCreateWithoutSession comprueba la ausencia de sesión al crear.
func TestUsersCreateWithoutSession(t *testing.T) {
	t.Parallel()

	users, _, _ := newUsersForTest(t, &stubExecutor{}, nil, ErrSessionUnavailable)

	if _, err := users.Create(context.Background(), nameJorge, false, false); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("Crear() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestUsersCreateWithoutPublic comprueba el fallo al leer la llave
// pública generada.
func TestUsersCreateWithoutPublic(t *testing.T) {
	t.Parallel()

	session := &stubSession{}
	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	if _, err := users.Create(context.Background(), nameJorge, false, true); err == nil {
		t.Fatal("Crear() error = nil, want el error de la llave pública")
	}
}

// TestUsersCreateErrorOfInstallation comprueba el fallo al instalar la
// llave pública en la máquina.
func TestUsersCreateErrorOfInstallation(t *testing.T) {
	t.Parallel()

	ejecutor := &stubExecutor{}
	session := &stubSession{errs: []error{nil, nil, &domain.ExitError{Code: 1, Command: binSh}}}

	users, _, directory := newUsersForTest(t, ejecutor, session, nil)
	ejecutor.onExecute = func() {
		if err := os.WriteFile(filepath.Join(directory, nameJorge+"_ed25519.pub"), []byte(keyPublic+"\n"), 0o600); err != nil {
			t.Fatalf(msgWritePub, err)
		}
	}

	if _, err := users.Create(context.Background(), nameJorge, false, true); err == nil {
		t.Fatal("Crear() error = nil, want el error de instalación")
	}
}

// TestUsersDeleteErrorOfCommand comprueba el fallo de userdel.
func TestUsersDeleteErrorOfCommand(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		results: []domain.Result{{Stdout: lineJorge}},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: binUserdel}},
	}

	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	if _, err := users.Delete(context.Background(), nameJorge, false); err == nil {
		t.Fatal("Eliminar() error = nil, want el error de userdel")
	}
}

// TestUsersDownloadSSHKeyError comprueba un fallo de lectura de la
// llave.
func TestUsersDownloadSSHKeyError(t *testing.T) {
	t.Parallel()

	users, _, directory := newUsersForTest(t, &stubExecutor{}, &stubSession{}, nil)

	if err := os.Mkdir(filepath.Join(directory, nameJorge+"_ed25519"), 0o700); err != nil {
		t.Fatalf(msgCreateDir, err)
	}

	if _, _, err := users.DownloadKey(nameJorge); err == nil {
		t.Fatal("DescargarLlave() error = nil, want el error de lectura")
	}
}

// TestParseUserInvalid comprueba una línea de passwd incompleta.
func TestParseUserInvalid(t *testing.T) {
	t.Parallel()

	if user := parseUser("invalido"); user.Name != "" {
		t.Errorf("parsearUsuario() = %+v, want un usuario vacío", user)
	}
}

// TestUsersListErrorOfGroup comprueba el fallo al consultar sudo.
func TestUsersListErrorOfGroup(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		results: []domain.Result{{Stdout: outputPasswd}},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: binGetent}},
	}
	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	if _, err := users.List(context.Background()); err == nil {
		t.Fatal("Listar() error = nil, want el error del grupo sudo")
	}
}

// TestUsersGetErrorOfGroup comprueba el fallo al consultar sudo
// al obtener una cuenta.
func TestUsersGetErrorOfGroup(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		results: []domain.Result{{Stdout: lineJorge}},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: binGetent}},
	}
	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	if _, err := users.Get(context.Background(), nameJorge); err == nil {
		t.Fatal("Obtener() error = nil, want el error del grupo sudo")
	}
}

// TestUsersCreateErrorOfDirectory comprueba el fallo al crear el
// directorio de llaves.
func TestUsersCreateErrorOfDirectory(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "archivo")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf(msgPrepareFile, err)
	}

	users := NewUsers(UsersConfig{
		Sessions:  &stubOpener{session: &stubSession{}},
		Ejecutor:  &stubExecutor{},
		Directory: file,
	})

	if _, err := users.Create(context.Background(), nameJorge, false, true); err == nil {
		t.Fatal("Crear() error = nil, want el error del directorio")
	}
}

// TestUsersListLimitesOfUID comprueba los extremos del rango de
// cuentas regulares (1000–65533) y descarta las líneas mal formadas o
// con UTF-8 en el nombre.
func TestUsersListLimitesOfUID(t *testing.T) {
	t.Parallel()

	passwd := "root:x:0:0:root:/root:/bin/bash\n" +
		"justo-antes:x:999:999::/home/justo-antes:/bin/bash\n" +
		"borde-inferior:x:1000:1000::/home/borde-inferior:/bin/bash\n" +
		"borde-superior:x:65533:65533::/home/borde-superior:/bin/bash\n" +
		"nadie:x:65534:65534::/nonexistent:/usr/sbin/nologin\n" +
		"fuera-de-rango:x:65535:65535::/home/fuera-de-rango:/bin/bash\n" +
		"sin-numero:x:abc:abc::/home/sin-numero:/bin/bash\n" +
		"incompleta:x:1001:1001\n" +
		"josé:x:1002:1002::/home/josé:/bin/bash\n"

	session := &stubSession{results: []domain.Result{
		{Stdout: passwd},
		{Stdout: ""},
	}}

	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	lists, err := users.List(context.Background())
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	expected := []struct {
		name string
		uid  int
	}{
		{"borde-inferior", 1000},
		{"borde-superior", 65533},
		{"josé", 1002},
	}

	if len(lists) != len(expected) {
		t.Fatalf("Listar() devolvió %d usuarios (%+v), want %d", len(lists), lists, len(expected))
	}

	for i, expected := range expected {
		if lists[i].Name != expected.name || lists[i].UID != expected.uid {
			t.Errorf("usuario[%d] = %+v, want %q con UID %d", i, lists[i], expected.name, expected.uid)
		}
	}
}

// TestUsersGenerateKey comprueba la generación e instalación de la llave
// de un usuario existente.
func TestUsersGenerateKey(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{
		{Stdout: "jorge:x:1000:1000:Jorge:/home/jorge:/bin/bash\n"},
		{},
		{},
	}}
	ejecutor := &stubExecutor{}

	users, opener, directory := newUsersForTest(t, ejecutor, session, nil)
	ejecutor.onExecute = func() {
		if err := os.WriteFile(filepath.Join(directory, "jorge_ed25519"), []byte(contentPrivate), 0o600); err != nil {
			t.Fatalf("crear la llave privada: %v", err)
		}

		if err := os.WriteFile(filepath.Join(directory, "jorge_ed25519.pub"), []byte(keyPublic+"\n"), 0o600); err != nil {
			t.Fatalf(msgWritePub, err)
		}
	}

	result, err := users.GenerateKey(context.Background(), nameJorge)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v, want nil", err)
	}

	if opener.open != 1 {
		t.Errorf(msgOpen, opener.open)
	}

	if len(session.commands) != 3 {
		t.Fatalf("la sesión ejecutó %d órdenes, want 3", len(session.commands))
	}

	install := session.commands[1].ShellLine()
	if !strings.Contains(install, "authorized_keys") || !strings.Contains(install, keyPublic) {
		t.Errorf("orden de instalación = %q, want authorized_keys con la pública", install)
	}

	password := session.commands[2].ShellLine()
	if !strings.Contains(password, binPasswd) || !strings.Contains(password, "'-l'") {
		t.Errorf("orden de contraseña = %q, want passwd -l", password)
	}

	if _, err := os.Stat(filepath.Join(directory, "jorge_ed25519")); err != nil {
		t.Errorf("la llave privada no quedó en el anfitrión: %v", err)
	}

	if result.Message == "" {
		t.Error("Mensaje vacío, want confirmación")
	}
}

// TestUsersGenerateKeyNotFound comprueba el usuario inexistente.
func TestUsersGenerateKeyNotFound(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{{Stdout: ""}}}
	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	if _, err := users.GenerateKey(context.Background(), nameJorge); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("GenerateKey() error = %v, want ErrUserNotFound", err)
	}
}

// TestUsersGenerateKeyExisting comprueba el conflicto si ya hay llave.
func TestUsersGenerateKeyExisting(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{
		{Stdout: "jorge:x:1000:1000:Jorge:/home/jorge:/bin/bash\n"},
	}}
	users, _, directory := newUsersForTest(t, &stubExecutor{}, session, nil)

	if err := os.WriteFile(filepath.Join(directory, "jorge_ed25519"), []byte(contentPrivate), 0o600); err != nil {
		t.Fatalf("crear la llave privada: %v", err)
	}

	if _, err := users.GenerateKey(context.Background(), nameJorge); !errors.Is(err, ErrUserKeyExists) {
		t.Fatalf("GenerateKey() error = %v, want ErrUserKeyExists", err)
	}
}

// TestUsersGenerateKeyInstallError limpia el par si falla la instalación.
func TestUsersGenerateKeyInstallError(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		results: []domain.Result{
			{Stdout: "jorge:x:1000:1000:Jorge:/home/jorge:/bin/bash\n"},
			{},
		},
		errs: []error{nil, errors.New("sin espacio")},
	}
	ejecutor := &stubExecutor{}

	users, _, directory := newUsersForTest(t, ejecutor, session, nil)
	ejecutor.onExecute = func() {
		if err := os.WriteFile(filepath.Join(directory, "jorge_ed25519"), []byte(contentPrivate), 0o600); err != nil {
			t.Fatalf("crear la llave privada: %v", err)
		}

		if err := os.WriteFile(filepath.Join(directory, "jorge_ed25519.pub"), []byte(keyPublic+"\n"), 0o600); err != nil {
			t.Fatalf(msgWritePub, err)
		}
	}

	if _, err := users.GenerateKey(context.Background(), nameJorge); err == nil {
		t.Fatal("GenerateKey() error = nil, want el fallo de instalación")
	}

	for _, file := range []string{"jorge_ed25519", "jorge_ed25519.pub"} {
		if _, err := os.Stat(filepath.Join(directory, file)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s quedó en el anfitrión", file)
		}
	}
}

// TestUsersGenerateKeyWithoutSession propaga el fallo al abrir la sesión.
func TestUsersGenerateKeyWithoutSession(t *testing.T) {
	t.Parallel()

	users, _, _ := newUsersForTest(t, &stubExecutor{}, nil, errors.New("sin sesión"))

	if _, err := users.GenerateKey(context.Background(), nameJorge); err == nil {
		t.Fatal("GenerateKey() error = nil, want el fallo de sesión")
	}
}

// TestUsersGenerateKeyPasswordError propaga el fallo de passwd -l.
func TestUsersGenerateKeyPasswordError(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		results: []domain.Result{
			{Stdout: "jorge:x:1000:1000:Jorge:/home/jorge:/bin/bash\n"},
			{},
			{},
		},
		errs: []error{nil, nil, errors.New("passwd caído")},
	}
	ejecutor := &stubExecutor{}

	users, _, directory := newUsersForTest(t, ejecutor, session, nil)
	ejecutor.onExecute = func() {
		if err := os.WriteFile(filepath.Join(directory, "jorge_ed25519"), []byte(contentPrivate), 0o600); err != nil {
			t.Fatalf("crear la llave privada: %v", err)
		}

		if err := os.WriteFile(filepath.Join(directory, "jorge_ed25519.pub"), []byte(keyPublic+"\n"), 0o600); err != nil {
			t.Fatalf(msgWritePub, err)
		}
	}

	if _, err := users.GenerateKey(context.Background(), nameJorge); err == nil {
		t.Fatal("GenerateKey() error = nil, want el fallo de passwd")
	}
}
