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

// ErrMaquinaEncendida indica que se pidió encender una máquina que ya
// está en ejecución.
var ErrMaquinaEncendida = errors.New("la máquina virtual ya está encendida")

// Mensajes de error reutilizados por las operaciones del inventario.
const (
	msgListarInventario = "listar las máquinas virtuales: %w"
	msgIdentificador    = "%w: %q"
)

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
	// Energia enciende las máquinas del hipervisor.
	Energia ports.Power
	// Maquina comprueba si una máquina en ejecución publica Guest
	// Additions y, por tanto, puede administrarse.
	Maquina ports.Machine
}

// Vinculacion busca máquinas en el inventario y cambia la que administra
// la consola.
type Vinculacion struct {
	inventario ports.Inventory
	seleccion  Selector
	energia    ports.Power
	maquina    ports.Machine
}

// NuevoVinculacion construye el servicio con las dependencias dadas.
func NuevoVinculacion(config *ConfigVinculacion) *Vinculacion {
	return &Vinculacion{
		inventario: config.Inventario,
		seleccion:  config.Seleccion,
		energia:    config.Energia,
		maquina:    config.Maquina,
	}
}

// Listar devuelve las máquinas del inventario marcando la vinculada. Si
// filtro no está vacío, se compara, sin distinguir mayúsculas, contra el
// nombre y el UUID.
func (v *Vinculacion) Listar(ctx context.Context, filtro string) ([]MaquinaVirtual, error) {
	maquinas, err := v.inventario.List(ctx)
	if err != nil {
		return nil, fmt.Errorf(msgListarInventario, err)
	}

	objetivo := v.seleccion.Objetivo()
	texto := strings.ToLower(strings.TrimSpace(filtro))
	visibles := make([]MaquinaVirtual, 0, len(maquinas))

	for _, maquina := range maquinas {
		if texto != "" && !coincide(maquina, texto) {
			continue
		}

		if !v.administrable(ctx, maquina) {
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

// administrable informa si la máquina puede administrarse: las apagadas
// se listan para poder encenderlas y las encendidas deben publicar Guest
// Additions, único canal para instalar SSH.
func (v *Vinculacion) administrable(ctx context.Context, maquina domain.VM) bool {
	if v.maquina == nil {
		return true
	}

	objetivo := domain.Target{ID: maquina.Name}

	modo, err := v.maquina.Network(ctx, objetivo)
	if err != nil {
		return false
	}

	switch modo {
	case domain.NetworkNAT, domain.NetworkInternal, domain.NetworkNone:
		return false
	case domain.NetworkBridged, domain.NetworkHostOnly:
	default:
		return false
	}

	if maquina.State != domain.StateRunning {
		return true
	}

	_, err = v.maquina.Address(ctx, objetivo)

	return err == nil
}

// Vincular establece la máquina que administra la consola. Acepta el
// nombre o el UUID del inventario y guarda siempre el nombre, que es el
// identificador estable que usan los adaptadores. Vincular una máquina
// apagada es válido: solo impide abrir sesión con ella.
func (v *Vinculacion) Vincular(ctx context.Context, identificador string) (MaquinaVirtual, error) {
	maquinas, err := v.inventario.List(ctx)
	if err != nil {
		return MaquinaVirtual{}, fmt.Errorf(msgListarInventario, err)
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

	return MaquinaVirtual{}, fmt.Errorf(msgIdentificador, ErrMaquinaNoEncontrada, buscado)
}

// Encender arranca una máquina del inventario. Rechaza con
// ErrMaquinaEncendida la que ya está en ejecución, para no informar un
// éxito que no ocurrió, y devuelve la máquina ya en estado «running».
func (v *Vinculacion) Encender(ctx context.Context, identificador string) (MaquinaVirtual, error) {
	maquina, err := v.buscar(ctx, identificador)
	if err != nil {
		return MaquinaVirtual{}, err
	}

	if maquina.State == domain.StateRunning {
		return MaquinaVirtual{}, fmt.Errorf(msgIdentificador, ErrMaquinaEncendida, maquina.Name)
	}

	if err := v.energia.Start(ctx, domain.Target{ID: maquina.Name}); err != nil {
		return MaquinaVirtual{}, fmt.Errorf("encender la máquina %q: %w", maquina.Name, err)
	}

	return MaquinaVirtual{
		Nombre: maquina.Name,
		UUID:   maquina.UUID,
		Estado: domain.StateRunning,
		Activa: v.seleccion.Objetivo().ID == maquina.Name || v.seleccion.Objetivo().ID == maquina.UUID,
	}, nil
}

// buscar devuelve la máquina del inventario que corresponde al
// identificador, por nombre o por UUID.
func (v *Vinculacion) buscar(ctx context.Context, identificador string) (domain.VM, error) {
	maquinas, err := v.inventario.List(ctx)
	if err != nil {
		return domain.VM{}, fmt.Errorf(msgListarInventario, err)
	}

	buscado := strings.TrimSpace(identificador)

	for _, maquina := range maquinas {
		if strings.EqualFold(maquina.Name, buscado) || strings.EqualFold(maquina.UUID, buscado) {
			return maquina, nil
		}
	}

	return domain.VM{}, fmt.Errorf(msgIdentificador, ErrMaquinaNoEncontrada, buscado)
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
