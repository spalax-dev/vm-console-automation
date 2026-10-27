// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// ErrKeyExists indica que la llave privada de root ya está en el
// anfitrión y no se sobrescribe.
var ErrKeyExists = errors.New("la llave SSH de root ya existe")

// ErrKeyNotFound indica que todavía no se ha generado la llave de
// root.
var ErrKeyNotFound = errors.New("la llave SSH de root no existe")

// KeysConfig reúne las dependencias del servicio de llaves.
type KeysConfig struct {
	// Sessions abre la sesión con la máquina para instalar la llave.
	Sessions SessionOpener
	// Ejecutor lanza ssh-keygen en el equipo anfitrión.
	Ejecutor ports.Executor
	// Path es la ruta de la llave privada en el anfitrión.
	Path string
}

// Keys implementa el caso de uso de las llaves SSH de root.
type Keys struct {
	sessions SessionOpener
	ejecutor ports.Executor
	path     string
}

// NewKeys construye el servicio de llaves con las dependencias dadas.
func NewKeys(config KeysConfig) *Keys {
	return &Keys{
		sessions: config.Sessions,
		ejecutor: config.Ejecutor,
		path:     config.Path,
	}
}

// Generate crea el par de llaves en el anfitrión, instala la pública en
// authorized_keys y deshabilita el acceso por contraseña. Devuelve
// ErrKeyExists si la llave privada ya está presente.
func (l *Keys) Generate(ctx context.Context) (Outcome, error) {
	if _, err := os.Stat(l.path); err == nil {
		return Outcome{}, ErrKeyExists
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Outcome{}, fmt.Errorf("comprobar la llave %q: %w", l.path, err)
	}

	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return Outcome{}, fmt.Errorf(msgCreateKeyDir, err)
	}

	// La sesión se abre antes de generar la llave: mientras no exista, el
	// descubrimiento usa Guest Additions, único canal disponible.
	session, err := l.sessions.Open(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("abrir la sesión con la máquina: %w", err)
	}

	defer closeSession(session)

	if err := generateKeyPair(ctx, l.ejecutor, l.path, "vmcons-root"); err != nil {
		return Outcome{}, err
	}

	key, err := readPublicKey(l.path)
	if err != nil {
		return Outcome{}, err
	}

	cmd := installScript(key)

	result, err := session.Run(ctx, cmd)
	output := Outcome{
		Message:   "Llave SSH de root generada e instalada; el acceso por contraseña quedó deshabilitado.",
		Operation: cmd.ShellLine(),
		Output:    result.Stdout,
		Stderr:    result.Stderr,
		Code:      result.ExitCode,
	}

	if err != nil {
		return output, fmt.Errorf("instalar la llave y configurar sshd: %w", err)
	}

	return output, nil
}

// Exists informa si la llave privada de root está en el anfitrión.
func (l *Keys) Exists() bool {
	_, err := os.Stat(l.path)

	return err == nil
}

// Download devuelve el nombre y el contenido de la llave privada, sin
// exponerla en los logs.
func (l *Keys) Download() (name string, content []byte, err error) {
	content, err = os.ReadFile(l.path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil, ErrKeyNotFound
	}

	if err != nil {
		return "", nil, fmt.Errorf("leer la llave privada: %w", err)
	}

	return filepath.Base(l.path), content, nil
}

// generateKeyPair crea un par de llaves ed25519 en el anfitrión.
func generateKeyPair(ctx context.Context, ejecutor ports.Executor, path, comentario string) error {
	if _, err := ejecutor.Execute(
		ctx, "ssh-keygen", "-t", "ed25519", "-N", "", "-f", path, "-C", comentario,
	); err != nil {
		return fmt.Errorf("generar el par de llaves: %w", err)
	}

	return nil
}

// readPublicKey devuelve el contenido de la llave pública sin el
// salto de línea final.
func readPublicKey(path string) (string, error) {
	// #nosec G304 -- la ruta proviene de la configuración de la consola.
	key, err := os.ReadFile(path + ".pub")
	if err != nil {
		return "", fmt.Errorf("leer la llave pública: %w", err)
	}

	return strings.TrimSpace(string(key)), nil
}

// installScript construye la orden POSIX que instala la llave pública
// y deshabilita el acceso por contraseña.
func installScript(key string) *domain.Command {
	return &domain.Command{Path: binSh, Args: []string{"-c", scriptInstallation(key)}}
}

// sudoInstallScript ejecuta la adecuación con sudo cuando el usuario
// de Guest Additions no es root. La contraseña se envía por la entrada
// estándar de sudo y no forma parte de la operación que se devuelve.
func sudoInstallScript(key, password string) *domain.Command {
	internal := "printf '%s\\n' " + domain.Quote(password) +
		" | sudo -S -p '' " + binSh + " -c " + domain.Quote(scriptInstallation(key))

	return &domain.Command{Path: binSh, Args: []string{"-c", internal}}
}

// scriptInstallation devuelve el guion que prepara SSH dentro de la VM.
func scriptInstallation(key string) string {
	config := "PasswordAuthentication no\nPermitRootLogin prohibit-password\n"

	return "if [ ! -x /usr/sbin/sshd ]; then " +
		"DEBIAN_FRONTEND=noninteractive apt-get install -y openssh-server; fi && " +
		"install -d -m 700 /root/.ssh && " +
		"printf '%s\\n' " + domain.Quote(key) + " >> /root/.ssh/authorized_keys && " +
		"chmod 600 /root/.ssh/authorized_keys && " +
		"install -d -m 755 /etc/ssh/sshd_config.d && " +
		"printf '%s' " + domain.Quote(config) + " > /etc/ssh/sshd_config.d/99-vmcons.conf && " +
		"systemctl enable --now ssh && " +
		"(systemctl reload ssh 2>/dev/null || systemctl restart ssh) && " +
		"/usr/sbin/sshd -T | grep -qx 'passwordauthentication no'"
}
