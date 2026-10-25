// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// Nombres de los conectores y mensajes repetidos en las pruebas.
const (
	conectorSSH    = "ssh"
	conectorGuest  = "guest-additions"
	msgDescubrirOK = "Descubrir() error = %v, want nil"
	msgIP          = "IP = %s, want %s"
)

// stubMachine es una implementación falsa de ports.Machine que devuelve
// la dirección y el estado preprogramados, sin hablar con VirtualBox.
type stubMachine struct {
	addr     netip.Addr
	state    domain.MachineState
	stateErr error
	addrErr  error
	calls    int
}

// Address devuelve la dirección preprogramada y cuenta la llamada.
func (s *stubMachine) Address(context.Context, domain.Target) (netip.Addr, error) {
	s.calls++

	return s.addr, s.addrErr
}

// State devuelve el estado de encendido preprogramado.
func (s *stubMachine) State(context.Context, domain.Target) (domain.MachineState, error) {
	return s.state, s.stateErr
}

// stubSession es una sesión falsa que registra la orden ejecutada y si
// se cerró.
type stubSession struct {
	closed   bool
	closeErr error
	result   domain.Result
	err      error
	commands []*domain.Command
}

// Run registra la orden y devuelve el resultado preprogramado.
func (s *stubSession) Run(_ context.Context, cmd *domain.Command) (domain.Result, error) {
	s.commands = append(s.commands, cmd)

	return s.result, s.err
}

// Put no transfiere nada y siempre termina bien.
func (s *stubSession) Put(context.Context, string, string) error {
	return nil
}

// Close marca la sesión como cerrada y devuelve el error configurado.
func (s *stubSession) Close() error {
	s.closed = true

	return s.closeErr
}

// stubConnector es un conector falso con nombre, sondeo y conexión
// preprogramados.
type stubConnector struct {
	name    string
	probe   error
	connect error
	session ports.Session
	probed  int
}

// Name devuelve el nombre preprogramado del conector.
func (c *stubConnector) Name() string { return c.name }

// Probe devuelve el error preprogramado y cuenta el sondeo.
func (c *stubConnector) Probe(context.Context, domain.Target) error {
	c.probed++

	return c.probe
}

// Connect devuelve la sesión preprogramada o el error configurado.
func (c *stubConnector) Connect(context.Context, domain.Target) (ports.Session, error) {
	if c.connect != nil {
		return nil, c.connect
	}

	if c.session != nil {
		return c.session, nil
	}

	return &stubSession{}, nil
}

// stubAbridor es un abridor de sesiones falso con sesión o error
// preprogramados.
type stubAbridor struct {
	sesion   ports.Session
	err      error
	abiertas int
}

// Abrir devuelve la sesión preprogramada o el error configurado.
func (a *stubAbridor) Abrir(context.Context) (ports.Session, error) {
	a.abiertas++

	if a.err != nil {
		return nil, a.err
	}

	return a.sesion, nil
}

// nuevoDescubridor arma un Descubridor de prueba con una máquina y dos
// conectores falsos.
func nuevoDescubridor(m *stubMachine, conectores ...ports.Connector) *Descubridor {
	return NuevoDescubridor(&ConfigDescubridor{
		Maquina:    m,
		Objetivo:   domain.Target{ID: "vm1"},
		Conectores: conectores,
	})
}

// TestDescubridorSeleccionaSSH comprueba que se prefiere el conector SSH
// cuando su sondeo tiene éxito.
func TestDescubridorSeleccionaSSH(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("192.168.1.50")
	m := &stubMachine{addr: ip, state: domain.StateRunning}
	sshConn := &stubConnector{name: conectorSSH}
	gaConn := &stubConnector{name: conectorGuest}

	info, err := nuevoDescubridor(m, sshConn, gaConn).Descubrir(context.Background())
	if err != nil {
		t.Fatalf(msgDescubrirOK, err)
	}

	if info.Conector != conectorSSH {
		t.Errorf("Conector = %q, want %q", info.Conector, conectorSSH)
	}

	if info.IP.String() != ip.String() {
		t.Errorf(msgIP, info.IP.String(), ip.String())
	}

	if info.Estado != domain.StateRunning {
		t.Errorf("Estado = %q, want %q", info.Estado, domain.StateRunning)
	}

	if gaConn.probed != 0 {
		t.Errorf("guest-additions se sondeó %d veces, want 0", gaConn.probed)
	}
}

// TestDescubridorUsaGuestAdditionsSinSSH comprueba que se usa Guest
// Additions cuando el conector SSH no puede atender la máquina.
func TestDescubridorUsaGuestAdditionsSinSSH(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("192.168.1.51")
	m := &stubMachine{addr: ip, state: domain.StateRunning}
	sshConn := &stubConnector{name: conectorSSH, probe: errors.New("sin llave")}
	gaConn := &stubConnector{name: conectorGuest}

	info, err := nuevoDescubridor(m, sshConn, gaConn).Descubrir(context.Background())
	if err != nil {
		t.Fatalf(msgDescubrirOK, err)
	}

	if info.Conector != conectorGuest {
		t.Errorf("Conector = %q, want %q", info.Conector, conectorGuest)
	}
}

// TestDescubridorFallaSiMaquinaApagada comprueba que no se elige ningún
// conector cuando la máquina no está en ejecución.
func TestDescubridorFallaSiMaquinaApagada(t *testing.T) {
	t.Parallel()

	m := &stubMachine{state: domain.StatePoweredOff}

	_, err := nuevoDescubridor(m, &stubConnector{name: conectorSSH}).Descubrir(context.Background())
	if !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf("Descubrir() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestDescubridorFallaSinConectores comprueba que se devuelve
// ErrSesionNoDisponible cuando ningún conector supera el sondeo.
func TestDescubridorFallaSinConectores(t *testing.T) {
	t.Parallel()

	m := &stubMachine{addr: netip.MustParseAddr("192.168.1.52"), state: domain.StateRunning}
	sshConn := &stubConnector{name: conectorSSH, probe: errors.New("sin llave")}
	gaConn := &stubConnector{name: conectorGuest, probe: errors.New("sin guest additions")}

	_, err := nuevoDescubridor(m, sshConn, gaConn).Descubrir(context.Background())
	if !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf("Descubrir() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestDescubridorDescubreDireccion comprueba que se consulta la máquina
// cuando el objetivo no trae dirección.
func TestDescubridorDescubreDireccion(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("192.168.1.53")
	m := &stubMachine{addr: ip, state: domain.StateRunning}

	info, err := nuevoDescubridor(m, &stubConnector{name: conectorSSH}).Descubrir(context.Background())
	if err != nil {
		t.Fatalf(msgDescubrirOK, err)
	}

	if m.calls != 1 {
		t.Errorf("Address() se llamó %d veces, want 1", m.calls)
	}

	if info.IP.String() != ip.String() {
		t.Errorf(msgIP, info.IP.String(), ip.String())
	}
}

// TestDescubridorErrorAlConsultarEstado comprueba que un fallo del
// puerto Machine al leer el estado se propaga envuelto.
func TestDescubridorErrorAlConsultarEstado(t *testing.T) {
	t.Parallel()

	m := &stubMachine{stateErr: errors.New("VBoxManage no responde")}

	_, err := nuevoDescubridor(m, &stubConnector{name: conectorSSH}).Descubrir(context.Background())
	if err == nil {
		t.Fatal("Descubrir() error = nil, want el error del puerto Machine")
	}

	if errors.Is(err, ErrSesionNoDisponible) {
		t.Errorf("Descubrir() error = %v, no debe ser ErrSesionNoDisponible", err)
	}
}

// TestDescubridorErrorAlDescubrirDireccion comprueba que un fallo al
// descubrir la dirección se propaga envuelto.
func TestDescubridorErrorAlDescubrirDireccion(t *testing.T) {
	t.Parallel()

	m := &stubMachine{state: domain.StateRunning, addrErr: errors.New("sin Guest Additions")}

	_, err := nuevoDescubridor(m, &stubConnector{name: conectorSSH}).Descubrir(context.Background())
	if err == nil {
		t.Fatal("Descubrir() error = nil, want el error al descubrir la dirección")
	}
}

// TestDescubridorUsaDireccionDelObjetivo comprueba que no se consulta la
// máquina si el objetivo ya trae una dirección válida.
func TestDescubridorUsaDireccionDelObjetivo(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("192.168.1.54")
	m := &stubMachine{state: domain.StateRunning}

	desc := NuevoDescubridor(&ConfigDescubridor{
		Maquina:    m,
		Objetivo:   domain.Target{ID: "vm1", Address: ip},
		Conectores: []ports.Connector{&stubConnector{name: conectorSSH}},
	})

	info, err := desc.Descubrir(context.Background())
	if err != nil {
		t.Fatalf(msgDescubrirOK, err)
	}

	if m.calls != 0 {
		t.Errorf("Address() se llamó %d veces, want 0", m.calls)
	}

	if info.IP.String() != ip.String() {
		t.Errorf(msgIP, info.IP.String(), ip.String())
	}
}

// TestAbrirDevuelveSesionDelConectorElegido comprueba que Abrir conecta
// con el primer conector disponible.
func TestAbrirDevuelveSesionDelConectorElegido(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("192.168.1.55")
	m := &stubMachine{addr: ip, state: domain.StateRunning}
	sesion := &stubSession{}
	sshConn := &stubConnector{name: conectorSSH, session: sesion}
	gaConn := &stubConnector{name: conectorGuest}

	desc := nuevoDescubridor(m, sshConn, gaConn)

	obtenida, err := desc.Abrir(context.Background())
	if err != nil {
		t.Fatalf("Abrir() error = %v, want nil", err)
	}

	if obtenida != sesion {
		t.Error("Abrir() no devolvió la sesión del conector SSH")
	}

	if gaConn.probed != 0 {
		t.Errorf("guest-additions se sondeó %d veces, want 0", gaConn.probed)
	}
}

// TestAbrirFallaSinSesion comprueba que Abrir propaga
// ErrSesionNoDisponible cuando la máquina no puede atenderse.
func TestAbrirFallaSinSesion(t *testing.T) {
	t.Parallel()

	m := &stubMachine{state: domain.StatePoweredOff}

	_, err := nuevoDescubridor(m, &stubConnector{name: conectorSSH}).Abrir(context.Background())
	if !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf("Abrir() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestAbrirPropagaErrorDeConexion comprueba que el fallo del conector al
// conectar se propaga envuelto.
func TestAbrirPropagaErrorDeConexion(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("192.168.1.56")
	m := &stubMachine{addr: ip, state: domain.StateRunning}
	sshConn := &stubConnector{name: conectorSSH, connect: errors.New("ssh rechazó la llave")}

	_, err := nuevoDescubridor(m, sshConn).Abrir(context.Background())
	if err == nil {
		t.Fatal("Abrir() error = nil, want el error de conexión")
	}
}
