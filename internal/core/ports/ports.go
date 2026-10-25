// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package ports declara las interfaces (puertos) que el núcleo de la
// consola necesita del exterior. Los adaptadores de internal/adapters
// las implementan; el núcleo no conoce ninguna implementación concreta.
package ports

import (
	"context"
	"net/netip"

	"consola-vm/internal/core/domain"
)

// Executor ejecuta un programa en el sistema operativo donde corre la
// consola, capturando su salida y su código de salida.
type Executor interface {
	Execute(ctx context.Context, program string, args ...string) (domain.Result, error)
}

// Machine expone información de la máquina administrada sin establecer
// sesión con ella.
type Machine interface {
	// Address descubre la dirección IP de la máquina.
	Address(ctx context.Context, target domain.Target) (netip.Addr, error)
	// State informa el estado de encendido de la máquina.
	State(ctx context.Context, target domain.Target) (domain.MachineState, error)
}

// Inventory enumera las máquinas virtuales registradas en el
// hipervisor, para que la consola pueda elegir con cuál trabajar.
type Inventory interface {
	// List devuelve todas las máquinas registradas con su estado.
	List(ctx context.Context) ([]domain.VM, error)
}

// Connector establece sesiones con la máquina administrada. Cada
// adaptador publica un nombre propio para que los servicios puedan
// distinguirlos.
type Connector interface {
	// Name identifica al conector (por ejemplo, "ssh").
	Name() string
	// Probe comprueba si el conector puede atender la máquina ahora.
	Probe(ctx context.Context, target domain.Target) error
	// Connect establece una sesión con la máquina.
	Connect(ctx context.Context, target domain.Target) (Session, error)
}

// Session representa una sesión establecida con la máquina, sobre la
// cual se ejecutan órdenes y se transfieren archivos.
type Session interface {
	// Run ejecuta una orden dentro de la máquina.
	Run(ctx context.Context, cmd *domain.Command) (domain.Result, error)
	// Put transfiere un archivo de la consola a la máquina.
	Put(ctx context.Context, localPath, remotePath string) error
	// Close libera los recursos de la sesión.
	Close() error
}
