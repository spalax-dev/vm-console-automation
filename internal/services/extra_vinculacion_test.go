// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"testing"

	"consola-vm/internal/core/domain"
)

// TestAdministrableCoversNetworks descarta los modos que el anfitrión no
// alcanza y las consultas fallidas.
func TestAdministrableCoversNetworks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		mode    domain.NetworkMode
		visible bool
	}{
		{"internal", domain.NetworkInternal, false},
		{"none", domain.NetworkNone, false},
		{"desconocida", domain.NetworkUnknown, false},
		{"host-only", domain.NetworkHostOnly, true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			machine := &stubGuestAdditions{
				accesibles: map[string]bool{nameUbuntu: true},
				networks:   map[string]domain.NetworkMode{nameUbuntu: testCase.mode},
			}
			binding := NewBinding(&BindingConfig{
				Inventory: &stubInventory{machines: []domain.VM{
					{Name: nameUbuntu, UUID: uuidUbuntu, State: domain.StatePoweredOff},
				}},
				Selection: &stubSelection{},
				Machine:   machine,
			})

			visibles, err := binding.List(context.Background(), "")
			if err != nil {
				t.Fatalf("Listar() error = %v, want nil", err)
			}

			if visible := len(visibles) == 1; visible != testCase.visible {
				t.Errorf("visible = %v, want %v", visible, testCase.visible)
			}
		})
	}
}

// TestAdministrableWithoutMachine lista todo cuando no hay comprobación.
func TestAdministrableWithoutMachine(t *testing.T) {
	t.Parallel()

	binding := NewBinding(&BindingConfig{
		Inventory: &stubInventory{machines: []domain.VM{
			{Name: testMachineName, UUID: uuidTest, State: domain.StateRunning},
		}},
		Selection: &stubSelection{},
	})

	visibles, err := binding.List(context.Background(), "")
	if err != nil || len(visibles) != 1 {
		t.Errorf("Listar() = %d, %v, want una máquina", len(visibles), err)
	}
}

// TestAdministrableWithNetworkUnreadable oculta la máquina si la red falla.
func TestAdministrableWithNetworkUnreadable(t *testing.T) {
	t.Parallel()

	binding := NewBinding(&BindingConfig{
		Inventory: &stubInventory{machines: []domain.VM{
			{Name: testMachineName, UUID: uuidTest, State: domain.StateRunning},
		}},
		Selection: &stubSelection{},
		Machine:   &stubGuestAdditions{errNetwork: errors.New("sin red")},
	})

	visibles, err := binding.List(context.Background(), "")
	if err != nil {
		t.Fatalf("Listar() error = %v, want nil", err)
	}

	if len(visibles) != 0 {
		t.Errorf("visibles = %+v, want vacío", visibles)
	}
}

// TestBindWithInventoryCaido propaga el fallo de VirtualBox.
func TestBindWithInventoryCaido(t *testing.T) {
	t.Parallel()

	binding := NewBinding(&BindingConfig{
		Inventory: &stubInventory{err: errors.New("sin VBoxManage")},
		Selection: &stubSelection{},
	})

	if _, err := binding.Bind(context.Background(), testMachineName); err == nil {
		t.Error("Vincular() error = nil, want el fallo del inventario")
	}

	if _, err := binding.PowerOn(context.Background(), testMachineName); err == nil {
		t.Error("Encender() error = nil, want el fallo del inventario")
	}
}

// TestSaveActiveNotRompeBinding solo advierte del fallo.
func TestSaveActiveNotRompeBinding(t *testing.T) {
	t.Parallel()

	binding := NewBinding(&BindingConfig{
		Inventory:   inventoryOfTests(),
		Selection:   &stubSelection{},
		Persistence: &stubPersistence{err: errors.New("base caída")},
	})

	if _, err := binding.Bind(context.Background(), testMachineName); err != nil {
		t.Errorf("Vincular() error = %v, want nil", err)
	}
}
