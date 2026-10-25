// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// ErrCredencialesInvalidas indica que Guest Additions rechazó el
// usuario o la contraseña.
var ErrCredencialesInvalidas = errors.New("las credenciales no permiten ejecutar órdenes")

// ConfigAcceso reúne las dependencias para validar y guardar las
// credenciales de una máquina.
type ConfigAcceso struct {
	// Almacen guarda las credenciales por máquina.
	Almacen *Almacen
	// Maquina consulta el estado y la dirección de la máquina activa.
	Maquina ports.Machine
	// Guest abre una sesión por Guest Additions con las credenciales
	// guardadas.
	Guest ports.Connector
	// Objetivo devuelve la máquina vinculada.
	Objetivo func() domain.Target
}

// Acceso valida las credenciales de Guest Additions y las persiste para
// la máquina activa.
type Acceso struct {
	almacen  *Almacen
	maquina  ports.Machine
	guest    ports.Connector
	objetivo func() domain.Target
}

// NuevoAcceso construye el servicio de acceso.
func NuevoAcceso(config *ConfigAcceso) *Acceso {
	return &Acceso{
		almacen:  config.Almacen,
		maquina:  config.Maquina,
		guest:    config.Guest,
		objetivo: config.Objetivo,
	}
}

// Iniciar guarda las credenciales y comprueba con Guest Additions que
// permitan ejecutar una orden. Si la comprobación falla, restaura las
// credenciales anteriores.
func (a *Acceso) Iniciar(ctx context.Context, usuario, password string, puerto uint16) (Resultado, error) {
	objetivo, direccion, err := a.resolver(ctx)
	if err != nil {
		return Resultado{}, err
	}

	if puerto == 0 {
		puerto = 22
	}

	anterior, existia := a.almacen.Obtener(objetivo.ID)

	if err := a.almacen.Guardar(objetivo.ID, Credenciales{
		Usuario:  usuario,
		Password: password,
		Puerto:   puerto,
	}); err != nil {
		return Resultado{}, err
	}

	exito := false

	defer func() {
		if exito {
			return
		}

		if existia {
			if err := a.almacen.Guardar(objetivo.ID, anterior); err != nil {
				slog.Warn("restaurar las credenciales", claveErrorLog, err)
			}

			return
		}

		if err := a.almacen.Borrar(objetivo.ID); err != nil {
			slog.Warn("borrar las credenciales", claveErrorLog, err)
		}
	}()

	sesion, err := a.guest.Connect(ctx, domain.Target{ID: objetivo.ID, Address: direccion})
	if err != nil {
		return Resultado{}, fmt.Errorf("abrir la sesión de Guest Additions: %w", err)
	}

	defer cerrarSesion(sesion)

	if _, err := sesion.Run(ctx, &domain.Command{Path: "/usr/bin/id", Args: []string{"-u"}}); err != nil {
		return Resultado{}, fmt.Errorf("%w: %w", ErrCredencialesInvalidas, err)
	}

	exito = true

	return Resultado{
		Mensaje: "Credenciales verificadas y guardadas.",
		Codigo:  0,
	}, nil
}

// resolver comprueba la máquina activa y devuelve su dirección.
func (a *Acceso) resolver(ctx context.Context) (domain.Target, netip.Addr, error) {
	objetivo := a.objetivo()
	if objetivo.ID == "" {
		return domain.Target{}, netip.Addr{}, fmt.Errorf("%w: elige una máquina del inventario", ErrSinMaquina)
	}

	estado, err := a.maquina.State(ctx, objetivo)
	if err != nil {
		return domain.Target{}, netip.Addr{}, fmt.Errorf("consultar el estado de la máquina: %w", err)
	}

	if estado != domain.StateRunning {
		return domain.Target{}, netip.Addr{}, fmt.Errorf(
			"%w: la máquina %q está en estado %q", ErrSesionNoDisponible, objetivo.ID, estado,
		)
	}

	direccion, err := a.maquina.Address(ctx, objetivo)
	if err != nil {
		return domain.Target{}, netip.Addr{}, fmt.Errorf("descubrir la dirección de la máquina: %w", err)
	}

	return objetivo, direccion, nil
}

// Olvidar elimina las credenciales de la máquina activa.
func (a *Acceso) Olvidar() error {
	objetivo := a.objetivo()
	if objetivo.ID == "" {
		return ErrSinMaquina
	}

	return a.almacen.Borrar(objetivo.ID)
}
