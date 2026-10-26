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
func (u *Users) Create(ctx context.Context, name string, sudo, keys bool) (Outcome, error) {
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

	if keys {
		guion += " && " + binPasswd + " -l " + domain.Quote(name)
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

	if keys {
		if err := u.installSSHKeys(ctx, session, name); err != nil {
			return output, err
		}
	}

	return output, nil
}

// installSSHKeys genera el par del usuario, instala la pública y ajusta
// los permisos.
func (u *Users) installSSHKeys(ctx context.Context, session ports.Session, name string) error {
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
	guion := "install -d -m 700 -o " + domain.Quote(name) + " -g " + domain.Quote(name) +
		" " + domain.Quote(home) +
		" && printf '%s\\n' " + domain.Quote(key) + " >> " + domain.Quote(home+"/authorized_keys") +
		" && chown " + domain.Quote(name+":"+name) + " " + domain.Quote(home+"/authorized_keys") +
		" && chmod 600 " + domain.Quote(home+"/authorized_keys")

	cmd := &domain.Command{Path: binSh, Args: []string{"-c", guion}}

	if _, err := session.Run(ctx, cmd); err != nil {
		return fmt.Errorf("instalar las llaves del usuario %q: %w", name, err)
	}

	return nil
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
