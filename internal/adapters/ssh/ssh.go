// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package ssh adapta el cliente OpenSSH del sistema operativo al puerto
// Connector de la consola. Ejecuta órdenes remotas con «ssh» y transfiere
// archivos con «scp», ambos resueltos por LookPath («ssh.exe» y «scp.exe»
// en Windows).
package ssh

import (
	"context"
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
	opcionHost = "StrictHostKeyChecking=accept-new"
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

// ConnectPassword implementa ports.PasswordConnector usando sshpass. La
// contraseña se guarda en un archivo temporal 0600 que la sesión elimina
// al cerrarse.
func (c *Connector) ConnectPassword(
	_ context.Context, target domain.Target, usuario, password string,
) (ports.Session, error) {
	archivo, err := os.CreateTemp("", "vmcons-pass-*")
	if err != nil {
		return nil, fmt.Errorf("ssh: crear el archivo temporal: %w", err)
	}

	if err := archivo.Chmod(0o600); err != nil {
		cerrarTemporal(archivo)

		return nil, fmt.Errorf("ssh: proteger el archivo temporal: %w", err)
	}

	if _, err := archivo.WriteString(password); err != nil {
		cerrarTemporal(archivo)

		return nil, fmt.Errorf("ssh: guardar la contraseña temporal: %w", err)
	}

	if err := archivo.Close(); err != nil {
		eliminarTemporal(archivo.Name())

		return nil, fmt.Errorf("ssh: cerrar el archivo temporal: %w", err)
	}

	copia := *c
	copia.user = usuario
	copia.passwordFile = archivo.Name()

	return &Session{connector: &copia, target: target, passwordFile: archivo.Name()}, nil
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
	programa, args := s.connector.comando(binarySSH, s.connector.baseArgs(), s.passwordFile)
	args = append(args, s.destination(), cmd.ShellLine())

	result, err := s.connector.exe.Execute(ctx, programa, args...)
	if err != nil {
		return result, fmt.Errorf("ssh: ejecutar %q: %w", cmd.Path, err)
	}

	return result, nil
}

// Put implements ports.Session: transfiere el archivo con scp y
// establece permisos de lectura, escritura y ejecución en el destino.
// Nota: scp usa -P (mayúscula) para el puerto, a diferencia de ssh.
func (s *Session) Put(ctx context.Context, localPath, remotePath string) error {
	programa, args := s.connector.comando(binarySCP, s.connector.scpArgs(), s.passwordFile)
	args = append(args, localPath, s.destination()+":"+remotePath)

	if _, err := s.connector.exe.Execute(ctx, programa, args...); err != nil {
		return fmt.Errorf("ssh: transferir %q: %w", localPath, err)
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

// destination devuelve el destino remoto «usuario@dirección».
func (s *Session) destination() string {
	return s.connector.user + "@" + s.target.Address.String()
}

// cerrarTemporal cierra y elimina el archivo temporal de la contraseña.
func cerrarTemporal(archivo *os.File) {
	if err := archivo.Close(); err != nil {
		slog.Warn("cerrar el archivo temporal de la contraseña", "error", err)
	}

	eliminarTemporal(archivo.Name())
}

// eliminarTemporal borra el archivo temporal sin interrumpir el flujo.
func eliminarTemporal(ruta string) {
	if err := os.Remove(ruta); err != nil && !os.IsNotExist(err) {
		slog.Warn("eliminar el archivo temporal de la contraseña", "error", err)
	}
}

// baseArgs construye los argumentos comunes de ssh y scp. En modo llave
// no pide contraseñas; en modo contraseña desactiva la llave y la
// autenticación por teclado para que sshpass pueda responder.
func (c *Connector) baseArgs() []string {
	puerto := strconv.FormatUint(uint64(c.port), 10)

	if c.passwordFile != "" {
		return []string{
			"-p", puerto,
			"-o", "BatchMode=no",
			"-o", "PubkeyAuthentication=no",
			"-o", "PreferredAuthentications=password",
			"-o", opcionHost,
		}
	}

	return []string{
		"-i", c.keyPath,
		"-p", puerto,
		"-o", "BatchMode=yes",
		"-o", opcionHost,
	}
}

// comando antepone sshpass cuando la sesión usa contraseña.
func (c *Connector) comando(programa string, args []string, passwordFile string) (nombre string, argumentos []string) {
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
		"-o", opcionHost,
	}
}
