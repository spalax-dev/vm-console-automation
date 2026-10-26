// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// ErrInvalidCredentials indica que Guest Additions rechazó el
// usuario o la contraseña.
var ErrInvalidCredentials = errors.New("las credenciales no permiten ejecutar órdenes")

// AccessConfig reúne las dependencias para registrar una máquina en la
// consola.
type AccessConfig struct {
	// Store guarda las credenciales y el estado de registro.
	Store *Store
	// Machine consulta el estado y la dirección de la máquina activa.
	Machine ports.Machine
	// Guest abre una sesión por Guest Additions con las credenciales
	// guardadas.
	Guest ports.Connector
	// SSH comprueba que la llave quede operativa tras la adecuación.
	SSH ports.Connector
	// SSHPassword abre una sesión con usuario y contraseña cuando Guest
	// Additions no puede autenticar.
	SSHPassword ports.PasswordConnector
	// Ejecutor genera la llave de root en el anfitrión.
	Ejecutor ports.Executor
	// RutaLlave es la llave privada de root en el anfitrión.
	PathSSHKey string
	// CurrentTarget devuelve la máquina vinculada.
	CurrentTarget func() domain.Target
}

// Access registra una máquina: guarda sus credenciales, prepara SSH y
// deja constancia del estado para no volver a pedirlas.
type Access struct {
	store       *Store
	machine     ports.Machine
	guest       ports.Connector
	ssh         ports.Connector
	sshPassword ports.PasswordConnector
	ejecutor    ports.Executor
	sshKeyPath  string
	target      func() domain.Target
}

// NewAccess construye el servicio de registro.
func NewAccess(config *AccessConfig) *Access {
	return &Access{
		store:       config.Store,
		machine:     config.Machine,
		guest:       config.Guest,
		ssh:         config.SSH,
		sshPassword: config.SSHPassword,
		ejecutor:    config.Ejecutor,
		sshKeyPath:  config.PathSSHKey,
		target:      config.CurrentTarget,
	}
}

// Register guarda las credenciales, instala SSH con la llave de root y
// verifica la conexión. Si algo falla, deja la máquina sin registrar y
// restaura las credenciales anteriores.
func (a *Access) Register(ctx context.Context, user, password string) (Outcome, error) {
	target, address, err := a.resolve(ctx)
	if err != nil {
		return Outcome{}, err
	}

	previous, existia := a.store.Get(target.ID)

	if err := a.store.Save(target.ID, Credentials{
		User:         user,
		Password:     password,
		Port:         22,
		IsRegistered: false,
	}); err != nil {
		return Outcome{}, err
	}

	success := false

	defer func() {
		if !success {
			a.restore(target.ID, previous, existia)
		}
	}()

	output, err := a.prepareSSH(ctx, target, address, password)
	if errors.Is(err, ErrInvalidCredentials) {
		errorGuest := err

		output, err = a.prepareSSHPassword(ctx, target, address, user, password)
		if err != nil {
			err = fmt.Errorf("%w (Guest Additions: %w; SSH: %w)", ErrInvalidCredentials, errorGuest, err)
		}
	}

	if err != nil {
		slog.Warn("registrar la máquina", "vm", target.ID, "usuario", user, logErrorKey, err)

		return output, err
	}

	if err := a.store.Save(target.ID, Credentials{
		User:         user,
		Password:     password,
		Port:         22,
		IsRegistered: true,
	}); err != nil {
		return output, err
	}

	success = true

	return output, nil
}

// prepareSSH valida las credenciales, instala SSH con la llave de root
// y comprueba que la conexión por llave funcione.
func (a *Access) prepareSSH(
	ctx context.Context, target domain.Target, address netip.Addr, password string,
) (Outcome, error) {
	key, err := a.ensureSSHKey(ctx)
	if err != nil {
		return Outcome{}, err
	}

	session, err := a.guest.Connect(ctx, domain.Target{ID: target.ID, Address: address})
	if err != nil {
		return Outcome{}, fmt.Errorf("abrir la sesión de Guest Additions: %w", err)
	}

	defer closeSession(session)

	verificacion, err := session.Run(ctx, &domain.Command{Path: "/usr/bin/id", Args: []string{"-u"}})
	if err != nil {
		return credentialsError(verificacion, err)
	}

	return a.installWithSession(ctx, session, target, address, key, strings.TrimSpace(verificacion.Stdout), password)
}

// prepareSSHPassword registra una máquina usando SSH con contraseña,
// como respaldo cuando Guest Additions rechaza las credenciales.
func (a *Access) prepareSSHPassword(
	ctx context.Context, target domain.Target, address netip.Addr, user, password string,
) (Outcome, error) {
	if a.sshPassword == nil {
		return Outcome{}, fmt.Errorf("%w: no hay respaldo SSH disponible", ErrInvalidCredentials)
	}

	key, err := a.ensureSSHKey(ctx)
	if err != nil {
		return Outcome{}, err
	}

	session, err := a.sshPassword.ConnectPassword(
		ctx, domain.Target{ID: target.ID, Address: address}, user, password,
	)
	if err != nil {
		return Outcome{}, fmt.Errorf("%w: %w", ErrInvalidCredentials, err)
	}

	defer closeSession(session)

	verificacion, err := session.Run(ctx, &domain.Command{Path: "/usr/bin/id", Args: []string{"-u"}})
	if err != nil {
		return credentialsError(verificacion, err)
	}

	return a.installWithSession(ctx, session, target, address, key, strings.TrimSpace(verificacion.Stdout), password)
}

// installWithSession ejecuta la adecuación dentro de una sesión ya
// autenticada y verifica la llave.
func (a *Access) installWithSession(
	ctx context.Context, session ports.Session, target domain.Target, address netip.Addr,
	key, uid, password string,
) (Outcome, error) {
	cmd := installScript(key)
	operation := cmd.ShellLine()

	if uid != "0" {
		cmd = sudoInstallScript(key, password)
		operation = "instalar SSH y configurar sshd con sudo"
	}

	result, err := session.Run(ctx, cmd)
	output := Outcome{
		Message:   "Máquina registrada: SSH quedó preparado para su administración.",
		Operation: operation,
		Output:    result.Stdout,
		Stderr:    result.Stderr,
		Code:      result.ExitCode,
	}

	if err != nil {
		return output, fmt.Errorf("preparar SSH en la máquina: %w", err)
	}

	if err := a.ssh.Probe(ctx, domain.Target{ID: target.ID, Address: address}); err != nil {
		return output, fmt.Errorf("verificar el acceso SSH: %w", err)
	}

	return output, nil
}

// credentialsError arma el error de autenticación con el stderr de la
// orden de verificación.
func credentialsError(result domain.Result, err error) (Outcome, error) {
	detail := strings.TrimSpace(result.Stderr)
	if detail == "" {
		detail = err.Error()
	}

	return Outcome{
		Code:   result.ExitCode,
		Stderr: result.Stderr,
	}, fmt.Errorf("%w: %s", ErrInvalidCredentials, detail)
}

// restore deja el almacén como estaba antes del intento de registro.
func (a *Access) restore(vm string, previous Credentials, existia bool) {
	if existia {
		if err := a.store.Save(vm, previous); err != nil {
			slog.Warn("restaurar las credenciales", logErrorKey, err)
		}

		return
	}

	if err := a.store.Delete(vm); err != nil {
		slog.Warn("borrar las credenciales", logErrorKey, err)
	}
}

// ensureSSHKey devuelve la llave pública de root, generando el par si
// todavía no existe en el anfitrión.
func (a *Access) ensureSSHKey(ctx context.Context) (string, error) {
	if _, err := os.Stat(a.sshKeyPath); err == nil {
		return readPublicKey(a.sshKeyPath)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("comprobar la llave %q: %w", a.sshKeyPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(a.sshKeyPath), 0o700); err != nil {
		return "", fmt.Errorf(msgCreateKeyDir, err)
	}

	if err := generateKeyPair(ctx, a.ejecutor, a.sshKeyPath, "vmcons-root"); err != nil {
		return "", err
	}

	return readPublicKey(a.sshKeyPath)
}

// Forget elimina el registro y las credenciales de la máquina activa.
func (a *Access) Forget() error {
	target := a.target()
	if target.ID == "" {
		return ErrNoMachine
	}

	return a.store.Delete(target.ID)
}

// resolve comprueba la máquina activa y devuelve su dirección.
func (a *Access) resolve(ctx context.Context) (domain.Target, netip.Addr, error) {
	target := a.target()
	if target.ID == "" {
		return domain.Target{}, netip.Addr{}, fmt.Errorf("%w: elige una máquina del inventario", ErrNoMachine)
	}

	state, err := a.machine.State(ctx, target)
	if err != nil {
		return domain.Target{}, netip.Addr{}, fmt.Errorf(msgStateMachine, err)
	}

	if state != domain.StateRunning {
		return domain.Target{}, netip.Addr{}, fmt.Errorf(
			"%w: la máquina %q está en estado %q", ErrSessionUnavailable, target.ID, state,
		)
	}

	address, err := a.machine.Address(ctx, target)
	if err != nil {
		return domain.Target{}, netip.Addr{}, fmt.Errorf("descubrir la dirección de la máquina: %w", err)
	}

	return target, address, nil
}
