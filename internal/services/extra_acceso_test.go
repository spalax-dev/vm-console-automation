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
)

// accessWith arma un servicio de registro con las piezas indicadas.
func accessWith(
	t *testing.T, store *Store, machine *stubMachine, guest *stubConnector,
	sshConnector *stubConnector, sshPassword *stubPasswordConnector, sshKeyPath string,
) *Access {
	t.Helper()

	return NewAccess(&AccessConfig{
		Store:         store,
		Machine:       machine,
		Guest:         guest,
		SSH:           sshConnector,
		SSHPassword:   sshPassword,
		Ejecutor:      &stubExecutor{},
		PathSSHKey:    sshKeyPath,
		CurrentTarget: func() domain.Target { return domain.Target{ID: testMachineName} },
	})
}

// TestAccessSaveCredentialsFails propaga el fallo de SQLite.
func TestAccessSaveCredentialsFails(t *testing.T) {
	t.Parallel()

	store := withClosedDB(t)
	access := accessWith(t, store, &stubMachine{addr: netip.MustParseAddr(ipTests), state: domain.StateRunning},
		&stubConnector{}, &stubConnector{}, nil, newSSHKeyOfTests(t))

	if _, err := access.Register(context.Background(), userRoot, passwordTests); err == nil {
		t.Error("Registrar() error = nil, want el fallo al guardar")
	}
}

// TestAccessSaveRegistrationFinalFails propaga el fallo al marcar el
// registro.
func TestAccessSaveRegistrationFinalFails(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)

	if _, err := store.db.ExecContext(context.Background(),
		`CREATE TRIGGER bloquear_update BEFORE UPDATE ON credenciales
		 BEGIN SELECT RAISE(ABORT, 'sin escritura'); END;`); err != nil {
		t.Fatalf("crear el disparador: %v", err)
	}

	access := accessWith(t, store, &stubMachine{addr: netip.MustParseAddr(ipTests), state: domain.StateRunning},
		&stubConnector{session: &stubSession{}}, &stubConnector{}, nil, newSSHKeyOfTests(t))

	if _, err := access.Register(context.Background(), userRoot, passwordTests); err == nil {
		t.Error("Registrar() error = nil, want el fallo al marcar el registro")
	}

	if _, ok := store.Get(testMachineName); ok {
		t.Error("Obtener() encontró credenciales de un registro fallido")
	}
}

// TestAccessResolveErrors propaga los fallos de VirtualBox.
func TestAccessResolveErrors(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)

	withoutState := accessWith(t, store, &stubMachine{stateErr: errors.New("sin VBoxManage")},
		&stubConnector{}, &stubConnector{}, nil, newSSHKeyOfTests(t))

	if _, err := withoutState.Register(context.Background(), userRoot, passwordTests); err == nil {
		t.Error("Registrar() error = nil, want el fallo del estado")
	}

	withoutAddress := accessWith(t, store, &stubMachine{state: domain.StateRunning, addrErr: errors.New("sin IP")},
		&stubConnector{}, &stubConnector{}, nil, newSSHKeyOfTests(t))

	if _, err := withoutAddress.Register(context.Background(), userRoot, passwordTests); err == nil {
		t.Error("Registrar() error = nil, want el fallo de la dirección")
	}
}

// TestAccessGuestAdditionsNotConecta propaga el fallo del conector.
func TestAccessGuestAdditionsNotConecta(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	guest := &stubConnector{connect: errors.New("sin Guest Additions")}

	access := accessWith(t, store, &stubMachine{addr: netip.MustParseAddr(ipTests), state: domain.StateRunning},
		guest, &stubConnector{}, nil, newSSHKeyOfTests(t))

	if _, err := access.Register(context.Background(), userRoot, passwordTests); err == nil {
		t.Error("Registrar() error = nil, want el fallo del conector")
	}
}

// TestAccessRespaldoSSHFails cubre el respaldo por contraseña cuando la
// llave no está disponible o el conector falla.
func TestAccessRespaldoSSHFails(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	guest := &stubConnector{session: &stubSession{err: &domain.ExitError{Code: 1, Command: "id"}}}

	// La llave vive bajo un archivo: comprobar su estado falla antes de
	// intentar el respaldo por contraseña.
	pathImpossible := filepath.Join(fileWithText(t), "id_ed25519")
	access := accessWith(t, store, &stubMachine{addr: netip.MustParseAddr(ipTests), state: domain.StateRunning},
		guest, &stubConnector{probe: errors.New("sin llave")}, &stubPasswordConnector{}, pathImpossible)

	if _, err := access.Register(context.Background(), userRoot, passwordTests); err == nil {
		t.Error("Registrar() error = nil, want el fallo al comprobar la llave")
	}

	if _, err := access.prepareSSHPassword(context.Background(),
		domain.Target{ID: testMachineName}, netip.MustParseAddr(ipTests), userRoot, passwordTests,
	); err == nil {
		t.Error("prepararSSHPassword() error = nil, want el fallo al comprobar la llave")
	}

	// El conector de contraseña falla al abrir la sesión.
	otherStore, _, _ := newTestDB(t)
	withError := accessWith(t, otherStore, &stubMachine{addr: netip.MustParseAddr(ipTests), state: domain.StateRunning},
		guest, &stubConnector{probe: errors.New("sin llave")}, &stubPasswordConnector{err: errors.New("sin sshpass")}, newSSHKeyOfTests(t))

	if _, err := withError.Register(context.Background(), userRoot, passwordTests); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Registrar() error = %v, want ErrCredencialesInvalidas", err)
	}

	// La sesión por contraseña rechaza la orden de verificación.
	tercerStore, _, _ := newTestDB(t)
	sessionRota := &stubSession{err: &domain.ExitError{Code: 1, Command: "id"}}
	withSessionRota := accessWith(t, tercerStore, &stubMachine{addr: netip.MustParseAddr(ipTests), state: domain.StateRunning},
		guest, &stubConnector{probe: errors.New("sin llave")}, &stubPasswordConnector{session: sessionRota}, newSSHKeyOfTests(t))

	if _, err := withSessionRota.Register(context.Background(), userRoot, passwordTests); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Registrar() error = %v, want ErrCredencialesInvalidas", err)
	}
}

// TestAccessInstallationFails propaga el fallo al preparar SSH.
func TestAccessInstallationFails(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	session := &stubSession{errs: []error{nil, errors.New("sin espacio")}}

	access := accessWith(t, store, &stubMachine{addr: netip.MustParseAddr(ipTests), state: domain.StateRunning},
		&stubConnector{session: session}, &stubConnector{}, nil, newSSHKeyOfTests(t))

	if _, err := access.Register(context.Background(), userRoot, passwordTests); err == nil {
		t.Error("Registrar() error = nil, want el fallo de la instalación")
	}

	if _, ok := store.Get(testMachineName); ok {
		t.Error("Obtener() encontró credenciales tras fallar la instalación")
	}
}

// TestAccessRegisterFailedRestoresCredentialsPrevious comprueba que
// un registro fallido no pisa las credenciales que ya funcionaban.
func TestAccessRegisterFailedRestoresCredentialsPrevious(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)

	previous := Credentials{
		User: "jorge", Password: "anterior", Port: 2222, Address: "10.0.0.9", IsRegistered: true,
	}
	if err := store.Save(testMachineName, previous); err != nil {
		t.Fatalf("Guardar() error = %v, want nil", err)
	}

	guest := &stubConnector{session: &stubSession{err: &domain.ExitError{Code: 1, Command: "id"}}}
	access := accessWith(t, store, &stubMachine{addr: netip.MustParseAddr(ipTests), state: domain.StateRunning},
		guest, &stubConnector{probe: errors.New("sin llave")}, &stubPasswordConnector{err: errors.New("sin sshpass")}, newSSHKeyOfTests(t))

	if _, err := access.Register(context.Background(), userRoot, passwordTests); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Registrar() error = %v, want ErrCredencialesInvalidas", err)
	}

	guardadas, ok := store.Get(testMachineName)
	if !ok {
		t.Fatal("Obtener() no encontró las credenciales anteriores")
	}

	if guardadas != previous {
		t.Errorf("Obtener() = %+v, want %+v", guardadas, previous)
	}
}

// TestAccessForgetErrors cubre la ausencia de máquina y el fallo de
// SQLite.
func TestAccessForgetErrors(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	withoutMachine := NewAccess(&AccessConfig{
		Store:         store,
		CurrentTarget: func() domain.Target { return domain.Target{} },
	})

	if err := withoutMachine.Forget(); !errors.Is(err, ErrNoMachine) {
		t.Errorf("Olvidar() error = %v, want ErrSinMaquina", err)
	}

	access := accessWith(t, withClosedDB(t), &stubMachine{}, &stubConnector{}, &stubConnector{}, nil, "")

	if err := access.Forget(); err == nil {
		t.Error("Olvidar() error = nil, want el fallo de SQLite")
	}
}

// TestEnsureSSHKeyErrors cubre los fallos al preparar la llave.
func TestEnsureSSHKeyErrors(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)

	pathImpossible := filepath.Join(fileWithText(t), "id_ed25519")
	if _, err := accessWith(t, store, &stubMachine{}, &stubConnector{}, &stubConnector{}, nil,
		pathImpossible).ensureSSHKey(context.Background()); err == nil {
		t.Error("asegurarLlave() error = nil, want el fallo de comprobación")
	}

	pathLowFile := filepath.Join(fileWithText(t), "sub", "id_ed25519")
	if _, err := accessWith(t, store, &stubMachine{}, &stubConnector{}, &stubConnector{}, nil,
		pathLowFile).ensureSSHKey(context.Background()); err == nil {
		t.Error("asegurarLlave() error = nil, want el fallo del directorio")
	}

	access := accessWith(t, store, &stubMachine{}, &stubConnector{}, &stubConnector{}, nil,
		filepath.Join(t.TempDir(), "id_ed25519"))
	access.ejecutor = &stubExecutor{err: errors.New("sin ssh-keygen")}

	if _, err := access.ensureSSHKey(context.Background()); err == nil {
		t.Error("asegurarLlave() error = nil, want el fallo del generador")
	}

	// El generador crea la llave pública: el camino correcto la devuelve.
	path := filepath.Join(t.TempDir(), nameSSHKey)
	generador := &stubExecutor{onExecute: func() {
		if err := os.WriteFile(path+".pub", []byte(keyPublic+"\n"), 0o600); err != nil {
			t.Errorf("escribir la pública: %v", err)
		}
	}}
	completo := accessWith(t, store, &stubMachine{}, &stubConnector{}, &stubConnector{}, nil, path)
	completo.ejecutor = generador

	key, err := completo.ensureSSHKey(context.Background())
	if err != nil || key != keyPublic {
		t.Errorf("asegurarLlave() = %q, %v, want la clave pública", key, err)
	}
}
