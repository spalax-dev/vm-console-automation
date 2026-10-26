// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// ErrMachineNotFound indica que el identificador recibido no
// corresponde a ninguna máquina del inventario del hipervisor.
var ErrMachineNotFound = errors.New("la máquina virtual no está en el inventario")

// ErrMachineRunning indica que se pidió encender una máquina que ya
// está en ejecución.
var ErrMachineRunning = errors.New("la máquina virtual ya está encendida")

// Mensajes de error reutilizados por las operaciones del inventario.
const (
	msgListInventory = "listar las máquinas virtuales: %w"
	msgIdentifier    = "%w: %q"
)

// VirtualMachine es una máquina del inventario tal como la publica la
// consola: el nombre, el UUID, su estado y si es la vinculada.
type VirtualMachine struct {
	// Name es el nombre con el que el hipervisor registra la máquina.
	Name string
	// UUID es el identificador único de la máquina.
	UUID string
	// State es el estado de encendido observado.
	State domain.MachineState
	// Active indica que la consola está vinculada con esta máquina.
	Active bool
	// IsRegistered indica que la consola ya preparó SSH en la máquina.
	IsRegistered bool
}

// Selector es la máquina con la que trabaja la consola. El descubridor
// de sesión lo implementa: vincular es cambiar su objetivo.
type Selector interface {
	// CurrentTarget devuelve la máquina vinculada.
	CurrentTarget() domain.Target
	// Set vincula la máquina indicada por su nombre.
	Set(name string)
}

// Registration informa si una máquina ya fue registrada en la consola.
type Registration interface {
	IsRegistered(vm string) bool
}

// BindingConfig reúne las dependencias de la vinculación.
type BindingConfig struct {
	// Inventory enumera las máquinas del hipervisor.
	Inventory ports.Inventory
	// Selection es la máquina con la que trabaja la consola.
	Selection Selector
	// Power enciende las máquinas del hipervisor.
	Power ports.Power
	// Machine comprueba si una máquina en ejecución publica Guest
	// Additions y, por tanto, puede administrarse.
	Machine ports.Machine
	// Registration consulta el estado de registro de las máquinas.
	Registration Registration
	// Persistence guarda la máquina vinculada entre arranques.
	Persistence ActivePersistence
}

// ActivePersistence guarda y recupera la máquina vinculada.
type ActivePersistence interface {
	Active() string
	SaveActive(vm string) error
}

// Binding busca máquinas en el inventario y cambia la que administra
// la consola.
type Binding struct {
	inventory    ports.Inventory
	selection    Selector
	power        ports.Power
	machine      ports.Machine
	registration Registration
	persistence  ActivePersistence
}

// NewBinding construye el servicio con las dependencias dadas.
func NewBinding(config *BindingConfig) *Binding {
	return &Binding{
		inventory:    config.Inventory,
		selection:    config.Selection,
		power:        config.Power,
		machine:      config.Machine,
		registration: config.Registration,
		persistence:  config.Persistence,
	}
}

// List devuelve las máquinas del inventario marcando la vinculada. Si
// filtro no está vacío, se compara, sin distinguir mayúsculas, contra el
// nombre y el UUID.
func (v *Binding) List(ctx context.Context, filter string) ([]VirtualMachine, error) {
	machines, err := v.inventory.List(ctx)
	if err != nil {
		return nil, fmt.Errorf(msgListInventory, err)
	}

	target := v.selection.CurrentTarget()
	text := strings.ToLower(strings.TrimSpace(filter))
	visibles := make([]VirtualMachine, 0, len(machines))

	for _, machine := range machines {
		if text != "" && !matches(machine, text) {
			continue
		}

		if !v.administrable(ctx, machine) {
			continue
		}

		visibles = append(visibles, VirtualMachine{
			Name:         machine.Name,
			UUID:         machine.UUID,
			State:        machine.State,
			Active:       machine.Name == target.ID || machine.UUID == target.ID,
			IsRegistered: v.estaRegistered(machine.Name),
		})
	}

	return visibles, nil
}

// administrable informa si la máquina tiene una red alcanzable desde el
// anfitrión. Las apagadas se listan para poder encenderlas y las encendidas
// se mantienen visibles aunque Guest Additions todavía no publique la
// dirección: mientras arrancan, el selector debe permitir diagnosticarlas o
// registrarlas.
func (v *Binding) administrable(ctx context.Context, machine domain.VM) bool {
	if v.machine == nil {
		return true
	}

	target := domain.Target{ID: machine.Name}

	mode, err := v.machine.Network(ctx, target)
	if err != nil {
		return false
	}

	switch mode {
	case domain.NetworkNAT, domain.NetworkInternal, domain.NetworkNone:
		return false
	case domain.NetworkBridged, domain.NetworkHostOnly:
	default:
		return false
	}

	return true
}

// Bind establece la máquina que administra la consola. Acepta el
// nombre o el UUID del inventario y guarda siempre el nombre, que es el
// identificador estable que usan los adaptadores. Bind una máquina
// apagada es válido: solo impide abrir sesión con ella.
func (v *Binding) Bind(ctx context.Context, identifier string) (VirtualMachine, error) {
	machines, err := v.inventory.List(ctx)
	if err != nil {
		return VirtualMachine{}, fmt.Errorf(msgListInventory, err)
	}

	buscado := strings.TrimSpace(identifier)

	for _, machine := range machines {
		if !strings.EqualFold(machine.Name, buscado) && !strings.EqualFold(machine.UUID, buscado) {
			continue
		}

		v.selection.Set(machine.Name)
		v.saveActive(machine.Name)

		return VirtualMachine{
			Name:         machine.Name,
			UUID:         machine.UUID,
			State:        machine.State,
			Active:       true,
			IsRegistered: v.estaRegistered(machine.Name),
		}, nil
	}

	return VirtualMachine{}, fmt.Errorf(msgIdentifier, ErrMachineNotFound, buscado)
}

// PowerOn arranca una máquina del inventario. Rechaza con
// ErrMachineRunning la que ya está en ejecución, para no informar un
// éxito que no ocurrió, y devuelve la máquina ya en estado «running».
func (v *Binding) PowerOn(ctx context.Context, identifier string) (VirtualMachine, error) {
	machine, err := v.find(ctx, identifier)
	if err != nil {
		return VirtualMachine{}, err
	}

	if machine.State == domain.StateRunning {
		return VirtualMachine{}, fmt.Errorf(msgIdentifier, ErrMachineRunning, machine.Name)
	}

	if err := v.power.Start(ctx, domain.Target{ID: machine.Name}); err != nil {
		return VirtualMachine{}, fmt.Errorf("encender la máquina %q: %w", machine.Name, err)
	}

	return VirtualMachine{
		Name:         machine.Name,
		UUID:         machine.UUID,
		State:        domain.StateRunning,
		Active:       v.selection.CurrentTarget().ID == machine.Name || v.selection.CurrentTarget().ID == machine.UUID,
		IsRegistered: v.estaRegistered(machine.Name),
	}, nil
}

// saveActive persiste la máquina vinculada sin romper la operación si
// el almacenamiento falla.
func (v *Binding) saveActive(name string) {
	if v.persistence == nil {
		return
	}

	if err := v.persistence.SaveActive(name); err != nil {
		slog.Warn("guardar la máquina vinculada", logErrorKey, err)
	}
}

// estaRegistered informa si la máquina ya fue registrada.
func (v *Binding) estaRegistered(name string) bool {
	return v.registration != nil && v.registration.IsRegistered(name)
}

// find devuelve la máquina del inventario que corresponde al
// identificador, por nombre o por UUID.
func (v *Binding) find(ctx context.Context, identifier string) (domain.VM, error) {
	machines, err := v.inventory.List(ctx)
	if err != nil {
		return domain.VM{}, fmt.Errorf(msgListInventory, err)
	}

	buscado := strings.TrimSpace(identifier)

	for _, machine := range machines {
		if strings.EqualFold(machine.Name, buscado) || strings.EqualFold(machine.UUID, buscado) {
			return machine, nil
		}
	}

	return domain.VM{}, fmt.Errorf(msgIdentifier, ErrMachineNotFound, buscado)
}

// Active devuelve el nombre de la máquina vinculada, o una cadena vacía
// si la consola todavía no está vinculada con ninguna.
func (v *Binding) Active() string {
	return v.selection.CurrentTarget().ID
}

// matches informa si la máquina contiene el texto de búsqueda en su
// nombre o en su UUID; el texto debe venir en minúsculas.
func matches(machine domain.VM, text string) bool {
	return strings.Contains(strings.ToLower(machine.Name), text) ||
		strings.Contains(strings.ToLower(machine.UUID), text)
}
