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
	uuidTest        = "8f3a1c2e-1111-4a4a-9a9a-000000000001"
	uuidUbuntu      = "8f3a1c2e-2222-4a4a-9a9a-000000000002"
	testMachineName = "vm-1"
	nameUbuntu      = "ubuntu-lab"
)

// stubInventory devuelve un inventario preprogramado.
type stubInventory struct {
	machines []domain.VM
	err      error
}

// List implementa ports.Inventory.
func (s *stubInventory) List(context.Context) ([]domain.VM, error) {
	return s.machines, s.err
}

// stubSelection imita el objetivo actual de la consola.
type stubSelection struct {
	id string
}

// CurrentTarget devuelve la máquina vinculada.
func (s *stubSelection) CurrentTarget() domain.Target { return domain.Target{ID: s.id} }

// Set vincula una máquina por nombre o UUID.
func (s *stubSelection) Set(id string) { s.id = id }

// stubGuestAdditions simula la comprobación de Guest Additions.
type stubGuestAdditions struct {
	accesibles map[string]bool
	networks   map[string]domain.NetworkMode
	consultas  int
	errNetwork error
}

// Address devuelve una IP solo si la máquina publica Guest Additions.
func (s *stubGuestAdditions) Address(_ context.Context, target domain.Target) (netip.Addr, error) {
	s.consultas++

	if s.accesibles[target.ID] {
		return netip.MustParseAddr(ipTests), nil
	}

	return netip.Addr{}, errors.New("sin guest additions")
}

// State devuelve el estado en ejecución.
func (s *stubGuestAdditions) State(context.Context, domain.Target) (domain.MachineState, error) {
	return domain.StateRunning, nil
}

// GuestAdditions no informa datos en esta prueba.
func (s *stubGuestAdditions) GuestAdditions(context.Context, domain.Target) (domain.GuestAdditions, error) {
	return domain.GuestAdditions{}, nil
}

// Network devuelve el modo de red preprogramado por máquina («bridged»
// por omisión).
func (s *stubGuestAdditions) Network(_ context.Context, target domain.Target) (domain.NetworkMode, error) {
	if s.errNetwork != nil {
		return domain.NetworkUnknown, s.errNetwork
	}

	if mode, ok := s.networks[target.ID]; ok {
		return mode, nil
	}

	return domain.NetworkBridged, nil
}

// inventoryOfTests arma dos máquinas, una en ejecución.
func inventoryOfTests() *stubInventory {
	return &stubInventory{machines: []domain.VM{
		{Name: testMachineName, UUID: uuidTest, State: domain.StateRunning},
		{Name: nameUbuntu, UUID: uuidUbuntu, State: domain.StatePoweredOff},
	}}
}

// TestBindingListsMarksActive comprueba que el inventario señala
// la máquina ya vinculada.
func TestBindingListsMarksActive(t *testing.T) {
	t.Parallel()

	selection := &stubSelection{id: uuidTest}
	binding := NewBinding(&BindingConfig{Inventory: inventoryOfTests(), Selection: selection})

	machines, err := binding.List(context.Background(), "")
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	if len(machines) != 2 {
		t.Fatalf("len(maquinas) = %d, want 2", len(machines))
	}

	if !machines[0].Active {
		t.Error("maquinas[0].Activa = false, want true")
	}

	if machines[1].Active {
		t.Error("maquinas[1].Activa = true, want false")
	}
}

// TestBindingFiltraByNameYUUID comprueba la búsqueda del selector.
func TestBindingFiltraByNameYUUID(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"por nombre":       "ubuntu",
		"por mayúsculas":   "VM-1",
		"por UUID parcial": "2222",
	}

	for name, filter := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			binding := NewBinding(&BindingConfig{
				Inventory: inventoryOfTests(),
				Selection: &stubSelection{},
			})

			machines, err := binding.List(context.Background(), filter)
			if err != nil {
				t.Fatalf(msgListOK, err)
			}

			if len(machines) != 1 {
				t.Fatalf("len(maquinas) = %d, want 1", len(machines))
			}

			if filter == "2222" && machines[0].Name != nameUbuntu {
				t.Errorf("maquinas[0].Nombre = %q, want %q", machines[0].Name, nameUbuntu)
			}
		})
	}
}

// TestBindingAcceptsNameYUUID comprueba las dos formas de
// identificar la máquina.
func TestBindingAcceptsNameYUUID(t *testing.T) {
	t.Parallel()

	for _, identifier := range []string{nameUbuntu, uuidUbuntu} {
		selection := &stubSelection{}
		binding := NewBinding(&BindingConfig{Inventory: inventoryOfTests(), Selection: selection})

		machine, err := binding.Bind(context.Background(), identifier)
		if err != nil {
			t.Fatalf("Vincular(%q) error = %v, want nil", identifier, err)
		}

		if selection.id != nameUbuntu {
			t.Errorf("objetivo = %q, want %q", selection.id, nameUbuntu)
		}

		if !machine.Active {
			t.Errorf("Vincular(%q).Activa = false, want true", identifier)
		}

		if machine.Name != nameUbuntu {
			t.Errorf("Vincular(%q).Nombre = %q, want %q", identifier, machine.Name, nameUbuntu)
		}
	}
}

// TestBindingRejectsMachineUnknown comprueba el error de
// búsqueda sin coincidencias.
func TestBindingRejectsMachineUnknown(t *testing.T) {
	t.Parallel()

	selection := &stubSelection{}
	binding := NewBinding(&BindingConfig{Inventory: inventoryOfTests(), Selection: selection})

	_, err := binding.Bind(context.Background(), "no-existe")

	if !errors.Is(err, ErrMachineNotFound) {
		t.Fatalf("Vincular() error = %v, want %v", err, ErrMachineNotFound)
	}

	if selection.id != "" {
		t.Errorf("objetivo = %q, want vacío", selection.id)
	}
}

// TestBindingPropagatesFailureOfInventory conserva la causa del
// error del hipervisor.
func TestBindingPropagatesFailureOfInventory(t *testing.T) {
	t.Parallel()

	causa := errors.New("VBoxManage no está disponible")
	binding := NewBinding(&BindingConfig{
		Inventory: &stubInventory{err: causa},
		Selection: &stubSelection{},
	})

	if _, err := binding.List(context.Background(), ""); !errors.Is(err, causa) {
		t.Fatalf("Listar() error = %v, want %v", err, causa)
	}
}

// TestBindingActiveReturnsTarget comprueba el nombre de la
// máquina vinculada que se publica en la sesión.
func TestBindingActiveReturnsTarget(t *testing.T) {
	t.Parallel()

	binding := NewBinding(&BindingConfig{
		Inventory: inventoryOfTests(),
		Selection: &stubSelection{id: testMachineName},
	})

	if active := binding.Active(); active != testMachineName {
		t.Errorf("Activa() = %q, want %q", active, testMachineName)
	}
}

// msgPowerOn es el formato de los fallos de PowerOn.
const msgPowerOn = "Encender() error = %v, want %v"

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

// TestBindingPowersOnMachine comprueba el encendido de una
// máquina apagada y que se devuelva ya en ejecución.
func TestBindingPowersOnMachine(t *testing.T) {
	t.Parallel()

	fuente := &stubPower{}
	binding := NewBinding(&BindingConfig{
		Inventory: inventoryOfTests(),
		Selection: &stubSelection{},
		Power:     fuente,
	})

	machine, err := binding.PowerOn(context.Background(), nameUbuntu)
	if err != nil {
		t.Fatalf("Encender() error = %v, want nil", err)
	}

	if len(fuente.encendidas) != 1 || fuente.encendidas[0] != nameUbuntu {
		t.Errorf("encendidas = %v, want [%q]", fuente.encendidas, nameUbuntu)
	}

	if machine.State != domain.StateRunning {
		t.Errorf("Estado = %q, want %q", machine.State, domain.StateRunning)
	}
}

// TestBindingNotPowersOnMachineInRunning evita un 200 engañoso.
func TestBindingNotPowersOnMachineInRunning(t *testing.T) {
	t.Parallel()

	fuente := &stubPower{}
	binding := NewBinding(&BindingConfig{
		Inventory: inventoryOfTests(),
		Selection: &stubSelection{},
		Power:     fuente,
	})

	_, err := binding.PowerOn(context.Background(), testMachineName)

	if !errors.Is(err, ErrMachineRunning) {
		t.Fatalf(msgPowerOn, err, ErrMachineRunning)
	}

	if len(fuente.encendidas) != 0 {
		t.Errorf("encendidas = %v, want ninguna", fuente.encendidas)
	}
}

// TestBindingNotPowersOnMachineUnknown comprueba el 404.
func TestBindingNotPowersOnMachineUnknown(t *testing.T) {
	t.Parallel()

	binding := NewBinding(&BindingConfig{
		Inventory: inventoryOfTests(),
		Selection: &stubSelection{},
		Power:     &stubPower{},
	})

	if _, err := binding.PowerOn(context.Background(), "fantasma"); !errors.Is(err, ErrMachineNotFound) {
		t.Fatalf(msgPowerOn, err, ErrMachineNotFound)
	}
}

// TestBindingPropagatesFailureOfPowerOn conserva la causa.
func TestBindingPropagatesFailureOfPowerOn(t *testing.T) {
	t.Parallel()

	causa := errors.New("VirtualBox rechazó el arranque")
	binding := NewBinding(&BindingConfig{
		Inventory: inventoryOfTests(),
		Selection: &stubSelection{},
		Power:     &stubPower{err: causa},
	})

	if _, err := binding.PowerOn(context.Background(), nameUbuntu); !errors.Is(err, causa) {
		t.Fatalf(msgPowerOn, err, causa)
	}
}

// TestBindingMarksWithoutGuestAdditions comprueba que una máquina en
// ejecución sin Guest Additions se lista con el aviso y la apagada como
// administrable.
func TestBindingMarksWithoutGuestAdditions(t *testing.T) {
	t.Parallel()

	machine := &stubGuestAdditions{accesibles: map[string]bool{}}
	binding := NewBinding(&BindingConfig{
		Inventory: inventoryOfTests(),
		Selection: &stubSelection{},
		Machine:   machine,
	})

	visibles, err := binding.List(context.Background(), "")
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	if len(visibles) != 2 {
		t.Fatalf("visibles = %+v, want las dos máquinas", visibles)
	}

	porNombre := map[string]VirtualMachine{}
	for _, visible := range visibles {
		porNombre[visible.Name] = visible
	}

	if porNombre[testMachineName].Administrable {
		t.Error("la máquina encendida sin Guest Additions no debería ser administrable")
	}

	if porNombre[testMachineName].Warning == "" {
		t.Error("falta el aviso de Guest Additions")
	}

	if !porNombre[nameUbuntu].Administrable {
		t.Error("la máquina apagada debería ser administrable")
	}

	if machine.consultas != 1 {
		t.Errorf("Address() se llamó %d veces, want 1 (solo la encendida)", machine.consultas)
	}
}

// TestBindingListsRegisteredWithoutGuestAdditions comprueba que una máquina
// registrada se lista aunque Guest Additions no publique la dirección.
func TestBindingListsRegisteredWithoutGuestAdditions(t *testing.T) {
	t.Parallel()

	machine := &stubGuestAdditions{accesibles: map[string]bool{}}
	binding := NewBinding(&BindingConfig{
		Inventory:    inventoryOfTests(),
		Selection:    &stubSelection{},
		Machine:      machine,
		Registration: &stubRegistration{registradas: map[string]bool{testMachineName: true}},
	})

	visibles, err := binding.List(context.Background(), "")
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	if len(visibles) != 2 {
		t.Errorf("visibles = %+v, want las dos máquinas", visibles)
	}

	if machine.consultas != 0 {
		t.Errorf("Address() se llamó %d veces, want 0 (la registrada usa SSH)", machine.consultas)
	}
}

// TestBindingListsWithGuestAdditions comprueba que la máquina
// encendida con Guest Additions sí aparece.
func TestBindingListsWithGuestAdditions(t *testing.T) {
	t.Parallel()

	machine := &stubGuestAdditions{accesibles: map[string]bool{testMachineName: true}}
	binding := NewBinding(&BindingConfig{
		Inventory: inventoryOfTests(),
		Selection: &stubSelection{},
		Machine:   machine,
	})

	visibles, err := binding.List(context.Background(), "")
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	if len(visibles) != 2 {
		t.Errorf("visibles = %+v, want las dos máquinas", visibles)
	}
}

// TestBindingOcultaNAT comprueba que una máquina con Guest Additions
// pero en NAT no se lista, porque el anfitrión no alcanza su IP.
func TestBindingOcultaNAT(t *testing.T) {
	t.Parallel()

	machine := &stubGuestAdditions{
		accesibles: map[string]bool{testMachineName: true},
		networks:   map[string]domain.NetworkMode{testMachineName: domain.NetworkNAT},
	}
	binding := NewBinding(&BindingConfig{
		Inventory: inventoryOfTests(),
		Selection: &stubSelection{},
		Machine:   machine,
	})

	visibles, err := binding.List(context.Background(), "")
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	if len(visibles) != 1 || visibles[0].Name != nameUbuntu {
		t.Errorf("visibles = %+v, want solo la máquina apagada", visibles)
	}
}

// stubRegistration informa qué máquinas están registradas.
type stubRegistration struct {
	registradas map[string]bool
}

// IsRegistered implementa la consulta de registro.
func (s *stubRegistration) IsRegistered(vm string) bool {
	return s.registradas[vm]
}

// TestBindingMarksRegistradas comprueba la marca de registro.
func TestBindingMarksRegistradas(t *testing.T) {
	t.Parallel()

	binding := NewBinding(&BindingConfig{
		Inventory:    inventoryOfTests(),
		Selection:    &stubSelection{},
		Machine:      &stubGuestAdditions{accesibles: map[string]bool{testMachineName: true}},
		Registration: &stubRegistration{registradas: map[string]bool{testMachineName: true}},
	})

	visibles, err := binding.List(context.Background(), "")
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	if !visibles[0].IsRegistered {
		t.Error("vm-1 no aparece registrada")
	}

	if visibles[1].IsRegistered {
		t.Error("ubuntu-lab aparece registrada sin estarlo")
	}
}

// stubPersistence registra las máquinas vinculadas.
type stubPersistence struct {
	active    string
	guardadas []string
	err       error
}

// Active devuelve la máquina guardada.
func (s *stubPersistence) Active() string { return s.active }

// SaveActive acumula el nombre vinculado.
func (s *stubPersistence) SaveActive(vm string) error {
	s.guardadas = append(s.guardadas, vm)

	return s.err
}

// TestBindingSavesMachineActive comprueba que vincular persiste
// la elección.
func TestBindingSavesMachineActive(t *testing.T) {
	t.Parallel()

	persistence := &stubPersistence{}
	binding := NewBinding(&BindingConfig{
		Inventory:   inventoryOfTests(),
		Selection:   &stubSelection{},
		Persistence: persistence,
	})

	if _, err := binding.Bind(context.Background(), testMachineName); err != nil {
		t.Fatalf("Vincular() error = %v", err)
	}

	if len(persistence.guardadas) != 1 || persistence.guardadas[0] != testMachineName {
		t.Errorf("guardadas = %v, want [%s]", persistence.guardadas, testMachineName)
	}
}
