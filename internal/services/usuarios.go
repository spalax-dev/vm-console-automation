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
	"strconv"
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// ErrUserExists indica que la cuenta ya está en la máquina.
var ErrUserExists = errors.New("el usuario ya existe")

// ErrUserNotFound indica que la cuenta no está en la máquina.
var ErrUserNotFound = errors.New("el usuario no existe")

// ErrUserKeyNotFound indica que no se ha generado la llave del
// usuario.
var ErrUserKeyNotFound = errors.New("la llave del usuario no existe")

// ErrUserKeyExists indica que el usuario ya tiene un par de llaves.
var ErrUserKeyExists = errors.New("el usuario ya tiene un par de llaves")

// UsersConfig reúne las dependencias del servicio de usuarios.
type UsersConfig struct {
	// Sessions abre la sesión con la máquina.
	Sessions SessionOpener
	// Ejecutor lanza ssh-keygen en el anfitrión.
	Ejecutor ports.Executor
	// Directory guarda las llaves privadas de los usuarios.
	Directory string
}

// User describe una cuenta local de la máquina.
type User struct {
	// Name es el nombre de la cuenta.
	Name string
	// Sudo indica si pertenece al grupo sudo.
	Sudo bool
	// Keys indica si tiene un par de llaves generado.
	Keys bool
	// UID es el identificador numérico.
	UID int
	// Home es el directorio personal.
	Home string
}

// Users implementa el caso de uso de gestión de cuentas locales.
type Users struct {
	sessions  SessionOpener
	ejecutor  ports.Executor
	directory string
}

// NewUsers construye el servicio de usuarios con las dependencias
// dadas.
func NewUsers(config UsersConfig) *Users {
	return &Users{
		sessions:  config.Sessions,
		ejecutor:  config.Ejecutor,
		directory: config.Directory,
	}
}

// List devuelve las cuentas regulares (UID entre 1000 y 65533).
func (u *Users) List(ctx context.Context) ([]User, error) {
	session, err := u.sessions.Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrir la sesión para listar usuarios: %w", err)
	}

	defer closeSession(session)

	passwd, err := runOutput(ctx, session, binGetent, "passwd")
	if err != nil {
		return nil, fmt.Errorf("listar los usuarios: %w", err)
	}

	group, err := runOutput(ctx, session, binGetent, "group", "sudo")
	if err != nil {
		return nil, fmt.Errorf("consultar el grupo sudo: %w", err)
	}

	members := groupMembers(group)
	users := []User{}

	for _, line := range strings.Split(passwd, "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 7 {
			continue
		}

		uid, err := strconv.Atoi(fields[2])
		if err != nil || uid < 1000 || uid >= 65534 {
			continue
		}

		users = append(users, User{
			Name: fields[0],
			Sudo: members[fields[0]],
			Keys: u.sshKeyExists(fields[0]),
			UID:  uid,
			Home: fields[5],
		})
	}

	return users, nil
}

// Get devuelve una cuenta de la máquina.
func (u *Users) Get(ctx context.Context, name string) (User, error) {
	session, err := u.sessions.Open(ctx)
	if err != nil {
		return User{}, fmt.Errorf("abrir la sesión para consultar al usuario: %w", err)
	}

	defer closeSession(session)

	passwd, err := findUser(ctx, session, name)
	if err != nil {
		return User{}, err
	}

	if passwd == "" {
		return User{}, ErrUserNotFound
	}

	group, err := runOutput(ctx, session, binGetent, "group", "sudo")
	if err != nil {
		return User{}, fmt.Errorf("consultar el grupo sudo: %w", err)
	}

	user := parseUser(passwd)
	user.Sudo = groupMembers(group)[name]
	user.Keys = u.sshKeyExists(name)

	return user, nil
}

// Create crea una cuenta, la agrega a sudo si se pide y genera sus llaves
// cuando corresponde.
func (u *Users) Create(ctx context.Context, name string, sudo, keys bool, password string) (Outcome, error) {
	session, err := u.sessions.Open(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("abrir la sesión para crear el usuario: %w", err)
	}

	defer closeSession(session)

	existing, err := findUser(ctx, session, name)
	if err != nil {
		return Outcome{}, err
	}

	if existing != "" {
		return Outcome{}, ErrUserExists
	}

	guion := binUseradd + " -m -s /bin/bash " + domain.Quote(name)

	if sudo {
		guion += " && " + binUsermod + " -aG sudo " + domain.Quote(name)
	}

	cmd := &domain.Command{Path: binSh, Args: []string{"-c", guion}}

	result, err := session.Run(ctx, cmd)

	output := Outcome{
		Message:   "Usuario «" + name + "» creado.",
		Operation: cmd.ShellLine(),
		Output:    result.Stdout,
		Stderr:    result.Stderr,
		Code:      result.ExitCode,
	}

	if err != nil {
		return output, fmt.Errorf("crear el usuario %q: %w", name, err)
	}

	if password != "" {
		if err := u.setPassword(ctx, session, name, password); err != nil {
			return output, err
		}
	}

	if keys {
		if err := u.installSSHKeys(ctx, session, name, false); err != nil {
			return output, err
		}
	}

	return output, nil
}

// setPassword asigna la contraseña sin exponerla en la orden ni en los
// logs: la envía en un archivo temporal dentro de /root y ejecuta
// chpasswd.
func (u *Users) setPassword(ctx context.Context, session ports.Session, name, password string) error {
	file, err := os.CreateTemp("", "vmcons-pass-*")
	if err != nil {
		return fmt.Errorf("crear el archivo temporal de la contraseña: %w", err)
	}

	local := file.Name()
	defer func() { _ = os.Remove(local) }() //nolint:errcheck // el temporal no cambia el resultado.

	if err := file.Chmod(0o600); err != nil {
		_ = file.Close() //nolint:errcheck // se cierra antes de devolver el error.

		return fmt.Errorf("proteger el archivo temporal: %w", err)
	}

	if _, err := fmt.Fprintf(file, "%s:%s\n", name, password); err != nil {
		_ = file.Close() //nolint:errcheck // se cierra antes de devolver el error.

		return fmt.Errorf("escribir la contraseña temporal: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("cerrar el archivo temporal: %w", err)
	}

	remote := "/root/" + filepath.Base(local)

	if err := session.Put(ctx, local, remote); err != nil {
		return fmt.Errorf("transferir la contraseña temporal: %w", err)
	}

	guion := "chmod 600 " + domain.Quote(remote) +
		" && chpasswd < " + domain.Quote(remote) +
		"; rm -f " + domain.Quote(remote)

	cmd := &domain.Command{Path: binSh, Args: []string{"-c", guion}}

	if _, err := session.Run(ctx, cmd); err != nil {
		return fmt.Errorf("asignar la contraseña del usuario %q: %w", name, err)
	}

	return nil
}

// installSSHKeys genera el par del usuario, instala la pública y ajusta
// los permisos.
func (u *Users) installSSHKeys(ctx context.Context, session ports.Session, name string, replace bool) error {
	path := u.sshKeyPath(name)

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf(msgCreateKeyDir, err)
	}

	if err := generateKeyPair(ctx, u.ejecutor, path, "vmcons-"+name); err != nil {
		return err
	}

	key, err := readPublicKey(path)
	if err != nil {
		return err
	}

	home := "/home/" + name + "/.ssh"

	redireccion := ">>"
	if replace {
		redireccion = ">"
	}

	guion := "install -d -m 700 -o " + domain.Quote(name) + " -g " + domain.Quote(name) +
		" " + domain.Quote(home) +
		" && printf '%s\\n' " + domain.Quote(key) + " " + redireccion + " " + domain.Quote(home+"/authorized_keys") +
		" && chown " + domain.Quote(name+":"+name) + " " + domain.Quote(home+"/authorized_keys") +
		" && chmod 600 " + domain.Quote(home+"/authorized_keys")

	cmd := &domain.Command{Path: binSh, Args: []string{"-c", guion}}

	if _, err := session.Run(ctx, cmd); err != nil {
		return fmt.Errorf("instalar las llaves del usuario %q: %w", name, err)
	}

	return nil
}

// GenerateKey crea el par de llaves de un usuario existente, instala la
// pública en su authorized_keys y deshabilita el acceso por contraseña.
func (u *Users) GenerateKey(ctx context.Context, name string, regenerate bool) (Outcome, error) {
	session, err := u.sessions.Open(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("abrir la sesión para generar la llave: %w", err)
	}

	defer closeSession(session)

	passwdLine, err := findUser(ctx, session, name)
	if err != nil {
		return Outcome{}, err
	}

	if passwdLine == "" {
		return Outcome{}, ErrUserNotFound
	}

	hasKey := u.sshKeyExists(name)
	if hasKey && !regenerate {
		return Outcome{}, ErrUserKeyExists
	}

	if hasKey {
		u.deleteSSHKeys(name)
	}

	if err := u.installSSHKeys(ctx, session, name, regenerate); err != nil {
		u.deleteSSHKeys(name)

		return Outcome{}, err
	}

	return Outcome{
		Message: "Llave SSH de «" + name + "» generada e instalada.",
	}, nil
}

// Delete elimina una cuenta y, opcionalmente, sus archivos.
func (u *Users) Delete(ctx context.Context, name string, files bool) (Outcome, error) {
	session, err := u.sessions.Open(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("abrir la sesión para eliminar al usuario: %w", err)
	}

	defer closeSession(session)

	existing, err := findUser(ctx, session, name)
	if err != nil {
		return Outcome{}, err
	}

	if existing == "" {
		return Outcome{}, ErrUserNotFound
	}

	guion := binUserdel + " "

	if files {
		guion += "-r "
	}

	guion += domain.Quote(name)

	cmd := &domain.Command{Path: binSh, Args: []string{"-c", guion}}

	result, err := session.Run(ctx, cmd)

	output := Outcome{
		Message:   "Usuario «" + name + "» eliminado.",
		Operation: cmd.ShellLine(),
		Output:    result.Stdout,
		Stderr:    result.Stderr,
		Code:      result.ExitCode,
	}

	if err != nil {
		return output, fmt.Errorf("eliminar el usuario %q: %w", name, err)
	}

	u.deleteSSHKeys(name)

	return output, nil
}

// DownloadKey devuelve el nombre y el contenido de la llave privada
// del usuario.
func (u *Users) DownloadKey(name string) (file string, content []byte, err error) {
	path := u.sshKeyPath(name)

	// #nosec G304 -- la ruta se compone con un nombre de usuario validado.
	content, err = os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil, ErrUserKeyNotFound
	}

	if err != nil {
		return "", nil, fmt.Errorf("leer la llave del usuario: %w", err)
	}

	return filepath.Base(path), content, nil
}

// sshKeyPath devuelve la ruta de la llave privada del usuario.
func (u *Users) sshKeyPath(name string) string {
	return filepath.Join(u.directory, name+"_ed25519")
}

// sshKeyExists informa si la llave privada del usuario está en el
// anfitrión.
func (u *Users) sshKeyExists(name string) bool {
	_, err := os.Stat(u.sshKeyPath(name))

	return err == nil
}

// deleteSSHKeys elimina del anfitrión el par de llaves del usuario.
func (u *Users) deleteSSHKeys(name string) {
	path := u.sshKeyPath(name)

	for _, file := range []string{path, path + ".pub"} {
		if err := os.Remove(file); err != nil && !errors.Is(err, fs.ErrNotExist) {
			continue
		}
	}
}

// findUser devuelve la línea de getent passwd o vacío si no existe.
func findUser(ctx context.Context, session ports.Session, name string) (string, error) {
	result, err := session.Run(ctx, &domain.Command{
		Path: binGetent,
		Args: []string{"passwd", name},
	})
	if err != nil {
		var exitErr *domain.ExitError
		if errors.As(err, &exitErr) && exitErr.Code == 2 {
			return "", nil
		}

		return "", fmt.Errorf("consultar al usuario %q: %w", name, err)
	}

	return strings.TrimSpace(result.Stdout), nil
}

// parseUser extrae los datos de una línea de getent passwd.
func parseUser(line string) User {
	fields := strings.Split(line, ":")
	user := User{}

	if len(fields) < 7 {
		return user
	}

	uid, err := strconv.Atoi(fields[2])
	if err != nil {
		return User{}
	}

	user.Name = fields[0]
	user.UID = uid
	user.Home = fields[5]

	return user
}

// groupMembers devuelve los miembros de la salida de getent group.
func groupMembers(output string) map[string]bool {
	members := map[string]bool{}

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 4 {
			continue
		}

		for _, member := range strings.Split(fields[3], ",") {
			if member != "" {
				members[member] = true
			}
		}
	}

	return members
}
