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

// Result contiene el resultado de la ejecución de una orden.
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
	StateRunning    MachineState = "running"
	StatePoweredOff MachineState = "poweroff"
	StatePaused     MachineState = "paused"
	StateSaved      MachineState = "saved"
	StateUnknown    MachineState = "unknown"
)

// Target identifica la máquina administrada. El identificador es opaco
// para el núcleo: cada adaptador sabe cómo interpretarlo. La dirección
// se descubre antes de usarla; si es la dirección cero, es desconocida.
type Target struct {
	ID      string
	Address netip.Addr
}
