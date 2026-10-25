// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"testing"

	"consola-vm/internal/core/domain"
)

// UUID de las máquinas de prueba.
const (
	uuidDebian   = "8f3a1c2e-1111-4a4a-9a9a-000000000001"
	uuidUbuntu   = "8f3a1c2e-2222-4a4a-9a9a-000000000002"
	nombreDebian = "debian-1"
	nombreUbuntu = "ubuntu-lab"
)

// stubInventory devuelve un inventario preprogramado.
type stubInventory struct {
	maquinas []domain.VM
	err      error
}

// List implementa ports.Inventory.
func (s *stubInventory) List(context.Context) ([]domain.VM, error) {
	return s.maquinas, s.err
}

// stubSeleccion imita el objetivo actual de la consola.
type stubSeleccion struct {
	id string
}

// Objetivo devuelve la máquina vinculada.
func (s *stubSeleccion) Objetivo() domain.Target { return domain.Target{ID: s.id} }

// Establecer vincula una máquina por nombre o UUID.
func (s *stubSeleccion) Establecer(id string) { s.id = id }

// inventarioDePrueba arma dos máquinas, una en ejecución.
func inventarioDePrueba() *stubInventory {
	return &stubInventory{maquinas: []domain.VM{
		{Name: nombreDebian, UUID: uuidDebian, State: domain.StateRunning},
		{Name: nombreUbuntu, UUID: uuidUbuntu, State: domain.StatePoweredOff},
	}}
}

// TestVinculacionListaMarcaLaActiva comprueba que el inventario señala
// la máquina ya vinculada.
func TestVinculacionListaMarcaLaActiva(t *testing.T) {
	t.Parallel()

	seleccion := &stubSeleccion{id: uuidDebian}
	vinculacion := NuevoVinculacion(&ConfigVinculacion{Inventario: inventarioDePrueba(), Seleccion: seleccion})

	maquinas, err := vinculacion.Listar(context.Background(), "")
	if err != nil {
		t.Fatalf("Listar() error = %v, want nil", err)
	}

	if len(maquinas) != 2 {
		t.Fatalf("len(maquinas) = %d, want 2", len(maquinas))
	}

	if !maquinas[0].Activa {
		t.Error("maquinas[0].Activa = false, want true")
	}

	if maquinas[1].Activa {
		t.Error("maquinas[1].Activa = true, want false")
	}
}

// TestVinculacionFiltraPorNombreYUUID comprueba la búsqueda del selector.
func TestVinculacionFiltraPorNombreYUUID(t *testing.T) {
	t.Parallel()

	casos := map[string]string{
		"por nombre":       "ubuntu",
		"por mayúsculas":   "DEBIAN",
		"por UUID parcial": "2222",
	}

	for nombre, filtro := range casos {
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()

			vinculacion := NuevoVinculacion(&ConfigVinculacion{
				Inventario: inventarioDePrueba(),
				Seleccion:  &stubSeleccion{},
			})

			maquinas, err := vinculacion.Listar(context.Background(), filtro)
			if err != nil {
				t.Fatalf("Listar() error = %v, want nil", err)
			}

			if len(maquinas) != 1 {
				t.Fatalf("len(maquinas) = %d, want 1", len(maquinas))
			}

			if filtro == "2222" && maquinas[0].Nombre != nombreUbuntu {
				t.Errorf("maquinas[0].Nombre = %q, want %q", maquinas[0].Nombre, nombreUbuntu)
			}
		})
	}
}

// TestVinculacionAceptaNombreYUUID comprueba las dos formas de
// identificar la máquina.
func TestVinculacionAceptaNombreYUUID(t *testing.T) {
	t.Parallel()

	for _, identificador := range []string{nombreUbuntu, uuidUbuntu} {
		seleccion := &stubSeleccion{}
		vinculacion := NuevoVinculacion(&ConfigVinculacion{Inventario: inventarioDePrueba(), Seleccion: seleccion})

		maquina, err := vinculacion.Vincular(context.Background(), identificador)
		if err != nil {
			t.Fatalf("Vincular(%q) error = %v, want nil", identificador, err)
		}

		if seleccion.id != nombreUbuntu {
			t.Errorf("objetivo = %q, want %q", seleccion.id, nombreUbuntu)
		}

		if !maquina.Activa {
			t.Errorf("Vincular(%q).Activa = false, want true", identificador)
		}

		if maquina.Nombre != nombreUbuntu {
			t.Errorf("Vincular(%q).Nombre = %q, want %q", identificador, maquina.Nombre, nombreUbuntu)
		}
	}
}

// TestVinculacionRechazaMaquinaDesconocida comprueba el error de
// búsqueda sin coincidencias.
func TestVinculacionRechazaMaquinaDesconocida(t *testing.T) {
	t.Parallel()

	seleccion := &stubSeleccion{}
	vinculacion := NuevoVinculacion(&ConfigVinculacion{Inventario: inventarioDePrueba(), Seleccion: seleccion})

	_, err := vinculacion.Vincular(context.Background(), "no-existe")

	if !errors.Is(err, ErrMaquinaNoEncontrada) {
		t.Fatalf("Vincular() error = %v, want %v", err, ErrMaquinaNoEncontrada)
	}

	if seleccion.id != "" {
		t.Errorf("objetivo = %q, want vacío", seleccion.id)
	}
}

// TestVinculacionPropagaElFalloDelInventario conserva la causa del
// error del hipervisor.
func TestVinculacionPropagaElFalloDelInventario(t *testing.T) {
	t.Parallel()

	causa := errors.New("VBoxManage no está disponible")
	vinculacion := NuevoVinculacion(&ConfigVinculacion{
		Inventario: &stubInventory{err: causa},
		Seleccion:  &stubSeleccion{},
	})

	if _, err := vinculacion.Listar(context.Background(), ""); !errors.Is(err, causa) {
		t.Fatalf("Listar() error = %v, want %v", err, causa)
	}
}

// TestVinculacionActivaDevuelveElObjetivo comprueba el nombre de la
// máquina vinculada que se publica en la sesión.
func TestVinculacionActivaDevuelveElObjetivo(t *testing.T) {
	t.Parallel()

	vinculacion := NuevoVinculacion(&ConfigVinculacion{
		Inventario: inventarioDePrueba(),
		Seleccion:  &stubSeleccion{id: nombreDebian},
	})

	if activa := vinculacion.Activa(); activa != nombreDebian {
		t.Errorf("Activa() = %q, want %q", activa, nombreDebian)
	}
}
