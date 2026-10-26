// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"sync"
	"testing"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// Nombres de los conectores y mensajes repetidos en las pruebas.
const (
	connectorSSH          = "ssh"
	connectorGuest        = "guest-additions"
	msgDiscoverOK         = "Descubrir() error = %v, want nil"
	msgSessionOpen        = "la sesión no se cerró"
	msgCodeCero           = "Codigo = %d, want 0"
	msgWriteOK            = "Escribir() error = %v, want nil"
	msgCommandUnica       = "la sesión ejecutó %d órdenes, want 1"
	msgStderrSession      = "Stderr = %q, want el stderr de la sesión"
	nameHostTests         = "nodo-01"
	ipHostTests           = "192.168.1.60"
	outputHostname        = "vm1\n"
	commandCat            = "cat"
	msgOpen               = "Abrir() se llamó %d veces, want 1"
	msgListOK             = "Listar() error = %v, want nil"
	msgWritePub           = "escribir la llave pública: %v"
	nameJorge             = "jorge"
	fileJorge             = "jorge_ed25519"
	msgCreateDir          = "crear el directorio: %v"
	lineJorge             = "jorge:x:1000:1000::/home/jorge:/bin/bash\n"
	msgListWithoutSession = "Listar() error = %v, want ErrSesionNoDisponible"
	msgGetWithoutSession  = "Obtener() error = %v, want ErrSesionNoDisponible"
	stateInactive         = "inactive"
	duenoTests            = "catalogo"
	ipTests               = "192.168.1.50"
	ipServerTests         = "192.168.1.61"
	ipServerNew           = "192.168.1.62"
	msgDeleteOK           = "Eliminar() error = %v, want nil"
	msgResultTechnical    = "resultado = %+v, want código y stderr"
	msgCommandDos         = "la sesión ejecutó %d órdenes, want 2"
	// #nosec G101 -- usuario de prueba, no una credencial real.
	userRoot                  = "root"
	passwordTests             = "secreto"
	vmTests                   = "vm1"
	fileKey                   = "credenciales.key"
	moduleSystemd             = "systemd"
	msgStoreOK                = "NuevoAlmacen() error = %v"
	msgDiscoverWithoutSession = "Descubrir() error = %v, want ErrSesionNoDisponible"
	msgSaveOK                 = "Guardar() error = %v, want nil"
	msgPrepareFile            = "preparar el archivo: %v"
	// #nosec G101 -- nombre de archivo de prueba, no una credencial.
	fileCredentials = "credenciales.json"
	msgIP           = "IP = %s, want %s"
)

// stubMachine es una implementación falsa de ports.Machine que devuelve
// la dirección y el estado preprogramados, sin hablar con VirtualBox.
type stubMachine struct {
	addr     netip.Addr
	state    domain.MachineState
	network  domain.NetworkMode
	guest    domain.GuestAdditions
	stateErr error
	addrErr  error
	netErr   error
	guestErr error
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

// GuestAdditions devuelve la información preprogramada.
func (s *stubMachine) GuestAdditions(context.Context, domain.Target) (domain.GuestAdditions, error) {
	return s.guest, s.guestErr
}

// Network devuelve el modo de red preprogramado («bridged» por omisión).
func (s *stubMachine) Network(context.Context, domain.Target) (domain.NetworkMode, error) {
	if s.network == "" {
		return domain.NetworkBridged, s.netErr
	}

	return s.network, s.netErr
}

// stubSession es una sesión falsa que registra la orden ejecutada y si
// se cerró.
type stubSession struct {
	closed   bool
	closeErr error
	result   domain.Result
	results  []domain.Result
	err      error
	errs     []error
	commands []*domain.Command
	puts     [][2]string
	putErr   error
}

// Run registra la orden y devuelve el siguiente resultado de la cola o
// el resultado único preprogramado.
func (s *stubSession) Run(_ context.Context, cmd *domain.Command) (domain.Result, error) {
	s.commands = append(s.commands, cmd)

	result := s.result

	if len(s.results) > 0 {
		result = s.results[0]
		s.results = s.results[1:]
	}

	if len(s.errs) > 0 {
		err := s.errs[0]
		s.errs = s.errs[1:]

		return result, err
	}

	return result, s.err
}

// Put registra la transferencia y siempre termina bien.
func (s *stubSession) Put(_ context.Context, source, destination string) error {
	s.puts = append(s.puts, [2]string{source, destination})

	return s.putErr
}

// Close marca la sesión como cerrada y devuelve el error configurado.
func (s *stubSession) Close() error {
	s.closed = true

	return s.closeErr
}

// stubConnector es un conector falso con nombre, sondeo y conexión
// preprogramados.
type stubConnector struct {
	name      string
	probe     error
	probeErrs []error
	connect   error
	session   ports.Session
	probed    int
}

// Name devuelve el nombre preprogramado del conector.
func (c *stubConnector) Name() string { return c.name }

// Probe devuelve el error preprogramado y cuenta el sondeo.
func (c *stubConnector) Probe(context.Context, domain.Target) error {
	c.probed++

	if len(c.probeErrs) > 0 {
		err := c.probeErrs[0]
		c.probeErrs = c.probeErrs[1:]

		return err
	}

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

// stubOpener es un abridor de sesiones falso con sesión o error
// preprogramados.
type stubOpener struct {
	session ports.Session
	err     error
	open    int
}

// Open devuelve la sesión preprogramada o el error configurado.
func (a *stubOpener) Open(context.Context) (ports.Session, error) {
	a.open++

	if a.err != nil {
		return nil, a.err
	}

	return a.session, nil
}

// newDiscoverer arma un Discoverer de prueba con una máquina y dos
// conectores falsos.
func newDiscoverer(m *stubMachine, connectors ...ports.Connector) *Discoverer {
	return NewDiscoverer(&DiscovererConfig{
		Machine:       m,
		CurrentTarget: domain.Target{ID: vmTests},
		Connectors:    connectors,
	})
}

// TestDiscovererSeleccionaSSH comprueba que se prefiere el conector SSH
// cuando su sondeo tiene éxito.
func TestDiscovererSeleccionaSSH(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr(ipTests)
	m := &stubMachine{addr: ip, state: domain.StateRunning}
	sshConn := &stubConnector{name: connectorSSH}
	gaConn := &stubConnector{name: connectorGuest}

	info, err := newDiscoverer(m, sshConn, gaConn).Discover(context.Background())
	if err != nil {
		t.Fatalf(msgDiscoverOK, err)
	}

	if info.Connector != connectorSSH {
		t.Errorf("Conector = %q, want %q", info.Connector, connectorSSH)
	}

	if info.IP.String() != ip.String() {
		t.Errorf(msgIP, info.IP.String(), ip.String())
	}

	if info.State != domain.StateRunning {
		t.Errorf("Estado = %q, want %q", info.State, domain.StateRunning)
	}

	if gaConn.probed != 0 {
		t.Errorf("guest-additions se sondeó %d veces, want 0", gaConn.probed)
	}
}

// TestDiscovererUsesGuestAdditionsWithoutSSH comprueba que se usa Guest
// Additions cuando el conector SSH no puede atender la máquina.
func TestDiscovererUsesGuestAdditionsWithoutSSH(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("192.168.1.51")
	m := &stubMachine{addr: ip, state: domain.StateRunning}
	sshConn := &stubConnector{name: connectorSSH, probe: errors.New("sin llave")}
	gaConn := &stubConnector{name: connectorGuest}

	info, err := newDiscoverer(m, sshConn, gaConn).Discover(context.Background())
	if err != nil {
		t.Fatalf(msgDiscoverOK, err)
	}

	if info.Connector != connectorGuest {
		t.Errorf("Conector = %q, want %q", info.Connector, connectorGuest)
	}
}

// TestDiscovererRegisteredNotUsesGuestAdditions comprueba que una
// máquina registrada no vuelve al canal de Guest Additions si SSH falla.
func TestDiscovererRegisteredNotUsesGuestAdditions(t *testing.T) {
	t.Parallel()

	m := &stubMachine{addr: netip.MustParseAddr("192.168.1.56"), state: domain.StateRunning}
	sshConn := &stubConnector{name: connectorSSH, probe: errors.New("ssh no responde")}
	gaConn := &stubConnector{name: connectorGuest}

	desc := NewDiscoverer(&DiscovererConfig{
		Machine:       m,
		CurrentTarget: domain.Target{ID: vmTests},
		Connectors:    []ports.Connector{sshConn, gaConn},
		Registration:  &stubRegistration{registradas: map[string]bool{vmTests: true}},
	})

	_, err := desc.Discover(context.Background())
	if !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf(msgDiscoverWithoutSession, err)
	}

	if gaConn.probed != 0 {
		t.Errorf("Guest Additions se sondeó %d veces, want 0", gaConn.probed)
	}
}

// TestDiscovererFailsIfMachineOff comprueba que no se elige ningún
// conector cuando la máquina no está en ejecución.
func TestDiscovererFailsIfMachineOff(t *testing.T) {
	t.Parallel()

	m := &stubMachine{state: domain.StatePoweredOff}

	_, err := newDiscoverer(m, &stubConnector{name: connectorSSH}).Discover(context.Background())
	if !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf(msgDiscoverWithoutSession, err)
	}
}

// TestDiscovererFailsWithoutConnectors comprueba que se devuelve
// ErrSessionUnavailable cuando ningún conector supera el sondeo.
func TestDiscovererFailsWithoutConnectors(t *testing.T) {
	t.Parallel()

	m := &stubMachine{addr: netip.MustParseAddr("192.168.1.52"), state: domain.StateRunning}
	sshConn := &stubConnector{name: connectorSSH, probe: errors.New("sin llave")}
	gaConn := &stubConnector{name: connectorGuest, probe: errors.New("sin guest additions")}

	_, err := newDiscoverer(m, sshConn, gaConn).Discover(context.Background())
	if !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf(msgDiscoverWithoutSession, err)
	}
}

// TestDiscovererDiscoversAddress comprueba que se consulta la máquina
// cuando el objetivo no trae dirección.
func TestDiscovererDiscoversAddress(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("192.168.1.53")
	m := &stubMachine{addr: ip, state: domain.StateRunning}

	info, err := newDiscoverer(m, &stubConnector{name: connectorSSH}).Discover(context.Background())
	if err != nil {
		t.Fatalf(msgDiscoverOK, err)
	}

	if m.calls != 1 {
		t.Errorf("Address() se llamó %d veces, want 1", m.calls)
	}

	if info.IP.String() != ip.String() {
		t.Errorf(msgIP, info.IP.String(), ip.String())
	}
}

// TestDiscovererErrorToGetState comprueba que un fallo del
// puerto Machine al leer el estado se propaga envuelto.
func TestDiscovererErrorToGetState(t *testing.T) {
	t.Parallel()

	m := &stubMachine{stateErr: errors.New("VBoxManage no responde")}

	_, err := newDiscoverer(m, &stubConnector{name: connectorSSH}).Discover(context.Background())
	if err == nil {
		t.Fatal("Descubrir() error = nil, want el error del puerto Machine")
	}

	if errors.Is(err, ErrSessionUnavailable) {
		t.Errorf("Descubrir() error = %v, no debe ser ErrSesionNoDisponible", err)
	}
}

// TestDiscovererErrorToDiscoverAddress comprueba que un fallo al
// descubrir la dirección se propaga envuelto.
func TestDiscovererErrorToDiscoverAddress(t *testing.T) {
	t.Parallel()

	m := &stubMachine{state: domain.StateRunning, addrErr: errors.New("sin Guest Additions")}

	_, err := newDiscoverer(m, &stubConnector{name: connectorSSH}).Discover(context.Background())
	if !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf(msgDiscoverWithoutSession, err)
	}
}

// TestDiscovererUsesAddressOfTarget comprueba que no se consulta la
// máquina si el objetivo ya trae una dirección válida.
func TestDiscovererUsesAddressOfTarget(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("192.168.1.54")
	m := &stubMachine{state: domain.StateRunning}

	desc := NewDiscoverer(&DiscovererConfig{
		Machine:       m,
		CurrentTarget: domain.Target{ID: vmTests, Address: ip},
		Connectors:    []ports.Connector{&stubConnector{name: connectorSSH}},
	})

	info, err := desc.Discover(context.Background())
	if err != nil {
		t.Fatalf(msgDiscoverOK, err)
	}

	if m.calls != 0 {
		t.Errorf("Address() se llamó %d veces, want 0", m.calls)
	}

	if info.IP.String() != ip.String() {
		t.Errorf(msgIP, info.IP.String(), ip.String())
	}
}

// TestOpenReturnsSessionOfConnectorChosen comprueba que Open conecta
// con el primer conector disponible.
func TestOpenReturnsSessionOfConnectorChosen(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("192.168.1.55")
	m := &stubMachine{addr: ip, state: domain.StateRunning}
	session := &stubSession{}
	sshConn := &stubConnector{name: connectorSSH, session: session}
	gaConn := &stubConnector{name: connectorGuest}

	desc := newDiscoverer(m, sshConn, gaConn)

	obtenida, err := desc.Open(context.Background())
	if err != nil {
		t.Fatalf("Abrir() error = %v, want nil", err)
	}

	if obtenida != session {
		t.Error("Abrir() no devolvió la sesión del conector SSH")
	}

	if gaConn.probed != 0 {
		t.Errorf("guest-additions se sondeó %d veces, want 0", gaConn.probed)
	}
}

// TestOpenFailsWithoutSession comprueba que Open propaga
// ErrSessionUnavailable cuando la máquina no puede atenderse.
func TestOpenFailsWithoutSession(t *testing.T) {
	t.Parallel()

	m := &stubMachine{state: domain.StatePoweredOff}

	_, err := newDiscoverer(m, &stubConnector{name: connectorSSH}).Open(context.Background())
	if !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("Abrir() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestOpenPropagatesErrorOfConnection comprueba que el fallo del conector al
// conectar se propaga envuelto.
func TestOpenPropagatesErrorOfConnection(t *testing.T) {
	t.Parallel()

	ip := netip.MustParseAddr("192.168.1.56")
	m := &stubMachine{addr: ip, state: domain.StateRunning}
	sshConn := &stubConnector{name: connectorSSH, connect: errors.New("ssh rechazó la llave")}

	_, err := newDiscoverer(m, sshConn).Open(context.Background())
	if err == nil {
		t.Fatal("Abrir() error = nil, want el error de conexión")
	}
}

// TestDiscovererAccessConcurrente ejercita el cambio de objetivo desde
// varias goroutines. Con -race detecta si se elimina la protección de
// mu, que los handlers necesitan para consultar y vincular en paralelo.
func TestDiscovererAccessConcurrente(t *testing.T) {
	t.Parallel()

	discoverer := NewDiscoverer(&DiscovererConfig{
		Machine:       &stubMachine{},
		CurrentTarget: domain.Target{ID: vmTests},
	})

	var espera sync.WaitGroup

	for i := 0; i < 50; i++ {
		espera.Add(2)

		go func(number int) {
			defer espera.Done()

			discoverer.Set(fmt.Sprintf("vm-%d", number))
		}(i)

		go func() {
			defer espera.Done()

			_ = discoverer.CurrentTarget()
		}()
	}

	espera.Wait()

	if target := discoverer.CurrentTarget(); target.ID == "" {
		t.Error("Objetivo().ID = vacío tras las vinculaciones concurrentes")
	}
}
