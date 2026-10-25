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

// stubGuestAdditions simula la comprobación de Guest Additions.
type stubGuestAdditions struct {
	accesibles map[string]bool
	consultas  int
}

// Address devuelve una IP solo si la máquina publica Guest Additions.
func (s *stubGuestAdditions) Address(_ context.Context, target domain.Target) (netip.Addr, error) {
	s.consultas++

	if s.accesibles[target.ID] {
		return netip.MustParseAddr(ipPrueba), nil
	}

	return netip.Addr{}, errors.New("sin guest additions")
}

// State devuelve el estado en ejecución.
func (s *stubGuestAdditions) State(context.Context, domain.Target) (domain.MachineState, error) {
	return domain.StateRunning, nil
}

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
		t.Fatalf(msgListarOK, err)
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
				t.Fatalf(msgListarOK, err)
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

// msgEncender es el formato de los fallos de Encender.
const msgEncender = "Encender() error = %v, want %v"

// stubPower registra la máquina que se intentó encender.
type stubPower struct {
	encendidas []string
	err        error
}

// Start implementa ports.Power.
func (s *stubPower) Start(_ context.Context, target domain.Target) error {
	s.encendidas = append(s.encendidas, target.ID)

	return s.err
}

// TestVinculacionEnciendeLaMaquina comprueba el encendido de una
// máquina apagada y que se devuelva ya en ejecución.
func TestVinculacionEnciendeLaMaquina(t *testing.T) {
	t.Parallel()

	fuente := &stubPower{}
	vinculacion := NuevoVinculacion(&ConfigVinculacion{
		Inventario: inventarioDePrueba(),
		Seleccion:  &stubSeleccion{},
		Energia:    fuente,
	})

	maquina, err := vinculacion.Encender(context.Background(), nombreUbuntu)
	if err != nil {
		t.Fatalf("Encender() error = %v, want nil", err)
	}

	if len(fuente.encendidas) != 1 || fuente.encendidas[0] != nombreUbuntu {
		t.Errorf("encendidas = %v, want [%q]", fuente.encendidas, nombreUbuntu)
	}

	if maquina.Estado != domain.StateRunning {
		t.Errorf("Estado = %q, want %q", maquina.Estado, domain.StateRunning)
	}
}

// TestVinculacionNoEnciendeUnaMaquinaEnMarcha evita un 200 engañoso.
func TestVinculacionNoEnciendeUnaMaquinaEnMarcha(t *testing.T) {
	t.Parallel()

	fuente := &stubPower{}
	vinculacion := NuevoVinculacion(&ConfigVinculacion{
		Inventario: inventarioDePrueba(),
		Seleccion:  &stubSeleccion{},
		Energia:    fuente,
	})

	_, err := vinculacion.Encender(context.Background(), nombreDebian)

	if !errors.Is(err, ErrMaquinaEncendida) {
		t.Fatalf(msgEncender, err, ErrMaquinaEncendida)
	}

	if len(fuente.encendidas) != 0 {
		t.Errorf("encendidas = %v, want ninguna", fuente.encendidas)
	}
}

// TestVinculacionNoEnciendeUnaMaquinaDesconocida comprueba el 404.
func TestVinculacionNoEnciendeUnaMaquinaDesconocida(t *testing.T) {
	t.Parallel()

	vinculacion := NuevoVinculacion(&ConfigVinculacion{
		Inventario: inventarioDePrueba(),
		Seleccion:  &stubSeleccion{},
		Energia:    &stubPower{},
	})

	if _, err := vinculacion.Encender(context.Background(), "fantasma"); !errors.Is(err, ErrMaquinaNoEncontrada) {
		t.Fatalf(msgEncender, err, ErrMaquinaNoEncontrada)
	}
}

// TestVinculacionPropagaElFalloDelEncendido conserva la causa.
func TestVinculacionPropagaElFalloDelEncendido(t *testing.T) {
	t.Parallel()

	causa := errors.New("VirtualBox rechazó el arranque")
	vinculacion := NuevoVinculacion(&ConfigVinculacion{
		Inventario: inventarioDePrueba(),
		Seleccion:  &stubSeleccion{},
		Energia:    &stubPower{err: causa},
	})

	if _, err := vinculacion.Encender(context.Background(), nombreUbuntu); !errors.Is(err, causa) {
		t.Fatalf(msgEncender, err, causa)
	}
}

// TestVinculacionOcultaSinGuestAdditions comprueba que una máquina en
// ejecución sin Guest Additions no se lista y que la apagada sí.
func TestVinculacionOcultaSinGuestAdditions(t *testing.T) {
	t.Parallel()

	maquina := &stubGuestAdditions{accesibles: map[string]bool{}}
	vinculacion := NuevoVinculacion(&ConfigVinculacion{
		Inventario: inventarioDePrueba(),
		Seleccion:  &stubSeleccion{},
		Maquina:    maquina,
	})

	visibles, err := vinculacion.Listar(context.Background(), "")
	if err != nil {
		t.Fatalf(msgListarOK, err)
	}

	if len(visibles) != 1 || visibles[0].Nombre != nombreUbuntu {
		t.Errorf("visibles = %+v, want solo la máquina apagada", visibles)
	}

	if maquina.consultas != 1 {
		t.Errorf("Address() se llamó %d veces, want 1 (solo la encendida)", maquina.consultas)
	}
}

// TestVinculacionListaConGuestAdditions comprueba que la máquina
// encendida con Guest Additions sí aparece.
func TestVinculacionListaConGuestAdditions(t *testing.T) {
	t.Parallel()

	maquina := &stubGuestAdditions{accesibles: map[string]bool{nombreDebian: true}}
	vinculacion := NuevoVinculacion(&ConfigVinculacion{
		Inventario: inventarioDePrueba(),
		Seleccion:  &stubSeleccion{},
		Maquina:    maquina,
	})

	visibles, err := vinculacion.Listar(context.Background(), "")
	if err != nil {
		t.Fatalf(msgListarOK, err)
	}

	if len(visibles) != 2 {
		t.Errorf("visibles = %+v, want las dos máquinas", visibles)
	}
}
