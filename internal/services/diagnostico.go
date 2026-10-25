// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// Diagnostico reúne el estado de una máquina para explicar por qué no se
// puede registrar.
type Diagnostico struct {
	// VM es la máquina vinculada.
	VM string
	// Estado es el estado de encendido.
	Estado domain.MachineState
	// Red es el modo de la primera interfaz.
	Red domain.NetworkMode
	// IP es la dirección descubierta.
	IP netip.Addr
	// Registrada indica si ya está registrada.
	Registrada bool
	// GuestAdditionsVersion es la versión que informa la VM.
	GuestAdditionsVersion string
	// GuestAdditionsRunLevel es el nivel de Guest Additions.
	GuestAdditionsRunLevel int
	// UsuarioSugerido es el primer usuario con sesión iniciada.
	UsuarioSugerido string
	// SSHAbierto indica si el puerto 22 responde.
	SSHAbierto bool
	// Problemas describe lo que impide el registro.
	Problemas []string
	// Sugerencias indica qué hacer para resolverlo.
	Sugerencias []string
}

// ConfigDiagnostico reúne las dependencias del diagnóstico.
type ConfigDiagnostico struct {
	Maquina        ports.Machine
	Registro       Registro
	Objetivo       func() domain.Target
	VersionSistema func(ctx context.Context) (string, error)
	ProbarSSH      func(ctx context.Context, direccion netip.Addr) bool
}

// DiagnosticoServicio explica el estado de la máquina vinculada.
type DiagnosticoServicio struct {
	maquina        ports.Machine
	registro       Registro
	objetivo       func() domain.Target
	versionSistema func(ctx context.Context) (string, error)
	probarSSH      func(ctx context.Context, direccion netip.Addr) bool
}

// NuevoDiagnostico construye el servicio.
func NuevoDiagnostico(config *ConfigDiagnostico) *DiagnosticoServicio {
	probar := config.ProbarSSH
	if probar == nil {
		probar = probarPuertoSSH
	}

	return &DiagnosticoServicio{
		maquina:        config.Maquina,
		registro:       config.Registro,
		objetivo:       config.Objetivo,
		versionSistema: config.VersionSistema,
		probarSSH:      probar,
	}
}

// Obtener devuelve el diagnóstico de la máquina vinculada.
func (d *DiagnosticoServicio) Obtener(ctx context.Context) (Diagnostico, error) {
	objetivo := d.objetivo()
	if objetivo.ID == "" {
		return Diagnostico{}, ErrSinMaquina
	}

	informe := Diagnostico{VM: objetivo.ID}

	if d.registro != nil {
		informe.Registrada = d.registro.Registrada(objetivo.ID)
	}

	estado, err := d.maquina.State(ctx, objetivo)
	if err != nil {
		return Diagnostico{}, fmt.Errorf(msgEstadoMaquina, err)
	}

	informe.Estado = estado

	if red, err := d.maquina.Network(ctx, objetivo); err == nil {
		informe.Red = red
	}

	if guest, err := d.maquina.GuestAdditions(ctx, objetivo); err == nil {
		informe.GuestAdditionsVersion = guest.Version
		informe.GuestAdditionsRunLevel = guest.RunLevel

		if len(guest.LoggedUsers) > 0 {
			informe.UsuarioSugerido = guest.LoggedUsers[0]
		}
	}

	if estado == domain.StateRunning {
		if direccion, err := d.maquina.Address(ctx, objetivo); err == nil {
			informe.IP = direccion
			informe.SSHAbierto = d.probarSSH(ctx, direccion)
		}
	}

	d.evaluar(ctx, &informe)

	return informe, nil
}

// evaluar llena los problemas y las sugerencias del informe.
func (d *DiagnosticoServicio) evaluar(ctx context.Context, informe *Diagnostico) {
	d.evaluarEstado(informe)
	d.evaluarRed(informe)
	d.evaluarSSH(informe)
	d.evaluarGuest(ctx, informe)

	if informe.UsuarioSugerido != "" {
		informe.Sugerencias = append(informe.Sugerencias,
			"Prueba el registro con el usuario "+informe.UsuarioSugerido+".")
	}

	if informe.Registrada {
		informe.Problemas = append(informe.Problemas, "La máquina ya está registrada.")
		informe.Sugerencias = append(informe.Sugerencias, "No necesitas registrarla de nuevo.")
	}
}

// evaluarEstado avisa si la máquina está apagada.
func (d *DiagnosticoServicio) evaluarEstado(informe *Diagnostico) {
	if informe.Estado == domain.StateRunning {
		return
	}

	informe.Problemas = append(informe.Problemas, "La máquina está apagada.")
	informe.Sugerencias = append(informe.Sugerencias, "Enciéndela con el botón de encendido del selector.")
}

// evaluarRed avisa si el adaptador no es alcanzable.
func (d *DiagnosticoServicio) evaluarRed(informe *Diagnostico) {
	switch informe.Red {
	case domain.NetworkNAT, domain.NetworkInternal, domain.NetworkNone:
		informe.Problemas = append(informe.Problemas,
			"La interfaz está en modo "+string(informe.Red)+"; el anfitrión no alcanza esa red.")
		informe.Sugerencias = append(informe.Sugerencias,
			"Cambia el adaptador a modo puente u host-only en VirtualBox.")
	case domain.NetworkBridged, domain.NetworkHostOnly, domain.NetworkUnknown:
	}
}

// evaluarSSH avisa si el puerto 22 no responde.
func (d *DiagnosticoServicio) evaluarSSH(informe *Diagnostico) {
	if informe.Estado != domain.StateRunning || !informe.IP.IsValid() || informe.SSHAbierto {
		return
	}

	informe.Problemas = append(informe.Problemas,
		"SSH no responde en "+informe.IP.String()+":22.")
	informe.Sugerencias = append(informe.Sugerencias,
		"Instala openssh-server dentro de la VM o actualiza Guest Additions.")
}

// evaluarGuest avisa de problemas de Guest Additions y de versión.
func (d *DiagnosticoServicio) evaluarGuest(ctx context.Context, informe *Diagnostico) {
	if informe.Estado != domain.StateRunning {
		return
	}

	if informe.GuestAdditionsVersion == "" {
		informe.Problemas = append(informe.Problemas, "Guest Additions no informa su versión.")
		informe.Sugerencias = append(informe.Sugerencias, "Instala o actualiza Guest Additions en la VM.")

		return
	}

	if informe.GuestAdditionsRunLevel < 2 {
		informe.Problemas = append(informe.Problemas,
			"Guest Additions no está listo (nivel "+strconv.Itoa(informe.GuestAdditionsRunLevel)+").")
		informe.Sugerencias = append(informe.Sugerencias, "Reinicia la VM para completar Guest Additions.")
	}

	if d.versionSistema == nil {
		return
	}

	version, err := d.versionSistema(ctx)
	if err != nil || mismaVersionMayor(version, informe.GuestAdditionsVersion) {
		return
	}

	informe.Problemas = append(informe.Problemas,
		"Guest Additions "+informe.GuestAdditionsVersion+" no coincide con VirtualBox "+version+".")
	informe.Sugerencias = append(informe.Sugerencias,
		"Actualiza Guest Additions a la versión del anfitrión.")
}

// mismaVersionMayor compara el número mayor de dos versiones.
func mismaVersionMayor(anfitrion, invitada string) bool {
	mayor := func(version string) string {
		partes := strings.SplitN(strings.TrimSpace(version), ".", 2)
		return partes[0]
	}

	return mayor(anfitrion) == mayor(invitada)
}

// probarPuertoSSH comprueba si el puerto 22 acepta conexiones.
func probarPuertoSSH(ctx context.Context, direccion netip.Addr) bool {
	dialer := net.Dialer{Timeout: 2 * time.Second}

	conexion, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(direccion.String(), "22"))
	if err != nil {
		return false
	}

	if err := conexion.Close(); err != nil {
		slog.Warn("cerrar la prueba SSH", claveErrorLog, err)
	}

	return true
}

// ErrDiagnosticoNoDisponible envuelve un fallo del diagnóstico.
var ErrDiagnosticoNoDisponible = errors.New("no se pudo diagnosticar la máquina")
