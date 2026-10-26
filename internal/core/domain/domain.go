// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package domain define los tipos del lenguaje de la consola: órdenes
// que se ejecutan en la máquina administrada, resultados de su
// ejecución, errores con código de salida, estados de la máquina virtual
// y el objetivo de administración. No conoce ninguna tecnología
// concreta: es el vocabulario compartido por los puertos y adaptadores.
package domain

import (
	"fmt"
	"net/netip"
)

// Result reúne la salida estándar, la salida de error y el código de
// salida de una orden ejecutada.
type Result struct {
	// Stdout es la salida estándar capturada.
	Stdout string
	// Stderr es la salida de error capturada.
	Stderr string
	// ExitCode es el código de salida del proceso.
	ExitCode int
}

// ExitError representa una orden que terminó con un código de salida
// distinto de cero. Permite que el núcleo distinga un fallo de la orden
// de un fallo del mecanismo de ejecución, sin conocer el adaptador.
type ExitError struct {
	// Code es el código de salida del proceso.
	Code int
	// Command es la orden que terminó con error.
	Command string
}

// Error implementa la interfaz error.
func (e *ExitError) Error() string {
	return fmt.Sprintf("la orden %q terminó con el código de salida %d", e.Command, e.Code)
}

// MachineState describe el estado de encendido de una máquina virtual.
type MachineState string

// Estados de encendido de una máquina virtual.
const (
	// StateRunning indica que la máquina está encendida.
	StateRunning MachineState = "running"
	// StatePoweredOff indica que la máquina está apagada.
	StatePoweredOff MachineState = "poweroff"
	// StatePaused indica que la máquina está pausada.
	StatePaused MachineState = "paused"
	// StateSaved indica que la máquina está guardada en disco.
	StateSaved MachineState = "saved"
	// StateUnknown indica que el estado no se pudo determinar.
	StateUnknown MachineState = "unknown"
)

// GuestAdditions describe lo que la máquina informa sobre Guest
// Additions.
type GuestAdditions struct {
	// Version es la versión instalada, vacía si no se informa.
	Version string
	// RunLevel es el nivel de ejecución (0 apagado, 2 completo).
	RunLevel int
	// LoggedUsers son los usuarios con sesión iniciada.
	LoggedUsers []string
}

// NetworkMode describe el modo de la primera interfaz de red de una
// máquina virtual.
type NetworkMode string

// Modos de red que la consola distingue.
const (
	// NetworkBridged indica una interfaz en modo puente.
	NetworkBridged NetworkMode = "bridged"
	// NetworkNAT indica una interfaz con traducción de direcciones.
	NetworkNAT NetworkMode = "nat"
	// NetworkHostOnly indica una red privada entre el anfitrión y la máquina.
	NetworkHostOnly NetworkMode = "hostonly"
	// NetworkInternal indica una red interna entre máquinas virtuales.
	NetworkInternal NetworkMode = "internal"
	// NetworkNone indica que la interfaz no tiene red.
	NetworkNone NetworkMode = "none"
	// NetworkUnknown indica que el modo de red no se pudo determinar.
	NetworkUnknown NetworkMode = "unknown"
)

// Target identifica la máquina administrada. El identificador es opaco
// para el núcleo: cada adaptador sabe cómo interpretarlo. La dirección
// se descubre antes de usarla; si es la dirección cero, es desconocida.
type Target struct {
	// ID es el nombre o UUID con el que el hipervisor identifica la máquina.
	ID string
	// Address es la dirección descubierta; su valor cero indica que aún se
	// desconoce.
	Address netip.Addr
}

// VM describe una máquina virtual registrada en el hipervisor, tal como
// la enumera el inventario. El nombre es único y sirve para vincularla.
type VM struct {
	// Name es el nombre con el que el hipervisor registra la máquina.
	Name string
	// UUID es el identificador único de la máquina.
	UUID string
	// State es el estado de encendido observado.
	State MachineState
}
