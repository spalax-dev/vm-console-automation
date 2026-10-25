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
	paqueteNano = "nano"
	// salidaInstalado simula la salida de apt-get.
	salidaInstalado = "instalado"
)

const salidaPaquetes = "bash\t5.2.15-2\tinstall ok installed\n" +
	"nano\t7.2-1\tinstall ok installed\n" +
	"foo\t1.0\tdeinstall ok config-files\n"

// nuevoPaquetesDePrueba arma el servicio con una sesión falsa.
func nuevoPaquetesDePrueba(sesion *stubSession, err error) (*Paquetes, *stubAbridor) {
	abridor := &stubAbridor{sesion: sesion, err: err}

	return NuevoPaquetes(abridor), abridor
}

// TestPaquetesInstalarEjecutaAptGet comprueba la orden de instalación y
// el cierre de la sesión.
func TestPaquetesInstalarEjecutaAptGet(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{result: domain.Result{Stdout: salidaInstalado}}
	paquetes, abridor := nuevoPaquetesDePrueba(sesion, nil)

	resultado, err := paquetes.Instalar(context.Background(), paqueteNano)
	if err != nil {
		t.Fatalf("Instalar() error = %v, want nil", err)
	}

	if abridor.abiertas != 1 {
		t.Errorf(msgAbrirUna, abridor.abiertas)
	}

	if len(sesion.commands) != 1 {
		t.Fatalf("Run() se llamó %d veces, want 1", len(sesion.commands))
	}

	cmd := sesion.commands[0]
	if cmd.Path != "/usr/bin/apt-get" {
		t.Errorf("Path = %q, want %q", cmd.Path, "/usr/bin/apt-get")
	}

	if len(cmd.Args) != 3 || cmd.Args[0] != "install" || cmd.Args[1] != "-y" || cmd.Args[2] != paqueteNano {
		t.Errorf("Args = %v, want [install -y nano]", cmd.Args)
	}

	if len(cmd.Env) != 1 || cmd.Env[0] != "DEBIAN_FRONTEND=noninteractive" {
		t.Errorf("Env = %v, want DEBIAN_FRONTEND=noninteractive", cmd.Env)
	}

	if !sesion.closed {
		t.Error(msgSesionAbierta)
	}

	if resultado.Codigo != 0 {
		t.Errorf(msgCodigoCero, resultado.Codigo)
	}

	if resultado.Salida != salidaInstalado {
		t.Errorf("Salida = %q, want %q", resultado.Salida, salidaInstalado)
	}

	if resultado.Mensaje != "Paquete «nano» instalado." {
		t.Errorf("Mensaje = %q, want el mensaje de éxito", resultado.Mensaje)
	}
}

// TestPaquetesEliminarEjecutaAptGet comprueba la orden de eliminación.
func TestPaquetesEliminarEjecutaAptGet(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{}
	paquetes, _ := nuevoPaquetesDePrueba(sesion, nil)

	if _, err := paquetes.Eliminar(context.Background(), paqueteNano); err != nil {
		t.Fatalf("Eliminar() error = %v, want nil", err)
	}

	cmd := sesion.commands[0]
	if len(cmd.Args) != 3 || cmd.Args[0] != "remove" || cmd.Args[1] != "-y" || cmd.Args[2] != paqueteNano {
		t.Errorf("Args = %v, want [remove -y nano]", cmd.Args)
	}
}

// TestPaquetesConservaExitError comprueba que el fallo de APT conserva
// el código de salida y el stderr.
func TestPaquetesConservaExitError(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{
		result: domain.Result{Stderr: "E: Unable to locate package", ExitCode: 100},
		err:    &domain.ExitError{Code: 100, Command: "apt-get install -y nano"},
	}
	paquetes, _ := nuevoPaquetesDePrueba(sesion, nil)

	resultado, err := paquetes.Instalar(context.Background(), paqueteNano)
	if err == nil {
		t.Fatal("Instalar() error = nil, want el error de APT")
	}

	var exitErr *domain.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("Instalar() error = %v, quiere ser *domain.ExitError", err)
	}

	if resultado.Codigo != 100 {
		t.Errorf("Codigo = %d, want 100", resultado.Codigo)
	}

	if resultado.Stderr != "E: Unable to locate package" {
		t.Errorf("Stderr = %q, want el stderr de APT", resultado.Stderr)
	}

	if resultado.Operacion == "" {
		t.Error("Operacion vacía, want la orden ejecutada")
	}

	if !sesion.closed {
		t.Error(msgSesionAbierta)
	}
}

// TestPaquetesPropagaErrorDeSesion comprueba que un fallo al abrir la
// sesión se propaga sin ejecutar nada.
func TestPaquetesPropagaErrorDeSesion(t *testing.T) {
	t.Parallel()

	paquetes, _ := nuevoPaquetesDePrueba(nil, ErrSesionNoDisponible)

	if _, err := paquetes.Instalar(context.Background(), paqueteNano); !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf("Instalar() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestPaquetesListarFiltraInstalados comprueba el parseo de dpkg-query.
func TestPaquetesListarFiltraInstalados(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{result: domain.Result{Stdout: salidaPaquetes}}
	paquetes, _ := nuevoPaquetesDePrueba(sesion, nil)

	lista, err := paquetes.Listar(context.Background())
	if err != nil {
		t.Fatalf(msgListarOK, err)
	}

	if len(lista) != 2 {
		t.Fatalf("Listar() devolvió %d paquetes, want 2", len(lista))
	}

	if lista[0].Nombre != "bash" || lista[0].Version != "5.2.15-2" {
		t.Errorf("primer paquete = %+v, want bash 5.2.15-2", lista[0])
	}

	if lista[1].Nombre != paqueteNano {
		t.Errorf("segundo paquete = %+v, want nano", lista[1])
	}

	cmd := sesion.commands[0]
	if cmd.Path != "/usr/bin/dpkg-query" {
		t.Errorf("Path = %q, want %q", cmd.Path, "/usr/bin/dpkg-query")
	}
}

// TestPaquetesListarVacioDevuelveSlice comprueba que una salida vacía
// produce una lista vacía y no nil.
func TestPaquetesListarVacioDevuelveSlice(t *testing.T) {
	t.Parallel()

	paquetes, _ := nuevoPaquetesDePrueba(&stubSession{}, nil)

	lista, err := paquetes.Listar(context.Background())
	if err != nil {
		t.Fatalf(msgListarOK, err)
	}

	if lista == nil {
		t.Error("Listar() = nil, want una lista vacía")
	}

	if len(lista) != 0 {
		t.Errorf("Listar() devolvió %d paquetes, want 0", len(lista))
	}
}

// TestPaquetesListarPropagaError comprueba que el fallo de dpkg-query
// se propaga.
func TestPaquetesListarPropagaError(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{err: &domain.ExitError{Code: 2, Command: "dpkg-query"}}
	paquetes, _ := nuevoPaquetesDePrueba(sesion, nil)

	if _, err := paquetes.Listar(context.Background()); err == nil {
		t.Fatal("Listar() error = nil, want el error de dpkg-query")
	}
}

// TestPaquetesListarSinSesion comprueba que el fallo al abrir la sesión
// se propaga al listar.
func TestPaquetesListarSinSesion(t *testing.T) {
	t.Parallel()

	paquetes, _ := nuevoPaquetesDePrueba(nil, ErrSesionNoDisponible)

	if _, err := paquetes.Listar(context.Background()); !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf("Listar() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestPaquetesCierraSesionConError comprueba que un fallo al cerrar la
// sesión no invalida el resultado de la operación.
func TestPaquetesCierraSesionConError(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{closeErr: errors.New("cierre fallido")}
	paquetes, _ := nuevoPaquetesDePrueba(sesion, nil)

	if _, err := paquetes.Instalar(context.Background(), paqueteNano); err != nil {
		t.Fatalf("Instalar() error = %v, want nil", err)
	}

	if !sesion.closed {
		t.Error("la sesión no intentó cerrarse")
	}
}
