// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"consola-vm/internal/core/domain"
)

// ipVinculada es la dirección que publica la máquina de prueba.
const ipVinculada = "192.168.1.77"

// stubLinkedMachine informa el estado y la dirección de la máquina vinculada.
type stubLinkedMachine struct {
	state   domain.MachineState
	address netip.Addr
}

// State devuelve el estado preprogramado.
func (s *stubLinkedMachine) State(context.Context, domain.Target) (domain.MachineState, error) {
	return s.state, nil
}

// Address devuelve la dirección preprogramada.
func (s *stubLinkedMachine) Address(context.Context, domain.Target) (netip.Addr, error) {
	return s.address, nil
}

// Network devuelve una red puente.
func (s *stubLinkedMachine) Network(context.Context, domain.Target) (domain.NetworkMode, error) {
	return domain.NetworkBridged, nil
}

// GuestAdditions no informa datos en esta prueba.
func (s *stubLinkedMachine) GuestAdditions(context.Context, domain.Target) (domain.GuestAdditions, error) {
	return domain.GuestAdditions{}, nil
}

// TestDiscovererWithoutMachineVinculada comprueba que la consola sin
// vínculo responde «sin sesión» en lugar de consultar el hipervisor.
func TestDiscovererWithoutMachineVinculada(t *testing.T) {
	t.Parallel()

	discoverer := NewDiscoverer(&DiscovererConfig{
		Machine:       &stubLinkedMachine{state: domain.StateRunning, address: netip.MustParseAddr(ipVinculada)},
		CurrentTarget: domain.Target{},
	})

	_, err := discoverer.Discover(context.Background())

	if !errors.Is(err, ErrNoMachine) {
		t.Fatalf("Descubrir() error = %v, want %v", err, ErrNoMachine)
	}

	if !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("Descubrir() error = %v, want que envuelva %v", err, ErrSessionUnavailable)
	}
}

// TestDiscovererBindsOtherMachine comprueba que establecer el
// objetivo cambia la máquina que se consulta.
func TestDiscovererBindsOtherMachine(t *testing.T) {
	t.Parallel()

	discoverer := NewDiscoverer(&DiscovererConfig{
		Machine:       &stubLinkedMachine{state: domain.StateRunning, address: netip.MustParseAddr(ipVinculada)},
		CurrentTarget: domain.Target{ID: testMachineName},
	})

	discoverer.Set(nameUbuntu)

	if target := discoverer.CurrentTarget(); target.ID != nameUbuntu {
		t.Fatalf("Objetivo().ID = %q, want %q", target.ID, nameUbuntu)
	}
}
