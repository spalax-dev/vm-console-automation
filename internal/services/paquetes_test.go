// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"testing"

	"consola-vm/internal/core/domain"
)

// salidaPaquetes simula la salida de dpkg-query con paquetes instalados,
// desinstalados y configurados a medias.
const (
	// paqueteNano es el paquete usado en las pruebas.
	pkgNano = "nano"
	// salidaInstalado simula la salida de apt-get.
	outputInstalado = "instalado"
)

const outputPackages = "bash\t5.2.15-2\tinstall ok installed\n" +
	"nano\t7.2-1\tinstall ok installed\n" +
	"foo\t1.0\tdeinstall ok config-files\n"

// newPackagesOfTests arma el servicio con una sesión falsa.
func newPackagesOfTests(session *stubSession, err error) (*Packages, *stubOpener) {
	opener := &stubOpener{session: session, err: err}

	return NewPackages(opener), opener
}

// TestPackagesInstallRunsAptGet comprueba la orden de instalación y
// el cierre de la sesión.
func TestPackagesInstallRunsAptGet(t *testing.T) {
	t.Parallel()

	session := &stubSession{result: domain.Result{Stdout: outputInstalado}}
	packages, opener := newPackagesOfTests(session, nil)

	result, err := packages.Install(context.Background(), pkgNano)
	if err != nil {
		t.Fatalf("Instalar() error = %v, want nil", err)
	}

	if opener.open != 1 {
		t.Errorf(msgOpen, opener.open)
	}

	if len(session.commands) != 1 {
		t.Fatalf("Run() se llamó %d veces, want 1", len(session.commands))
	}

	cmd := session.commands[0]
	if cmd.Path != "/usr/bin/apt-get" {
		t.Errorf("Path = %q, want %q", cmd.Path, "/usr/bin/apt-get")
	}

	if len(cmd.Args) != 3 || cmd.Args[0] != "install" || cmd.Args[1] != "-y" || cmd.Args[2] != pkgNano {
		t.Errorf("Args = %v, want [install -y nano]", cmd.Args)
	}

	if len(cmd.Env) != 1 || cmd.Env[0] != "DEBIAN_FRONTEND=noninteractive" {
		t.Errorf("Env = %v, want DEBIAN_FRONTEND=noninteractive", cmd.Env)
	}

	if !session.closed {
		t.Error(msgSessionOpen)
	}

	if result.Code != 0 {
		t.Errorf(msgCodeCero, result.Code)
	}

	if result.Output != outputInstalado {
		t.Errorf("Salida = %q, want %q", result.Output, outputInstalado)
	}

	if result.Message != "Paquete «nano» instalado." {
		t.Errorf("Mensaje = %q, want el mensaje de éxito", result.Message)
	}
}

// TestPackagesDeleteRunsAptGet comprueba la orden de eliminación.
func TestPackagesDeleteRunsAptGet(t *testing.T) {
	t.Parallel()

	session := &stubSession{}
	packages, _ := newPackagesOfTests(session, nil)

	if _, err := packages.Delete(context.Background(), pkgNano); err != nil {
		t.Fatalf(msgDeleteOK, err)
	}

	cmd := session.commands[0]
	if len(cmd.Args) != 3 || cmd.Args[0] != "remove" || cmd.Args[1] != "-y" || cmd.Args[2] != pkgNano {
		t.Errorf("Args = %v, want [remove -y nano]", cmd.Args)
	}
}

// TestPackagesKeepsExitError comprueba que el fallo de APT conserva
// el código de salida y el stderr.
func TestPackagesKeepsExitError(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		result: domain.Result{Stderr: "E: Unable to locate package", ExitCode: 100},
		err:    &domain.ExitError{Code: 100, Command: "apt-get install -y nano"},
	}
	packages, _ := newPackagesOfTests(session, nil)

	result, err := packages.Install(context.Background(), pkgNano)
	if err == nil {
		t.Fatal("Instalar() error = nil, want el error de APT")
	}

	var exitErr *domain.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("Instalar() error = %v, quiere ser *domain.ExitError", err)
	}

	if result.Code != 100 {
		t.Errorf("Codigo = %d, want 100", result.Code)
	}

	if result.Stderr != "E: Unable to locate package" {
		t.Errorf("Stderr = %q, want el stderr de APT", result.Stderr)
	}

	if result.Operation == "" {
		t.Error("Operacion vacía, want la orden ejecutada")
	}

	if !session.closed {
		t.Error(msgSessionOpen)
	}
}

// TestPackagesPropagatesErrorOfSession comprueba que un fallo al abrir la
// sesión se propaga sin ejecutar nada.
func TestPackagesPropagatesErrorOfSession(t *testing.T) {
	t.Parallel()

	packages, _ := newPackagesOfTests(nil, ErrSessionUnavailable)

	if _, err := packages.Install(context.Background(), pkgNano); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("Instalar() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestPackagesListFiltraInstalados comprueba el parseo de dpkg-query.
func TestPackagesListFiltraInstalados(t *testing.T) {
	t.Parallel()

	session := &stubSession{result: domain.Result{Stdout: outputPackages}}
	packages, _ := newPackagesOfTests(session, nil)

	lists, err := packages.List(context.Background())
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	if len(lists) != 2 {
		t.Fatalf("Listar() devolvió %d paquetes, want 2", len(lists))
	}

	if lists[0].Name != "bash" || lists[0].Version != "5.2.15-2" {
		t.Errorf("primer paquete = %+v, want bash 5.2.15-2", lists[0])
	}

	if lists[1].Name != pkgNano {
		t.Errorf("segundo paquete = %+v, want nano", lists[1])
	}

	cmd := session.commands[0]
	if cmd.Path != "/usr/bin/dpkg-query" {
		t.Errorf("Path = %q, want %q", cmd.Path, "/usr/bin/dpkg-query")
	}
}

// TestPackagesListEmptyReturnsSlice comprueba que una salida vacía
// produce una lista vacía y no nil.
func TestPackagesListEmptyReturnsSlice(t *testing.T) {
	t.Parallel()

	packages, _ := newPackagesOfTests(&stubSession{}, nil)

	lists, err := packages.List(context.Background())
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	if lists == nil {
		t.Error("Listar() = nil, want una lista vacía")
	}

	if len(lists) != 0 {
		t.Errorf("Listar() devolvió %d paquetes, want 0", len(lists))
	}
}

// TestPackagesListPropagatesError comprueba que el fallo de dpkg-query
// se propaga.
func TestPackagesListPropagatesError(t *testing.T) {
	t.Parallel()

	session := &stubSession{err: &domain.ExitError{Code: 2, Command: "dpkg-query"}}
	packages, _ := newPackagesOfTests(session, nil)

	if _, err := packages.List(context.Background()); err == nil {
		t.Fatal("Listar() error = nil, want el error de dpkg-query")
	}
}

// TestPackagesListWithoutSession comprueba que el fallo al abrir la sesión
// se propaga al listar.
func TestPackagesListWithoutSession(t *testing.T) {
	t.Parallel()

	packages, _ := newPackagesOfTests(nil, ErrSessionUnavailable)

	if _, err := packages.List(context.Background()); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf(msgListWithoutSession, err)
	}
}

// TestPackagesClosesSessionWithError comprueba que un fallo al cerrar la
// sesión no invalida el resultado de la operación.
func TestPackagesClosesSessionWithError(t *testing.T) {
	t.Parallel()

	session := &stubSession{closeErr: errors.New("cierre fallido")}
	packages, _ := newPackagesOfTests(session, nil)

	if _, err := packages.Install(context.Background(), pkgNano); err != nil {
		t.Fatalf("Instalar() error = %v, want nil", err)
	}

	if !session.closed {
		t.Error("la sesión no intentó cerrarse")
	}
}
