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

// stubMaquina informa el estado y la dirección de la máquina vinculada.
type stubMaquina struct {
	estado    domain.MachineState
	direccion netip.Addr
}

// State devuelve el estado preprogramado.
func (s *stubMaquina) State(context.Context, domain.Target) (domain.MachineState, error) {
	return s.estado, nil
}

// Address devuelve la dirección preprogramada.
func (s *stubMaquina) Address(context.Context, domain.Target) (netip.Addr, error) {
	return s.direccion, nil
}

// Network devuelve una red puente.
func (s *stubMaquina) Network(context.Context, domain.Target) (domain.NetworkMode, error) {
	return domain.NetworkBridged, nil
}

// TestDescubridorSinMaquinaVinculada comprueba que la consola sin
// vínculo responde «sin sesión» en lugar de consultar el hipervisor.
func TestDescubridorSinMaquinaVinculada(t *testing.T) {
	t.Parallel()

	descubridor := NuevoDescubridor(&ConfigDescubridor{
		Maquina:  &stubMaquina{estado: domain.StateRunning, direccion: netip.MustParseAddr(ipVinculada)},
		Objetivo: domain.Target{},
	})

	_, err := descubridor.Descubrir(context.Background())

	if !errors.Is(err, ErrSinMaquina) {
		t.Fatalf("Descubrir() error = %v, want %v", err, ErrSinMaquina)
	}

	if !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf("Descubrir() error = %v, want que envuelva %v", err, ErrSesionNoDisponible)
	}
}

// TestDescubridorVinculaOtraMaquina comprueba que establecer el
// objetivo cambia la máquina que se consulta.
func TestDescubridorVinculaOtraMaquina(t *testing.T) {
	t.Parallel()

	descubridor := NuevoDescubridor(&ConfigDescubridor{
		Maquina:  &stubMaquina{estado: domain.StateRunning, direccion: netip.MustParseAddr(ipVinculada)},
		Objetivo: domain.Target{ID: nombreDebian},
	})

	descubridor.Establecer(nombreUbuntu)

	if objetivo := descubridor.Objetivo(); objetivo.ID != nombreUbuntu {
		t.Fatalf("Objetivo().ID = %q, want %q", objetivo.ID, nombreUbuntu)
	}
}
