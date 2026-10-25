// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// ErrMaquinaNoEncontrada indica que el identificador recibido no
// corresponde a ninguna máquina del inventario del hipervisor.
var ErrMaquinaNoEncontrada = errors.New("la máquina virtual no está en el inventario")

// MaquinaVirtual es una máquina del inventario tal como la publica la
// consola: el nombre, el UUID, su estado y si es la vinculada.
type MaquinaVirtual struct {
	// Nombre es el nombre con el que el hipervisor registra la máquina.
	Nombre string
	// UUID es el identificador único de la máquina.
	UUID string
	// Estado es el estado de encendido observado.
	Estado domain.MachineState
	// Activa indica que la consola está vinculada con esta máquina.
	Activa bool
}

// Selector es la máquina con la que trabaja la consola. El descubridor
// de sesión lo implementa: vincular es cambiar su objetivo.
type Selector interface {
	// Objetivo devuelve la máquina vinculada.
	Objetivo() domain.Target
	// Establecer vincula la máquina indicada por su nombre.
	Establecer(nombre string)
}

// ConfigVinculacion reúne las dependencias de la vinculación.
type ConfigVinculacion struct {
	// Inventario enumera las máquinas del hipervisor.
	Inventario ports.Inventory
	// Seleccion es la máquina con la que trabaja la consola.
	Seleccion Selector
}

// Vinculacion busca máquinas en el inventario y cambia la que administra
// la consola.
type Vinculacion struct {
	inventario ports.Inventory
	seleccion  Selector
}

// NuevoVinculacion construye el servicio con las dependencias dadas.
func NuevoVinculacion(config *ConfigVinculacion) *Vinculacion {
	return &Vinculacion{inventario: config.Inventario, seleccion: config.Seleccion}
}

// Listar devuelve las máquinas del inventario marcando la vinculada. Si
// filtro no está vacío, se compara, sin distinguir mayúsculas, contra el
// nombre y el UUID.
func (v *Vinculacion) Listar(ctx context.Context, filtro string) ([]MaquinaVirtual, error) {
	maquinas, err := v.inventario.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar las máquinas virtuales: %w", err)
	}

	objetivo := v.seleccion.Objetivo()
	texto := strings.ToLower(strings.TrimSpace(filtro))
	visibles := make([]MaquinaVirtual, 0, len(maquinas))

	for _, maquina := range maquinas {
		if texto != "" && !coincide(maquina, texto) {
			continue
		}

		visibles = append(visibles, MaquinaVirtual{
			Nombre: maquina.Name,
			UUID:   maquina.UUID,
			Estado: maquina.State,
			Activa: maquina.Name == objetivo.ID || maquina.UUID == objetivo.ID,
		})
	}

	return visibles, nil
}

// Vincular establece la máquina que administra la consola. Acepta el
// nombre o el UUID del inventario y guarda siempre el nombre, que es el
// identificador estable que usan los adaptadores. Vincular una máquina
// apagada es válido: solo impide abrir sesión con ella.
func (v *Vinculacion) Vincular(ctx context.Context, identificador string) (MaquinaVirtual, error) {
	maquinas, err := v.inventario.List(ctx)
	if err != nil {
		return MaquinaVirtual{}, fmt.Errorf("listar las máquinas virtuales: %w", err)
	}

	buscado := strings.TrimSpace(identificador)

	for _, maquina := range maquinas {
		if !strings.EqualFold(maquina.Name, buscado) && !strings.EqualFold(maquina.UUID, buscado) {
			continue
		}

		v.seleccion.Establecer(maquina.Name)

		return MaquinaVirtual{
			Nombre: maquina.Name,
			UUID:   maquina.UUID,
			Estado: maquina.State,
			Activa: true,
		}, nil
	}

	return MaquinaVirtual{}, fmt.Errorf("%w: %q", ErrMaquinaNoEncontrada, buscado)
}

// Activa devuelve el nombre de la máquina vinculada, o una cadena vacía
// si la consola todavía no está vinculada con ninguna.
func (v *Vinculacion) Activa() string {
	return v.seleccion.Objetivo().ID
}

// coincide informa si la máquina contiene el texto de búsqueda en su
// nombre o en su UUID; el texto debe venir en minúsculas.
func coincide(maquina domain.VM, texto string) bool {
	return strings.Contains(strings.ToLower(maquina.Name), texto) ||
		strings.Contains(strings.ToLower(maquina.UUID), texto)
}
