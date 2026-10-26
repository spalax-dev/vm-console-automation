// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"sync"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// ErrSessionUnavailable indica que ningún conector puede atender la
// máquina en este momento (por ejemplo, la VM está apagada o no hay
// llave SSH instalada ni Guest Additions disponible).
var ErrSessionUnavailable = errors.New("no hay una sesión disponible con la máquina")

// ErrNoMachine indica que la consola todavía no está vinculada con
// ninguna máquina del inventario. Se envuelve con ErrSessionUnavailable
// para que el manejador responda 503 con un mensaje propio.
var ErrNoMachine = errors.New("no hay una máquina vinculada")

// SessionInfo describe el resultado del descubrimiento: el conector
// elegido, la dirección de la máquina y su estado de encendido.
type SessionInfo struct {
	// Connector es el nombre publicado por el conector seleccionado.
	Connector string
	// IP es la dirección descubierta de la máquina.
	IP netip.Addr
	// State es el estado de encendido observado.
	State domain.MachineState
}

// DiscovererConfig reúne las dependencias del descubridor de sesión.
type DiscovererConfig struct {
	// Machine consulta el estado y la dirección de la VM.
	Machine ports.Machine
	// CurrentTarget identifica la máquina que se va a administrar.
	CurrentTarget domain.Target
	// Connectors son los conectores candidatos, en orden de preferencia.
	Connectors []ports.Connector
	// Registration informa si una máquina ya fue preparada para SSH. Una
	// máquina registrada no vuelve a usar Guest Additions.
	Registration Registration
}

// Discoverer elige, en tiempo de ejecución, el conector con el que la
// consola puede administrar la máquina virtual.
type Discoverer struct {
	machine      ports.Machine
	connectors   []ports.Connector
	registration Registration

	// mu protege objetivo: los manejadores HTTP consultan la sesión y
	// vinculan una máquina distinta en paralelo.
	mu     sync.RWMutex
	target domain.Target
}

// NewDiscoverer construye un descubridor con las dependencias dadas.
func NewDiscoverer(config *DiscovererConfig) *Discoverer {
	return &Discoverer{
		machine:      config.Machine,
		target:       config.CurrentTarget,
		connectors:   config.Connectors,
		registration: config.Registration,
	}
}

// CurrentTarget devuelve la máquina con la que trabaja la consola.
func (d *Discoverer) CurrentTarget() domain.Target {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.target
}

// Set vincula la consola con la máquina indicada. La dirección
// descubierta se descarta porque pertenece a la máquina anterior.
func (d *Discoverer) Set(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.target = domain.Target{ID: name}
}

// Discover resuelve el conector disponible para la máquina. Verifica
// primero que la máquina esté en ejecución, descubre su dirección si
// falta y sondea los conectores en orden de preferencia. Devuelve
// ErrSessionUnavailable cuando la máquina no puede atenderse o ningún
// conector supera el sondeo.
func (d *Discoverer) Discover(ctx context.Context) (SessionInfo, error) {
	connector, target, state, err := d.resolve(ctx)
	if err != nil {
		return SessionInfo{}, err
	}

	return SessionInfo{
		Connector: connector.Name(),
		IP:        target.Address,
		State:     state,
	}, nil
}

// Open implementa SessionOpener: resuelve el conector disponible y
// establece una sesión con la máquina. La sesión queda a cargo del
// llamador, que debe cerrarla.
func (d *Discoverer) Open(ctx context.Context) (ports.Session, error) {
	connector, target, _, err := d.resolve(ctx)
	if err != nil {
		return nil, err
	}

	session, err := connector.Connect(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("abrir la sesión con la máquina %q: %w", target.ID, err)
	}

	return session, nil
}

// resolve verifica el estado de la máquina, completa la dirección y
// elige el conector disponible.
func (d *Discoverer) resolve(ctx context.Context) (ports.Connector, domain.Target, domain.MachineState, error) {
	target := d.CurrentTarget()

	if target.ID == "" {
		return nil, domain.Target{}, domain.StateUnknown, fmt.Errorf(
			"%w: %w, elige una del inventario", ErrSessionUnavailable, ErrNoMachine,
		)
	}

	state, err := d.machine.State(ctx, target)
	if err != nil {
		return nil, domain.Target{}, domain.StateUnknown, fmt.Errorf(msgStateMachine, err)
	}

	if state != domain.StateRunning {
		return nil, domain.Target{}, state, fmt.Errorf(
			"%w: la máquina %q está en estado %q", ErrSessionUnavailable, target.ID, state,
		)
	}

	target, err = d.withAddress(ctx, target)
	if err != nil {
		return nil, domain.Target{}, state, fmt.Errorf("%w: %w", ErrSessionUnavailable, err)
	}

	connector := d.firstAvailableConnector(ctx, target)
	if connector == nil {
		return nil, domain.Target{}, state, fmt.Errorf(
			"%w: ningún conector pudo atender la máquina %q", ErrSessionUnavailable, target.ID,
		)
	}

	return connector, target, state, nil
}

// withAddress devuelve el objetivo con una dirección válida, ya sea la
// que traía o la descubierta con el puerto Machine.
func (d *Discoverer) withAddress(ctx context.Context, target domain.Target) (domain.Target, error) {
	if target.Address.IsValid() {
		return target, nil
	}

	address, err := d.machine.Address(ctx, target)
	if err != nil {
		return domain.Target{}, fmt.Errorf("descubrir la dirección de la máquina: %w", err)
	}

	target.Address = address

	return target, nil
}

// firstAvailableConnector devuelve el primer conector cuyo sondeo
// termina sin error, o nil si ninguno puede atender la máquina. En una
// máquina ya registrada no se recurre al conector de respaldo: después
// de instalar SSH, la comunicación debe ser exclusivamente por SSH.
func (d *Discoverer) firstAvailableConnector(ctx context.Context, target domain.Target) ports.Connector {
	registered := d.registration != nil && d.registration.IsRegistered(target.ID)

	for index, connector := range d.connectors {
		if index > 0 && registered {
			break
		}

		if err := connector.Probe(ctx, target); err == nil {
			return connector
		}
	}

	return nil
}
