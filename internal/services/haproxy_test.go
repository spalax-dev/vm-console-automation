// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"consola-vm/internal/core/domain"
)

// configHaproxy simula una configuración con un front-end y un back-end.
const configHaproxy = `global
    daemon

defaults
    mode http

frontend http
    bind *:80
    default_backend app

backend app
    balance roundrobin
    server srv1 192.168.1.61:8080 check
`

// nuevoHaproxyDePrueba arma el servicio con una sesión falsa.
func nuevoHaproxyDePrueba(sesion *stubSession, err error) *Haproxy {
	return NuevoHaproxy(&stubAbridor{sesion: sesion, err: err})
}

// TestHaproxyListar interpreta los servidores del backend.
func TestHaproxyListar(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{result: domain.Result{Stdout: configHaproxy}}

	servidores, err := nuevoHaproxyDePrueba(sesion, nil).Listar(context.Background())
	if err != nil {
		t.Fatalf("Listar() error = %v, want nil", err)
	}

	if len(servidores) != 1 || servidores[0].Direccion != ipServidorPrueba || servidores[0].Puerto != 8080 {
		t.Errorf("servidores = %+v, want 192.168.1.61:8080", servidores)
	}
}

// TestHaproxyAgregar modifica la configuración y recarga HAProxy.
func TestHaproxyAgregar(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{results: []domain.Result{{Stdout: configHaproxy}, {}}}
	haproxy := nuevoHaproxyDePrueba(sesion, nil)

	resultado, err := haproxy.Agregar(context.Background(), ipServidorNuevo, 9090)
	if err != nil {
		t.Fatalf("Agregar() error = %v, want nil", err)
	}

	if len(sesion.commands) != 2 {
		t.Fatalf(msgOrdenDos, len(sesion.commands))
	}

	orden := sesion.commands[1].ShellLine()
	for _, esperado := range []string{"192.168.1.62:9090", binHaproxy, "-c", "reload haproxy"} {
		if !strings.Contains(orden, esperado) {
			t.Errorf("orden = %q, want %q", orden, esperado)
		}
	}

	if resultado.Codigo != 0 {
		t.Errorf("Codigo = %d, want 0", resultado.Codigo)
	}
}

// TestHaproxyAgregarDuplicado comprueba el servidor ya presente.
func TestHaproxyAgregarDuplicado(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{result: domain.Result{Stdout: configHaproxy}}
	haproxy := nuevoHaproxyDePrueba(sesion, nil)

	if _, err := haproxy.Agregar(context.Background(), ipServidorPrueba, 8080); !errors.Is(err, ErrServidorExistente) {
		t.Fatalf("Agregar() error = %v, want ErrServidorExistente", err)
	}
}

// TestHaproxySinBackend comprueba una configuración sin backend.
func TestHaproxySinBackend(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{result: domain.Result{Stdout: "global\n    daemon\n"}}

	if _, err := nuevoHaproxyDePrueba(sesion, nil).Listar(context.Background()); !errors.Is(err, ErrBackendNoEncontrado) {
		t.Fatalf("Listar() error = %v, want ErrBackendNoEncontrado", err)
	}
}

// TestHaproxyEliminar quita el servidor de la configuración.
func TestHaproxyEliminar(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{results: []domain.Result{{Stdout: configHaproxy}, {}}}
	haproxy := nuevoHaproxyDePrueba(sesion, nil)

	if _, err := haproxy.Eliminar(context.Background(), ipServidorPrueba, 8080); err != nil {
		t.Fatalf(msgEliminarOK, err)
	}

	orden := sesion.commands[1].ShellLine()
	if strings.Contains(orden, "srv1 192.168.1.61:8080") {
		t.Errorf("orden = %q, el servidor no se eliminó", orden)
	}
}

// TestHaproxyEliminarInexistente comprueba el servidor ausente.
func TestHaproxyEliminarInexistente(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{result: domain.Result{Stdout: configHaproxy}}

	if _, err := nuevoHaproxyDePrueba(sesion, nil).Eliminar(context.Background(), "192.168.1.99", 9999); !errors.Is(err, ErrServidorNoEncontrado) {
		t.Fatalf("Eliminar() error = %v, want ErrServidorNoEncontrado", err)
	}
}

// TestAgregarServidorInsertaEnBackend comprueba la línea generada.
func TestAgregarServidorInsertaEnBackend(t *testing.T) {
	t.Parallel()

	nuevo, err := agregarServidor(configHaproxy, ipServidorNuevo, 9090)
	if err != nil {
		t.Fatalf("agregarServidor() error = %v, want nil", err)
	}

	if !strings.Contains(nuevo, "    server srv_192_168_1_62_9090 192.168.1.62:9090 check") {
		t.Errorf("configuración = %q, want la línea del servidor", nuevo)
	}

	if strings.Contains(nuevo, "frontend http\n    server") {
		t.Errorf("configuración = %q, el servidor debe ir en el backend", nuevo)
	}
}

// TestQuitarServidorDejaBackend comprueba la eliminación de la línea.
func TestQuitarServidorDejaBackend(t *testing.T) {
	t.Parallel()

	nuevo, err := quitarServidor(configHaproxy, ipServidorPrueba, 8080)
	if err != nil {
		t.Fatalf("quitarServidor() error = %v, want nil", err)
	}

	if strings.Contains(nuevo, "srv1") {
		t.Errorf("configuración = %q, el servidor no se quitó", nuevo)
	}

	if !strings.Contains(nuevo, "backend app") || !strings.Contains(nuevo, "balance roundrobin") {
		t.Errorf("configuración = %q, want el backend intacto", nuevo)
	}
}

// TestHaproxyAgregarErrorDeRecarga comprueba el fallo de recarga.
func TestHaproxyAgregarErrorDeRecarga(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{
		results: []domain.Result{{Stdout: configHaproxy}},
		result:  domain.Result{Stderr: "haproxy: invalid config", ExitCode: 1},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: binSh}},
	}

	resultado, err := nuevoHaproxyDePrueba(sesion, nil).Agregar(context.Background(), ipServidorNuevo, 9090)
	if err == nil {
		t.Fatal("Agregar() error = nil, want el error de validación")
	}

	if resultado.Codigo != 1 || resultado.Stderr == "" {
		t.Errorf(msgResultadoTecnico, resultado)
	}
}

// TestHaproxySinSesion comprueba la ausencia de sesión.
func TestHaproxySinSesion(t *testing.T) {
	t.Parallel()

	haproxy := nuevoHaproxyDePrueba(nil, ErrSesionNoDisponible)

	if _, err := haproxy.Listar(context.Background()); !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf("Listar() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestHaproxyErroresDeSesion cubre la ausencia de sesión y la lectura de
// la configuración.
func TestHaproxyErroresDeSesion(t *testing.T) {
	t.Parallel()

	sinSesion := nuevoHaproxyDePrueba(nil, ErrSesionNoDisponible)

	if _, err := sinSesion.Agregar(context.Background(), ipServidorPrueba, 8080); !errors.Is(err, ErrSesionNoDisponible) {
		t.Errorf("Agregar() error = %v, want ErrSesionNoDisponible", err)
	}

	if _, err := sinSesion.Eliminar(context.Background(), ipServidorPrueba, 8080); !errors.Is(err, ErrSesionNoDisponible) {
		t.Errorf("Eliminar() error = %v, want ErrSesionNoDisponible", err)
	}

	conError := nuevoHaproxyDePrueba(&stubSession{err: &domain.ExitError{Code: 1, Command: binCat}}, nil)

	if _, err := conError.Listar(context.Background()); err == nil {
		t.Error("Listar() error = nil, want el error de lectura")
	}

	if _, err := conError.Agregar(context.Background(), ipServidorPrueba, 8080); err == nil {
		t.Error("Agregar() error = nil, want el error de lectura")
	}
}

// TestHaproxyEliminarErrorDeRecarga cubre el fallo al recargar HAProxy.
func TestHaproxyEliminarErrorDeRecarga(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{
		results: []domain.Result{{Stdout: configHaproxy}},
		result:  domain.Result{Stderr: "reload failed", ExitCode: 1},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: binSh}},
	}

	if _, err := nuevoHaproxyDePrueba(sesion, nil).Eliminar(context.Background(), ipServidorPrueba, 8080); err == nil {
		t.Fatal("Eliminar() error = nil, want el error de recarga")
	}
}
