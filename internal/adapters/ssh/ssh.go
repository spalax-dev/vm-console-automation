// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package ssh adapta el cliente OpenSSH del sistema operativo al puerto
// Connector de la consola. Ejecuta órdenes remotas con «ssh» y transfiere
// archivos con «scp», ambos resueltos por LookPath («ssh.exe» y «scp.exe»
// en Windows).
package ssh

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// binarySSH y binarySCP son los programas del cliente OpenSSH.
const (
	binarySSH  = "ssh"
	binarySCP  = "scp"
	optionHost = "StrictHostKeyChecking=accept-new"
)

// defaultUser y defaultPort son el usuario y el puerto por omisión.
const (
	defaultUser = "root"
	defaultPort = 22
)

// Connector implementa ports.Connector usando el cliente OpenSSH.
type Connector struct {
	exe          ports.Executor
	keyPath      string
	user         string
	port         uint16
	passwordFile string
}

// Option configura un Connector.
type Option func(*Connector)

// WithKey establece la ruta de la llave privada de autenticación.
func WithKey(path string) Option {
	return func(c *Connector) {
		c.keyPath = path
	}
}

// WithUser establece el usuario remoto con el que se abre la sesión.
func WithUser(user string) Option {
	return func(c *Connector) {
		c.user = user
	}
}

// WithPort establece el puerto del servidor SSH remoto.
func WithPort(port uint16) Option {
	return func(c *Connector) {
		c.port = port
	}
}

// New crea un conector SSH que usa el cliente OpenSSH instalado en el
// sistema operativo anfitrión.
func New(exe ports.Executor, opts ...Option) *Connector {
	connector := &Connector{
		exe:  exe,
		user: defaultUser,
		port: defaultPort,
	}

	for _, opt := range opts {
		opt(connector)
	}

	return connector
}

// Name implements ports.Connector.
func (c *Connector) Name() string {
	return "ssh"
}

// Probe implements ports.Connector: exige conocer la dirección de la
// máquina, que la llave privada exista y que la autenticación funcione.
// La prueba real evita elegir SSH cuando la llave pertenece a otra VM o
// la máquina está en NAT y no es alcanzable.
func (c *Connector) Probe(ctx context.Context, target domain.Target) error {
	if !target.Address.IsValid() {
		return fmt.Errorf("ssh: dirección desconocida de %q; descúbrela antes de conectar", target.ID)
	}

	if _, err := os.Stat(c.keyPath); err != nil {
		return fmt.Errorf("ssh: la llave privada %q no está disponible: %w", c.keyPath, err)
	}

	args := c.baseArgs()
	args = append(args, c.user+"@"+target.Address.String(), "true")

	if _, err := c.exe.Execute(ctx, binarySSH, args...); err != nil {
		return fmt.Errorf("ssh: no se pudo autenticar con %q: %w", target.ID, err)
	}

	return nil
}

// Connect implements ports.Connector: devuelve una sesión que ejecuta
// órdenes remotas con el cliente OpenSSH.
func (c *Connector) Connect(ctx context.Context, target domain.Target) (ports.Session, error) {
	if err := c.Probe(ctx, target); err != nil {
		return nil, err
	}

	return &Session{connector: c, target: target}, nil
}

// tempFile es el subconjunto de *os.File que usa la contraseña
// temporal; permite simular fallos en las pruebas.
type tempFile interface {
	Chmod(mode os.FileMode) error
	WriteString(text string) (int, error)
	Name() string
	Close() error
}

// createTempFile abre el archivo temporal de la contraseña.
var createTempFile = func() (*os.File, error) {
	return os.CreateTemp("", "vmcons-pass-*")
}

// ConnectPassword implementa ports.PasswordConnector usando sshpass. La
// contraseña se guarda en un archivo temporal 0600 que la sesión elimina
// al cerrarse.
func (c *Connector) ConnectPassword(
	_ context.Context, target domain.Target, user, password string,
) (ports.Session, error) {
	file, err := createTempFile()
	if err != nil {
		return nil, fmt.Errorf("ssh: crear el archivo temporal: %w", err)
	}

	if err := prepareTempFile(file, password); err != nil {
		return nil, err
	}

	duplicate := *c
	duplicate.user = user
	duplicate.passwordFile = file.Name()

	return &Session{connector: &duplicate, target: target, passwordFile: file.Name()}, nil
}

// prepareTempFile protege el archivo, escribe la contraseña y lo cierra.
// Ante un fallo deja el sistema sin archivos temporales abandonados.
func prepareTempFile(file tempFile, password string) error {
	if err := file.Chmod(0o600); err != nil {
		closeTempFile(file)

		return fmt.Errorf("ssh: proteger el archivo temporal: %w", err)
	}

	if _, err := file.WriteString(password); err != nil {
		closeTempFile(file)

		return fmt.Errorf("ssh: guardar la contraseña temporal: %w", err)
	}

	if err := file.Close(); err != nil {
		removeTempFile(file.Name())

		return fmt.Errorf("ssh: cerrar el archivo temporal: %w", err)
	}

	return nil
}

// Session implements ports.Session usando los clientes ssh y scp.
type Session struct {
	connector    *Connector
	target       domain.Target
	passwordFile string
}

// Run implements ports.Session: ejecuta la orden en la máquina remota,
// entregándola al shell del usuario remoto ya citada con ShellLine.
func (s *Session) Run(ctx context.Context, cmd *domain.Command) (domain.Result, error) {
	programa, args := s.connector.command(binarySSH, s.connector.baseArgs(), s.passwordFile)
	args = append(args, s.destination(), cmd.ShellLine())

	result, err := s.connector.exe.Execute(ctx, programa, args...)
	if err != nil {
		return result, fmt.Errorf("ssh: ejecutar %q: %w", cmd.Path,
			sanitizeError(err, "ssh "+s.destination()))
	}

	return result, nil
}

// Put implements ports.Session: transfiere el archivo con scp y
// establece permisos de lectura, escritura y ejecución en el destino.
// Nota: scp usa -P (mayúscula) para el puerto, a diferencia de ssh.
func (s *Session) Put(ctx context.Context, localPath, remotePath string) error {
	programa, args := s.connector.command(binarySCP, s.connector.scpArgs(), s.passwordFile)
	args = append(args, localPath, s.destination()+":"+remotePath)

	if _, err := s.connector.exe.Execute(ctx, programa, args...); err != nil {
		return fmt.Errorf("ssh: transferir %q: %w", localPath,
			sanitizeError(err, "scp "+s.destination()))
	}

	chmod := &domain.Command{
		Path: "/bin/chmod",
		Args: []string{"755", remotePath},
	}

	if _, err := s.Run(ctx, chmod); err != nil {
		return fmt.Errorf("ssh: permisos de %q: %w", remotePath, err)
	}

	return nil
}

// Close implements ports.Session: elimina el archivo temporal de la
// contraseña, si existe.
func (s *Session) Close() error {
	if s.passwordFile == "" {
		return nil
	}

	if err := os.Remove(s.passwordFile); err != nil {
		return fmt.Errorf("ssh: eliminar el archivo temporal: %w", err)
	}

	return nil
}

// sanitizeError reemplaza la orden completa por una versión sin
// argumentos sensibles (contraseña, script remoto o rutas temporales).
func sanitizeError(err error, command string) error {
	var exitErr *domain.ExitError
	if errors.As(err, &exitErr) {
		return &domain.ExitError{Code: exitErr.Code, Command: command}
	}

	return err
}

// destination devuelve el destino remoto «usuario@dirección».
func (s *Session) destination() string {
	return s.connector.user + "@" + s.target.Address.String()
}

// closeTempFile cierra y elimina el archivo temporal de la contraseña.
func closeTempFile(file tempFile) {
	if err := file.Close(); err != nil {
		slog.Warn("cerrar el archivo temporal de la contraseña", "error", err)
	}

	removeTempFile(file.Name())
}

// removeTempFile borra el archivo temporal sin interrumpir el flujo.
func removeTempFile(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		slog.Warn("eliminar el archivo temporal de la contraseña", "error", err)
	}
}

// baseArgs construye los argumentos comunes de ssh y scp. En modo llave
// no pide contraseñas; en modo contraseña desactiva la llave y la
// autenticación por teclado para que sshpass pueda responder.
func (c *Connector) baseArgs() []string {
	port := strconv.FormatUint(uint64(c.port), 10)

	if c.passwordFile != "" {
		return []string{
			"-p", port,
			"-o", "BatchMode=no",
			"-o", "PubkeyAuthentication=no",
			"-o", "PreferredAuthentications=password",
			"-o", optionHost,
		}
	}

	return []string{
		"-i", c.keyPath,
		"-p", port,
		"-o", "BatchMode=yes",
		"-o", optionHost,
	}
}

// command antepone sshpass cuando la sesión usa contraseña.
func (c *Connector) command(programa string, args []string, passwordFile string) (name string, arguments []string) {
	if passwordFile == "" {
		return programa, args
	}

	return "sshpass", append([]string{"-f", passwordFile, programa}, args...)
}

// scpArgs construye los argumentos comunes de scp. Nota: scp usa -P
// (mayúscula) para el puerto, a diferencia de ssh.
func (c *Connector) scpArgs() []string {
	return []string{
		"-i", c.keyPath,
		"-P", strconv.FormatUint(uint64(c.port), 10),
		"-o", "BatchMode=yes",
		"-o", optionHost,
	}
}
