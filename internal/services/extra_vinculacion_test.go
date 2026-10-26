// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"testing"

	"consola-vm/internal/core/domain"
)

// TestAdministrableCubreLasRedes descarta los modos que el anfitrión no
// alcanza y las consultas fallidas.
func TestAdministrableCubreLasRedes(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre  string
		modo    domain.NetworkMode
		visible bool
	}{
		{"internal", domain.NetworkInternal, false},
		{"none", domain.NetworkNone, false},
		{"desconocida", domain.NetworkUnknown, false},
		{"host-only", domain.NetworkHostOnly, true},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			maquina := &stubGuestAdditions{
				accesibles: map[string]bool{nombreUbuntu: true},
				redes:      map[string]domain.NetworkMode{nombreUbuntu: caso.modo},
			}
			vinculacion := NuevoVinculacion(&ConfigVinculacion{
				Inventario: &stubInventory{maquinas: []domain.VM{
					{Name: nombreUbuntu, UUID: uuidUbuntu, State: domain.StatePoweredOff},
				}},
				Seleccion: &stubSeleccion{},
				Maquina:   maquina,
			})

			visibles, err := vinculacion.Listar(context.Background(), "")
			if err != nil {
				t.Fatalf("Listar() error = %v, want nil", err)
			}

			if visible := len(visibles) == 1; visible != caso.visible {
				t.Errorf("visible = %v, want %v", visible, caso.visible)
			}
		})
	}
}

// TestAdministrableSinMaquina lista todo cuando no hay comprobación.
func TestAdministrableSinMaquina(t *testing.T) {
	t.Parallel()

	vinculacion := NuevoVinculacion(&ConfigVinculacion{
		Inventario: &stubInventory{maquinas: []domain.VM{
			{Name: nombreDebian, UUID: uuidDebian, State: domain.StateRunning},
		}},
		Seleccion: &stubSeleccion{},
	})

	visibles, err := vinculacion.Listar(context.Background(), "")
	if err != nil || len(visibles) != 1 {
		t.Errorf("Listar() = %d, %v, want una máquina", len(visibles), err)
	}
}

// TestAdministrableConRedIlegible oculta la máquina si la red falla.
func TestAdministrableConRedIlegible(t *testing.T) {
	t.Parallel()

	vinculacion := NuevoVinculacion(&ConfigVinculacion{
		Inventario: &stubInventory{maquinas: []domain.VM{
			{Name: nombreDebian, UUID: uuidDebian, State: domain.StateRunning},
		}},
		Seleccion: &stubSeleccion{},
		Maquina:   &stubGuestAdditions{errRed: errors.New("sin red")},
	})

	visibles, err := vinculacion.Listar(context.Background(), "")
	if err != nil {
		t.Fatalf("Listar() error = %v, want nil", err)
	}

	if len(visibles) != 0 {
		t.Errorf("visibles = %+v, want vacío", visibles)
	}
}

// TestVincularConInventarioCaido propaga el fallo de VirtualBox.
func TestVincularConInventarioCaido(t *testing.T) {
	t.Parallel()

	vinculacion := NuevoVinculacion(&ConfigVinculacion{
		Inventario: &stubInventory{err: errors.New("sin VBoxManage")},
		Seleccion:  &stubSeleccion{},
	})

	if _, err := vinculacion.Vincular(context.Background(), nombreDebian); err == nil {
		t.Error("Vincular() error = nil, want el fallo del inventario")
	}

	if _, err := vinculacion.Encender(context.Background(), nombreDebian); err == nil {
		t.Error("Encender() error = nil, want el fallo del inventario")
	}
}

// TestGuardarActivaNoRompeLaVinculacion solo advierte del fallo.
func TestGuardarActivaNoRompeLaVinculacion(t *testing.T) {
	t.Parallel()

	vinculacion := NuevoVinculacion(&ConfigVinculacion{
		Inventario:   inventarioDePrueba(),
		Seleccion:    &stubSeleccion{},
		Persistencia: &stubPersistencia{err: errors.New("base caída")},
	})

	if _, err := vinculacion.Vincular(context.Background(), nombreDebian); err != nil {
		t.Errorf("Vincular() error = %v, want nil", err)
	}
}
