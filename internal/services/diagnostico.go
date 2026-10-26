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

// Diagnostics reúne el estado de una máquina para explicar por qué no se
// puede registrar.
type Diagnostics struct {
	// VM es la máquina vinculada.
	VM string
	// State es el estado de encendido.
	State domain.MachineState
	// Network es el modo de la primera interfaz.
	Network domain.NetworkMode
	// IP es la dirección descubierta.
	IP netip.Addr
	// IsRegistered indica si ya está registrada.
	IsRegistered bool
	// GuestAdditionsVersion es la versión que informa la VM.
	GuestAdditionsVersion string
	// GuestAdditionsRunLevel es el nivel de Guest Additions.
	GuestAdditionsRunLevel int
	// UserSugerido es el primer usuario con sesión iniciada.
	UserSugerido string
	// SSHOpen indica si el puerto 22 responde.
	SSHOpen bool
	// Problemas describe lo que impide el registro.
	Problemas []string
	// Sugerencias indica qué hacer para resolverlo.
	Sugerencias []string
}

// DiagnosticsConfig reúne las dependencias del diagnóstico.
type DiagnosticsConfig struct {
	// Machine consulta el estado, la red, la dirección y Guest Additions.
	Machine ports.Machine
	// Registration informa si la máquina ya está registrada en la consola.
	Registration Registration
	// CurrentTarget devuelve la máquina vinculada.
	CurrentTarget func() domain.Target
	// VersionSystem devuelve la versión de VirtualBox del anfitrión.
	VersionSystem func(ctx context.Context) (string, error)
	// TestSSH comprueba si el puerto 22 responde; si es nil se usa la
	// prueba TCP por omisión.
	TestSSH func(ctx context.Context, address netip.Addr) bool
}

// DiagnosticsService explica el estado de la máquina vinculada.
type DiagnosticsService struct {
	machine       ports.Machine
	registration  Registration
	target        func() domain.Target
	versionSystem func(ctx context.Context) (string, error)
	testSSH       func(ctx context.Context, address netip.Addr) bool
}

// NewDiagnostics construye el servicio.
func NewDiagnostics(config *DiagnosticsConfig) *DiagnosticsService {
	test := config.TestSSH
	if test == nil {
		test = testPortSSH
	}

	return &DiagnosticsService{
		machine:       config.Machine,
		registration:  config.Registration,
		target:        config.CurrentTarget,
		versionSystem: config.VersionSystem,
		testSSH:       test,
	}
}

// Get devuelve el diagnóstico de la máquina vinculada.
func (d *DiagnosticsService) Get(ctx context.Context) (Diagnostics, error) {
	target := d.target()
	if target.ID == "" {
		return Diagnostics{}, ErrNoMachine
	}

	report := Diagnostics{VM: target.ID}

	if d.registration != nil {
		report.IsRegistered = d.registration.IsRegistered(target.ID)
	}

	state, err := d.machine.State(ctx, target)
	if err != nil {
		return Diagnostics{}, fmt.Errorf(msgStateMachine, err)
	}

	report.State = state

	if network, err := d.machine.Network(ctx, target); err == nil {
		report.Network = network
	}

	if guest, err := d.machine.GuestAdditions(ctx, target); err == nil {
		report.GuestAdditionsVersion = guest.Version
		report.GuestAdditionsRunLevel = guest.RunLevel

		if len(guest.LoggedUsers) > 0 {
			report.UserSugerido = guest.LoggedUsers[0]
		}
	}

	if state == domain.StateRunning {
		if address, err := d.machine.Address(ctx, target); err == nil {
			report.IP = address
			report.SSHOpen = d.testSSH(ctx, address)
		}
	}

	d.evaluate(ctx, &report)

	return report, nil
}

// evaluate llena los problemas y las sugerencias del informe.
func (d *DiagnosticsService) evaluate(ctx context.Context, report *Diagnostics) {
	d.evaluateState(report)
	d.evaluateNetwork(report)
	d.evaluateSSH(report)
	d.evaluateGuest(ctx, report)

	if report.UserSugerido != "" {
		report.Sugerencias = append(report.Sugerencias,
			"Prueba el registro con el usuario "+report.UserSugerido+".")
	}

	if report.IsRegistered {
		report.Problemas = append(report.Problemas, "La máquina ya está registrada.")
		report.Sugerencias = append(report.Sugerencias, "No necesitas registrarla de nuevo.")
	}
}

// evaluateState avisa si la máquina está apagada.
func (d *DiagnosticsService) evaluateState(report *Diagnostics) {
	if report.State == domain.StateRunning {
		return
	}

	report.Problemas = append(report.Problemas, "La máquina está apagada.")
	report.Sugerencias = append(report.Sugerencias, "Enciéndela con el botón de encendido del selector.")
}

// evaluateNetwork avisa si el adaptador no es alcanzable.
func (d *DiagnosticsService) evaluateNetwork(report *Diagnostics) {
	switch report.Network {
	case domain.NetworkNAT, domain.NetworkInternal, domain.NetworkNone:
		report.Problemas = append(report.Problemas,
			"La interfaz está en modo "+string(report.Network)+"; el anfitrión no alcanza esa red.")
		report.Sugerencias = append(report.Sugerencias,
			"Cambia el adaptador a modo puente u host-only en VirtualBox.")
	case domain.NetworkBridged, domain.NetworkHostOnly, domain.NetworkUnknown:
	}
}

// evaluateSSH avisa si el puerto 22 no responde.
func (d *DiagnosticsService) evaluateSSH(report *Diagnostics) {
	if report.State != domain.StateRunning || !report.IP.IsValid() || report.SSHOpen {
		return
	}

	report.Problemas = append(report.Problemas,
		"SSH no responde en "+report.IP.String()+":22.")
	report.Sugerencias = append(report.Sugerencias,
		"Instala openssh-server dentro de la VM o actualiza Guest Additions.")
}

// evaluateGuest avisa de problemas de Guest Additions y de versión.
func (d *DiagnosticsService) evaluateGuest(ctx context.Context, report *Diagnostics) {
	if report.State != domain.StateRunning {
		return
	}

	if report.GuestAdditionsVersion == "" {
		report.Problemas = append(report.Problemas, "Guest Additions no informa su versión.")
		report.Sugerencias = append(report.Sugerencias, "Instala o actualiza Guest Additions en la VM.")

		return
	}

	if report.GuestAdditionsRunLevel < 2 {
		report.Problemas = append(report.Problemas,
			"Guest Additions no está listo (nivel "+strconv.Itoa(report.GuestAdditionsRunLevel)+").")
		report.Sugerencias = append(report.Sugerencias, "Reinicia la VM para completar Guest Additions.")
	}

	if d.versionSystem == nil {
		return
	}

	version, err := d.versionSystem(ctx)
	if err != nil || sameVersionGreater(version, report.GuestAdditionsVersion) {
		return
	}

	report.Problemas = append(report.Problemas,
		"Guest Additions "+report.GuestAdditionsVersion+" no coincide con VirtualBox "+version+".")
	report.Sugerencias = append(report.Sugerencias,
		"Actualiza Guest Additions a la versión del anfitrión.")
}

// sameVersionGreater compara el número mayor de dos versiones.
func sameVersionGreater(host, invitada string) bool {
	greater := func(version string) string {
		parts := strings.SplitN(strings.TrimSpace(version), ".", 2)
		return parts[0]
	}

	return greater(host) == greater(invitada)
}

// openConnection abre la conexión TCP de la prueba; es una variable para
// poder simular el resultado en las pruebas.
var openConnection = func(ctx context.Context, address string) (net.Conn, error) {
	dialer := net.Dialer{Timeout: 2 * time.Second}

	return dialer.DialContext(ctx, "tcp", address)
}

// testPortSSH comprueba si el puerto 22 acepta conexiones.
func testPortSSH(ctx context.Context, address netip.Addr) bool {
	connection, err := openConnection(ctx, net.JoinHostPort(address.String(), "22"))
	if err != nil {
		return false
	}

	if err := connection.Close(); err != nil {
		slog.Warn("cerrar la prueba SSH", logErrorKey, err)
	}

	return true
}

// ErrDiagnosticsUnavailable envuelve un fallo del diagnóstico.
var ErrDiagnosticsUnavailable = errors.New("no se pudo diagnosticar la máquina")
