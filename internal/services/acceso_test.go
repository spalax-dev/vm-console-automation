// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// newSSHKeyOfTests crea un par de llaves falso en el anfitrión.
func newSSHKeyOfTests(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), nameSSHKey)
	if err := os.WriteFile(path, []byte(contentPrivate), 0o600); err != nil {
		t.Fatalf(msgCreateDir, err)
	}

	if err := os.WriteFile(path+".pub", []byte(keyPublic+"\n"), 0o600); err != nil {
		t.Fatalf(msgWritePub, err)
	}

	return path
}

// newAccessForTest arma el servicio con máquina, conectores y llave
// falsos.
func newAccessForTest(t *testing.T, state domain.MachineState, session *stubSession, probeErr error) (*Access, *Store) {
	t.Helper()

	store, _, _ := newTestDB(t)

	machine := &stubMachine{addr: netip.MustParseAddr(ipTests), state: state}
	guest := &stubConnector{name: connectorGuest, session: session}
	connectorSSH := &stubConnector{name: connectorSSH, probe: probeErr}

	access := NewAccess(&AccessConfig{
		Store:         store,
		Machine:       machine,
		Guest:         guest,
		SSH:           connectorSSH,
		Ejecutor:      &stubExecutor{},
		PathSSHKey:    newSSHKeyOfTests(t),
		CurrentTarget: func() domain.Target { return domain.Target{ID: testMachineName} },
	})

	return access, store
}

// TestAccessRegisterMarksMachine comprueba el registro completo.
func TestAccessRegisterMarksMachine(t *testing.T) {
	t.Parallel()

	access, store := newAccessForTest(t, domain.StateRunning, &stubSession{}, nil)

	result, err := access.Register(context.Background(), userRoot, passwordTests)
	if err != nil {
		t.Fatalf("Registrar() error = %v, want nil", err)
	}

	if result.Message == "" {
		t.Error("Mensaje vacío, want confirmación")
	}

	credentials, ok := store.Get(testMachineName)
	if !ok || !credentials.IsRegistered {
		t.Errorf("credenciales = %+v, want registrada", credentials)
	}

	if !store.IsRegistered(testMachineName) {
		t.Error("Registrada() = false, want true")
	}
}

// TestAccessRegisterCredentialsInvalid restaura el estado si falla
// la verificación.
func TestAccessRegisterCredentialsInvalid(t *testing.T) {
	t.Parallel()

	session := &stubSession{err: &domain.ExitError{Code: 1, Command: "id"}}
	access, store := newAccessForTest(t, domain.StateRunning, session, nil)

	if _, err := access.Register(context.Background(), userRoot, passwordTests); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Registrar() error = %v, want ErrCredencialesInvalidas", err)
	}

	if _, ok := store.Get(testMachineName); ok {
		t.Error("se guardó una máquina sin registrar")
	}
}

// TestAccessRegisterWithoutSSH falla si la llave no queda operativa.
func TestAccessRegisterWithoutSSH(t *testing.T) {
	t.Parallel()

	access, store := newAccessForTest(t, domain.StateRunning, &stubSession{}, errors.New("ssh no responde"))

	if _, err := access.Register(context.Background(), userRoot, passwordTests); err == nil {
		t.Fatal("Registrar() error = nil, want el fallo de verificación SSH")
	}

	if _, ok := store.Get(testMachineName); ok {
		t.Error("se registró una máquina sin SSH")
	}
}

// TestAccessRegisterWithoutMachine comprueba la consola sin vínculo.
func TestAccessRegisterWithoutMachine(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	access := NewAccess(&AccessConfig{
		Store:         store,
		CurrentTarget: func() domain.Target { return domain.Target{} },
	})

	if _, err := access.Register(context.Background(), userRoot, passwordTests); !errors.Is(err, ErrNoMachine) {
		t.Fatalf("Registrar() error = %v, want ErrSinMaquina", err)
	}
}

// TestAccessRegisterMachineOff comprueba la VM apagada.
func TestAccessRegisterMachineOff(t *testing.T) {
	t.Parallel()

	access, store := newAccessForTest(t, domain.StatePoweredOff, &stubSession{}, nil)

	if _, err := access.Register(context.Background(), userRoot, passwordTests); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("Registrar() error = %v, want ErrSesionNoDisponible", err)
	}

	if _, ok := store.Get(testMachineName); ok {
		t.Error("Obtener() encontró credenciales de una máquina apagada")
	}
}

// TestAccessForget elimina las credenciales guardadas.
func TestAccessForget(t *testing.T) {
	t.Parallel()

	access, store := newAccessForTest(t, domain.StateRunning, &stubSession{}, nil)

	if err := store.Save(testMachineName, Credentials{User: userRoot, IsRegistered: true}); err != nil {
		t.Fatalf("Guardar() error = %v", err)
	}

	if err := access.Forget(); err != nil {
		t.Fatalf("Olvidar() error = %v, want nil", err)
	}

	if _, ok := store.Get(testMachineName); ok {
		t.Error("Obtener() encontró credenciales olvidadas")
	}
}

// stubPasswordConnector abre una sesión falsa con contraseña.
type stubPasswordConnector struct {
	session  ports.Session
	err      error
	user     string
	password string
	llamadas int
}

// ConnectPassword registra las credenciales y devuelve la sesión.
func (s *stubPasswordConnector) ConnectPassword(
	_ context.Context, _ domain.Target, user, password string,
) (ports.Session, error) {
	s.user = user
	s.password = password
	s.llamadas++

	return s.session, s.err
}

// TestAccessRegisterWithSSHPassword comprueba el respaldo cuando Guest
// Additions rechaza las credenciales.
func TestAccessRegisterWithSSHPassword(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	machine := &stubMachine{addr: netip.MustParseAddr(ipTests), state: domain.StateRunning}
	guest := &stubConnector{name: connectorGuest, session: &stubSession{err: &domain.ExitError{Code: 1, Command: "id"}}}
	sshSession := &stubSession{}
	sshPassword := &stubPasswordConnector{session: sshSession}

	access := NewAccess(&AccessConfig{
		Store:         store,
		Machine:       machine,
		Guest:         guest,
		SSH:           &stubConnector{name: connectorSSH},
		SSHPassword:   sshPassword,
		Ejecutor:      &stubExecutor{},
		PathSSHKey:    newSSHKeyOfTests(t),
		CurrentTarget: func() domain.Target { return domain.Target{ID: testMachineName} },
	})

	if _, err := access.Register(context.Background(), userRoot, passwordTests); err != nil {
		t.Fatalf("Registrar() error = %v, want nil", err)
	}

	if sshPassword.llamadas != 1 {
		t.Errorf("ConnectPassword() se llamó %d veces, want 1", sshPassword.llamadas)
	}

	if !store.IsRegistered(testMachineName) {
		t.Error("la máquina no quedó registrada con el respaldo SSH")
	}
}
