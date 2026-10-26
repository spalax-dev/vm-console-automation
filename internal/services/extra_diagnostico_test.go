// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"

	"consola-vm/internal/core/domain"
)

// stubRegistrationDiag informa si la máquina está registrada.
type stubRegistrationDiag struct{ registered bool }

// IsRegistered devuelve el estado configurado.
func (s *stubRegistrationDiag) IsRegistered(string) bool { return s.registered }

// newDiagnostics arma el servicio con la máquina y el objetivo dados.
func newDiagnostics(machine *stubMachine, registered bool, target string) *DiagnosticsService {
	return NewDiagnostics(&DiagnosticsConfig{
		Machine:       machine,
		Registration:  &stubRegistrationDiag{registered: registered},
		CurrentTarget: func() domain.Target { return domain.Target{ID: target} },
	})
}

// TestDiagnosticsWithStateUnreadable propaga el fallo de VirtualBox.
func TestDiagnosticsWithStateUnreadable(t *testing.T) {
	t.Parallel()

	machine := &stubMachine{stateErr: errors.New("sin VBoxManage")}

	if _, err := newDiagnostics(machine, false, "otra-maquina").Get(context.Background()); err == nil {
		t.Error("Obtener() error = nil, want el fallo del estado")
	}
}

// TestDiagnosticsOfMachineOff explica que está apagada.
func TestDiagnosticsOfMachineOff(t *testing.T) {
	t.Parallel()

	machine := &stubMachine{state: domain.StatePoweredOff}

	report, err := newDiagnostics(machine, false, testMachineName).Get(context.Background())
	if err != nil {
		t.Fatalf("Obtener() error = %v, want nil", err)
	}

	if len(report.Problemas) == 0 || report.Problemas[0] != "La máquina está apagada." {
		t.Errorf("problemas = %v, want el aviso de máquina apagada", report.Problemas)
	}

	if report.SSHOpen {
		t.Error("SSHAbierto = true para una máquina apagada")
	}
}

// TestDiagnosticsOfMachineRegistered avisa que no hace falta registrarla.
func TestDiagnosticsOfMachineRegistered(t *testing.T) {
	t.Parallel()

	machine := &stubMachine{state: domain.StateRunning, guest: domain.GuestAdditions{
		Version: "7.2.18", RunLevel: 2,
	}}

	report, err := newDiagnostics(machine, true, testMachineName).Get(context.Background())
	if err != nil {
		t.Fatalf("Obtener() error = %v, want nil", err)
	}

	if !report.IsRegistered {
		t.Error("Registrada = false, want true")
	}
}

// TestDiagnosticsWithoutGuestAdditions avisa de la versión ausente.
func TestDiagnosticsWithoutGuestAdditions(t *testing.T) {
	t.Parallel()

	machine := &stubMachine{state: domain.StateRunning}

	report, err := newDiagnostics(machine, false, testMachineName).Get(context.Background())
	if err != nil {
		t.Fatalf("Obtener() error = %v, want nil", err)
	}

	if len(report.Problemas) == 0 {
		t.Error("problemas vacío, want el aviso de Guest Additions")
	}
}

// TestDiagnosticsWithLevelLow avisa de un nivel incompleto.
func TestDiagnosticsWithLevelLow(t *testing.T) {
	t.Parallel()

	machine := &stubMachine{state: domain.StateRunning, guest: domain.GuestAdditions{
		Version: "7.2.18", RunLevel: 1,
	}}

	report, err := newDiagnostics(machine, false, testMachineName).Get(context.Background())
	if err != nil {
		t.Fatalf("Obtener() error = %v, want nil", err)
	}

	if len(report.Problemas) == 0 {
		t.Error("problemas vacío, want el aviso del nivel de Guest Additions")
	}
}

// TestDiagnosticsWithVersionesDifferent compara con el anfitrión.
func TestDiagnosticsWithVersionesDifferent(t *testing.T) {
	t.Parallel()

	machine := &stubMachine{state: domain.StateRunning, guest: domain.GuestAdditions{
		Version: "6.0.0", RunLevel: 2,
	}}

	service := NewDiagnostics(&DiagnosticsConfig{
		Machine:       machine,
		CurrentTarget: func() domain.Target { return domain.Target{ID: testMachineName} },
		VersionSystem: func(context.Context) (string, error) { return "7.2.18", nil },
	})

	report, err := service.Get(context.Background())
	if err != nil {
		t.Fatalf("Obtener() error = %v, want nil", err)
	}

	if len(report.Problemas) == 0 {
		t.Error("problemas vacío, want el aviso de versión distinta")
	}
}

// TestDiagnosticsWithVersionUnreadable no falla si el anfitrión no responde.
func TestDiagnosticsWithVersionUnreadable(t *testing.T) {
	t.Parallel()

	machine := &stubMachine{state: domain.StateRunning, guest: domain.GuestAdditions{
		Version: "7.2.18", RunLevel: 2,
	}}

	service := NewDiagnostics(&DiagnosticsConfig{
		Machine:       machine,
		CurrentTarget: func() domain.Target { return domain.Target{ID: testMachineName} },
		VersionSystem: func(context.Context) (string, error) { return "", errors.New("sin VBoxManage") },
	})

	if _, err := service.Get(context.Background()); err != nil {
		t.Errorf("Obtener() error = %v, want nil", err)
	}
}

// fakeConn simula una conexión TCP ya establecida.
type fakeConn struct {
	closeErr error
}

// Read no se usa.
func (fakeConn) Read([]byte) (int, error) { return 0, nil }

// Write no se usa.
func (fakeConn) Write(data []byte) (int, error) { return len(data), nil }

// Close devuelve el error configurado.
func (c fakeConn) Close() error { return c.closeErr }

// LocalAddr no se usa.
func (fakeConn) LocalAddr() net.Addr { return &net.TCPAddr{} }

// RemoteAddr no se usa.
func (fakeConn) RemoteAddr() net.Addr { return &net.TCPAddr{} }

// SetDeadline no se usa.
func (fakeConn) SetDeadline(time.Time) error { return nil }

// SetReadDeadline no se usa.
func (fakeConn) SetReadDeadline(time.Time) error { return nil }

// SetWriteDeadline no se usa.
func (fakeConn) SetWriteDeadline(time.Time) error { return nil }

// TestTestPortSSHWithSuccess cubre la conexión correcta.
func TestTestPortSSHWithSuccess(t *testing.T) {
	original := openConnection

	t.Cleanup(func() { openConnection = original })

	openConnection = func(context.Context, string) (net.Conn, error) { return fakeConn{}, nil }

	if !testPortSSH(context.Background(), netip.MustParseAddr("127.0.0.1")) {
		t.Error("probarPuertoSSH() = false, want true")
	}

	openConnection = func(context.Context, string) (net.Conn, error) {
		return fakeConn{closeErr: errors.New("ya cerrada")}, nil
	}

	if !testPortSSH(context.Background(), netip.MustParseAddr("127.0.0.1")) {
		t.Error("probarPuertoSSH() = false, want true aunque falle el cierre")
	}

	openConnection = func(context.Context, string) (net.Conn, error) { return nil, errors.New("sin ruta") }

	if testPortSSH(context.Background(), netip.MustParseAddr("127.0.0.1")) {
		t.Error("probarPuertoSSH() = true, want false sin conexión")
	}
}
