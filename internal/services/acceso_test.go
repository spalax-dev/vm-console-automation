// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"net/netip"
	"path/filepath"
	"testing"

	"consola-vm/internal/core/domain"
)

// nuevoAccesoDePrueba arma el servicio con máquina y conector falsos.
func nuevoAccesoDePrueba(t *testing.T, estado domain.MachineState, sesion *stubSession) (*Acceso, *Almacen) {
	t.Helper()

	almacen := NuevoAlmacen(filepath.Join(t.TempDir(), archivoCredenciales))
	maquina := &stubMachine{addr: netip.MustParseAddr(ipPrueba), state: estado}
	conector := &stubConnector{name: conectorGuest, session: sesion}

	acceso := NuevoAcceso(&ConfigAcceso{
		Almacen:  almacen,
		Maquina:  maquina,
		Guest:    conector,
		Objetivo: func() domain.Target { return domain.Target{ID: nombreDebian} },
	})

	return acceso, almacen
}

// TestAccesoIniciarGuardaCredenciales comprueba el guardado tras la
// verificación.
func TestAccesoIniciarGuardaCredenciales(t *testing.T) {
	t.Parallel()

	acceso, almacen := nuevoAccesoDePrueba(t, domain.StateRunning, &stubSession{})

	resultado, err := acceso.Iniciar(context.Background(), usuarioRoot, passwordPrueba, 22)
	if err != nil {
		t.Fatalf("Iniciar() error = %v, want nil", err)
	}

	if resultado.Mensaje == "" {
		t.Error("Mensaje vacío, want confirmación")
	}

	credenciales, ok := almacen.Obtener(nombreDebian)
	if !ok || credenciales.Usuario != usuarioRoot {
		t.Errorf("credenciales = %+v, want root", credenciales)
	}
}

// TestAccesoIniciarRestauraSiFalla comprueba que no deja credenciales
// inválidas.
func TestAccesoIniciarRestauraSiFalla(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{err: &domain.ExitError{Code: 1, Command: "id"}}
	acceso, almacen := nuevoAccesoDePrueba(t, domain.StateRunning, sesion)

	if _, err := acceso.Iniciar(context.Background(), usuarioRoot, "mala", 22); err == nil {
		t.Fatal("Iniciar() error = nil, want el fallo de verificación")
	}

	if _, ok := almacen.Obtener(nombreDebian); ok {
		t.Error("se guardaron credenciales inválidas")
	}
}

// TestAccesoIniciarSinMaquina comprueba la consola sin vínculo.
func TestAccesoIniciarSinMaquina(t *testing.T) {
	t.Parallel()

	almacen := NuevoAlmacen(filepath.Join(t.TempDir(), archivoCredenciales))
	acceso := NuevoAcceso(&ConfigAcceso{
		Almacen:  almacen,
		Objetivo: func() domain.Target { return domain.Target{} },
	})

	if _, err := acceso.Iniciar(context.Background(), usuarioRoot, "x", 22); !errors.Is(err, ErrSinMaquina) {
		t.Fatalf("Iniciar() error = %v, want ErrSinMaquina", err)
	}
}

// TestAccesoIniciarMaquinaApagada comprueba la VM apagada.
func TestAccesoIniciarMaquinaApagada(t *testing.T) {
	t.Parallel()

	acceso, _ := nuevoAccesoDePrueba(t, domain.StatePoweredOff, &stubSession{})

	if _, err := acceso.Iniciar(context.Background(), usuarioRoot, "x", 22); !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf("Iniciar() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestAccesoOlvidar elimina las credenciales guardadas.
func TestAccesoOlvidar(t *testing.T) {
	t.Parallel()

	acceso, almacen := nuevoAccesoDePrueba(t, domain.StateRunning, &stubSession{})

	if err := almacen.Guardar(nombreDebian, Credenciales{Usuario: usuarioRoot}); err != nil {
		t.Fatalf("Guardar() error = %v", err)
	}

	if err := acceso.Olvidar(); err != nil {
		t.Fatalf("Olvidar() error = %v, want nil", err)
	}

	if _, ok := almacen.Obtener(nombreDebian); ok {
		t.Error("Obtener() encontró credenciales olvidadas")
	}
}
