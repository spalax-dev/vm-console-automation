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

// ErrSesionNoDisponible indica que ningún conector puede atender la
// máquina en este momento (por ejemplo, la VM está apagada o no hay
// llave SSH instalada ni Guest Additions disponible).
var ErrSesionNoDisponible = errors.New("no hay una sesión disponible con la máquina")

// ErrSinMaquina indica que la consola todavía no está vinculada con
// ninguna máquina del inventario. Se envuelve con ErrSesionNoDisponible
// para que el manejador responda 503 con un mensaje propio.
var ErrSinMaquina = errors.New("no hay una máquina vinculada")

// InfoSesion describe el resultado del descubrimiento: el conector
// elegido, la dirección de la máquina y su estado de encendido.
type InfoSesion struct {
	// Conector es el nombre publicado por el conector seleccionado.
	Conector string
	// IP es la dirección descubierta de la máquina.
	IP netip.Addr
	// Estado es el estado de encendido observado.
	Estado domain.MachineState
}

// ConfigDescubridor reúne las dependencias del descubridor de sesión.
type ConfigDescubridor struct {
	// Maquina consulta el estado y la dirección de la VM.
	Maquina ports.Machine
	// Objetivo identifica la máquina que se va a administrar.
	Objetivo domain.Target
	// Conectores son los conectores candidatos, en orden de preferencia.
	Conectores []ports.Connector
}

// Descubridor elige, en tiempo de ejecución, el conector con el que la
// consola puede administrar la máquina virtual.
type Descubridor struct {
	maquina    ports.Machine
	conectores []ports.Connector

	// mu protege objetivo: los manejadores HTTP consultan la sesión y
	// vinculan una máquina distinta en paralelo.
	mu       sync.RWMutex
	objetivo domain.Target
}

// NuevoDescubridor construye un descubridor con las dependencias dadas.
func NuevoDescubridor(config *ConfigDescubridor) *Descubridor {
	return &Descubridor{
		maquina:    config.Maquina,
		objetivo:   config.Objetivo,
		conectores: config.Conectores,
	}
}

// Objetivo devuelve la máquina con la que trabaja la consola.
func (d *Descubridor) Objetivo() domain.Target {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.objetivo
}

// Establecer vincula la consola con la máquina indicada. La dirección
// descubierta se descarta porque pertenece a la máquina anterior.
func (d *Descubridor) Establecer(nombre string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.objetivo = domain.Target{ID: nombre}
}

// Descubrir resuelve el conector disponible para la máquina. Verifica
// primero que la máquina esté en ejecución, descubre su dirección si
// falta y sondea los conectores en orden de preferencia. Devuelve
// ErrSesionNoDisponible cuando la máquina no puede atenderse o ningún
// conector supera el sondeo.
func (d *Descubridor) Descubrir(ctx context.Context) (InfoSesion, error) {
	conector, objetivo, estado, err := d.resolver(ctx)
	if err != nil {
		return InfoSesion{}, err
	}

	return InfoSesion{
		Conector: conector.Name(),
		IP:       objetivo.Address,
		Estado:   estado,
	}, nil
}

// Abrir implementa AbridorSesion: resuelve el conector disponible y
// establece una sesión con la máquina. La sesión queda a cargo del
// llamador, que debe cerrarla.
func (d *Descubridor) Abrir(ctx context.Context) (ports.Session, error) {
	conector, objetivo, _, err := d.resolver(ctx)
	if err != nil {
		return nil, err
	}

	sesion, err := conector.Connect(ctx, objetivo)
	if err != nil {
		return nil, fmt.Errorf("abrir la sesión con la máquina %q: %w", objetivo.ID, err)
	}

	return sesion, nil
}

// resolver verifica el estado de la máquina, completa la dirección y
// elige el conector disponible.
func (d *Descubridor) resolver(ctx context.Context) (ports.Connector, domain.Target, domain.MachineState, error) {
	objetivo := d.Objetivo()

	if objetivo.ID == "" {
		return nil, domain.Target{}, domain.StateUnknown, fmt.Errorf(
			"%w: %w, elige una del inventario", ErrSesionNoDisponible, ErrSinMaquina,
		)
	}

	estado, err := d.maquina.State(ctx, objetivo)
	if err != nil {
		return nil, domain.Target{}, domain.StateUnknown, fmt.Errorf(msgEstadoMaquina, err)
	}

	if estado != domain.StateRunning {
		return nil, domain.Target{}, estado, fmt.Errorf(
			"%w: la máquina %q está en estado %q", ErrSesionNoDisponible, objetivo.ID, estado,
		)
	}

	objetivo, err = d.conDireccion(ctx, objetivo)
	if err != nil {
		return nil, domain.Target{}, estado, err
	}

	conector := d.primerConectorDisponible(ctx, objetivo)
	if conector == nil {
		return nil, domain.Target{}, estado, fmt.Errorf(
			"%w: ningún conector pudo atender la máquina %q", ErrSesionNoDisponible, objetivo.ID,
		)
	}

	return conector, objetivo, estado, nil
}

// conDireccion devuelve el objetivo con una dirección válida, ya sea la
// que traía o la descubierta con el puerto Machine.
func (d *Descubridor) conDireccion(ctx context.Context, objetivo domain.Target) (domain.Target, error) {
	if objetivo.Address.IsValid() {
		return objetivo, nil
	}

	direccion, err := d.maquina.Address(ctx, objetivo)
	if err != nil {
		return domain.Target{}, fmt.Errorf("descubrir la dirección de la máquina: %w", err)
	}

	objetivo.Address = direccion

	return objetivo, nil
}

// primerConectorDisponible devuelve el primer conector cuyo sondeo
// termina sin error, o nil si ninguno puede atender la máquina.
func (d *Descubridor) primerConectorDisponible(ctx context.Context, objetivo domain.Target) ports.Connector {
	for _, conector := range d.conectores {
		if err := conector.Probe(ctx, objetivo); err == nil {
			return conector
		}
	}

	return nil
}
