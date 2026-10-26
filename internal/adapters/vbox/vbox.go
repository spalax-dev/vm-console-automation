// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package vbox adapta Oracle VM VirtualBox a los puertos Machine y
// Connector de la consola, hablando exclusivamente con el programa
// VBoxManage. Usa Guest Additions para descubrir la dirección de la
// máquina y para ejecutar órdenes dentro de ella.
package vbox

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// guestIPv4Property es la propiedad de Guest Additions que publica la
// dirección IPv4 de la primera interfaz de red de la máquina invitada.
const guestIPv4Property = "/VirtualBox/GuestInfo/Net/0/V4/IP"

// defaultBinary es el nombre del programa VBoxManage; en Windows,
// exec.LookPath resuelve VBoxManage.exe sin necesidad de la extensión.
const defaultBinary = "VBoxManage"

// flagMachineReadable pide la salida legible por máquina de showvminfo.
const flagMachineReadable = "--machinereadable"

// cmdShowVMInfo es el subcomando que describe una máquina.
const cmdShowVMInfo = "showvminfo"

// cmdGuestProperty es el subcomando que consulta propiedades.
const cmdGuestProperty = "guestproperty"

// cmdList es el subcomando que enumera máquinas.
const cmdList = "list"

// msgErrorCommand es el contexto de error de VBoxManage.
const msgErrorCommand = "%s: %w"

// Client implementa ports.Machine y ports.Connector sobre VirtualBox.
type Client struct {
	exe         ports.Executor
	bin         string
	user        string
	pass        string
	credentials Credentials

	// mu serializa las llamadas a VBoxManage: el programa usa un bloqueo
	// global y las peticiones concurrentes fallan mientras otra lo tiene.
	mu sync.Mutex
}

// Credentials resuelve el usuario y la contraseña de Guest Additions
// para una máquina concreta.
type Credentials interface {
	Guest(vm string) (user, password string, ok bool)
}

// Option configura un Client.
type Option func(*Client)

// WithBinary establece la ruta o nombre del programa VBoxManage.
func WithBinary(bin string) Option {
	return func(c *Client) {
		c.bin = bin
	}
}

// WithCredentials establece el almacén que resuelve las credenciales
// por máquina; tiene prioridad sobre las credenciales por omisión.
func WithCredentials(credentials Credentials) Option {
	return func(c *Client) {
		c.credentials = credentials
	}
}

// WithGuestCredentials establece las credenciales que Guest Additions
// exige para abrir sesiones dentro de la máquina invitada.
func WithGuestCredentials(user, password string) Option {
	return func(c *Client) {
		c.user = user
		c.pass = password
	}
}

// New crea un cliente de VirtualBox que ejecuta VBoxManage mediante exe.
func New(exe ports.Executor, opts ...Option) *Client {
	client := &Client{
		exe:  exe,
		bin:  defaultBinary,
		user: "root",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// run lanza VBoxManage en serie y reintenta una vez las consultas
// de solo lectura cuando VirtualBox está ocupado.
func (c *Client) run(ctx context.Context, args ...string) (domain.Result, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	command := "VBoxManage"
	if len(args) > 0 {
		command += " " + args[0]
	}

	result, err := c.exe.Execute(ctx, c.bin, args...)
	if err == nil {
		return result, nil
	}

	if ctx.Err() != nil || !reintentable(args) {
		return result, fmt.Errorf(msgErrorCommand, command, err)
	}

	select {
	case <-ctx.Done():
		return result, fmt.Errorf(msgErrorCommand, command, err)
	case <-time.After(250 * time.Millisecond):
	}

	result, err = c.exe.Execute(ctx, c.bin, args...)
	if err != nil {
		return result, fmt.Errorf(msgErrorCommand, command, err)
	}

	return result, nil
}

// reintentable limita los reintentos a las consultas que no modifican
// el estado de la máquina.
func reintentable(args []string) bool {
	if len(args) == 0 {
		return false
	}

	switch args[0] {
	case "showvminfo", cmdList, cmdGuestProperty, "--version":
		return true
	default:
		return false
	}
}

// Name implements ports.Connector.
func (c *Client) Name() string {
	return "guest-additions"
}

// List implements ports.Inventory: enumera las máquinas registradas en
// VirtualBox y marca como en ejecución las que aparecen en
// «list runningvms». Consultar el estado exacto de cada máquina costaría
// una ejecución de VBoxManage por máquina, así que las que no están en
// ejecución se informan como apagadas.
func (c *Client) List(ctx context.Context) ([]domain.VM, error) {
	todas, err := c.run(ctx, cmdList, "vms")
	if err != nil {
		return nil, fmt.Errorf("listar las máquinas virtuales: %w", err)
	}

	inRunning, err := c.run(ctx, cmdList, "runningvms")
	if err != nil {
		return nil, fmt.Errorf("listar las máquinas en ejecución: %w", err)
	}

	running := make(map[string]struct{})

	for _, machine := range parseList(inRunning.Stdout) {
		running[machine.Name] = struct{}{}
	}

	machines := parseList(todas.Stdout)

	for i := range machines {
		if _, active := running[machines[i].Name]; active {
			machines[i].State = domain.StateRunning
		} else {
			machines[i].State = domain.StatePoweredOff
		}
	}

	return machines, nil
}

// Start implements ports.Power: arranca la máquina en modo sin interfaz
// gráfica, que es el que corresponde a una consola de administración.
func (c *Client) Start(ctx context.Context, target domain.Target) error {
	if _, err := c.run(ctx, "startvm", target.ID, "--type", "headless"); err != nil {
		return fmt.Errorf("encender la máquina %q: %w", target.ID, err)
	}

	return nil
}

// listPattern reconoce una línea de «VBoxManage list vms» o «list
// runningvms»: el nombre entre comillas y el UUID entre llaves.
var listPattern = regexp.MustCompile(`^"([^"]*)" \{([^}]*)\}$`)

// parseList extrae las máquinas de la salida de VBoxManage, ignorando
// las líneas que no siguen el formato.
func parseList(output string) []domain.VM {
	machines := make([]domain.VM, 0, strings.Count(output, "\n"))

	for _, line := range strings.Split(output, "\n") {
		coincidencia := listPattern.FindStringSubmatch(strings.TrimSpace(line))
		if coincidencia == nil {
			continue
		}

		machines = append(machines, domain.VM{
			Name: coincidencia[1],
			UUID: coincidencia[2],
		})
	}

	return machines
}

// Address implements ports.Machine: lee la propiedad que publica Guest
// Additions y devuelve la dirección de la máquina.
func (c *Client) Address(ctx context.Context, target domain.Target) (netip.Addr, error) {
	result, err := c.run(ctx, cmdGuestProperty, "get", target.ID, guestIPv4Property)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("guest-additions: leer la dirección de %q: %w", target.ID, err)
	}

	addr, err := parseAddress(result.Stdout)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("guest-additions: leer la dirección de %q: %w", target.ID, err)
	}

	return addr, nil
}

// State implements ports.Machine: consulta el estado de encendido de la
// máquina con salida legible por máquina.
func (c *Client) State(ctx context.Context, target domain.Target) (domain.MachineState, error) {
	result, err := c.run(ctx, cmdShowVMInfo, target.ID, flagMachineReadable)
	if err != nil {
		return domain.StateUnknown, fmt.Errorf("guest-additions: leer el estado de %q: %w", target.ID, err)
	}

	return parseState(result.Stdout), nil
}

// Network implements ports.Machine: interpreta el modo de la primera
// interfaz de red de «showvminfo --machinereadable».
func (c *Client) Network(ctx context.Context, target domain.Target) (domain.NetworkMode, error) {
	result, err := c.run(ctx, cmdShowVMInfo, target.ID, flagMachineReadable)
	if err != nil {
		return domain.NetworkUnknown, fmt.Errorf("leer la red de %q: %w", target.ID, err)
	}

	return parseNetwork(result.Stdout), nil
}

// Version devuelve la versión de VirtualBox del anfitrión.
func (c *Client) Version(ctx context.Context) (string, error) {
	result, err := c.run(ctx, "--version")
	if err != nil {
		return "", fmt.Errorf("leer la versión de VirtualBox: %w", err)
	}

	return strings.TrimSpace(result.Stdout), nil
}

// GuestAdditions implements ports.Machine: lee la versión y el nivel de
// Guest Additions y los usuarios con sesión iniciada.
func (c *Client) GuestAdditions(ctx context.Context, target domain.Target) (domain.GuestAdditions, error) {
	result, err := c.run(ctx, cmdShowVMInfo, target.ID, flagMachineReadable)
	if err != nil {
		return domain.GuestAdditions{}, fmt.Errorf("leer Guest Additions de %q: %w", target.ID, err)
	}

	guest := domain.GuestAdditions{
		Version:  valueProperty(result.Stdout, "GuestAdditionsVersion"),
		RunLevel: enteroProperty(result.Stdout, "GuestAdditionsRunLevel"),
	}

	users, err := c.run(ctx, cmdGuestProperty, "get", target.ID,
		"/VirtualBox/GuestInfo/OS/LoggedInUsersList")
	if err == nil {
		guest.LoggedUsers = parseUsers(users.Stdout)
	}

	return guest, nil
}

// valueProperty extrae «clave="valor"» de showvminfo.
func valueProperty(output, key string) string {
	prefijo := key + `="`

	for _, line := range strings.Split(output, "\n") {
		value, found := strings.CutPrefix(strings.TrimSpace(line), prefijo)
		if found {
			return strings.TrimSuffix(value, `"`)
		}
	}

	return ""
}

// enteroProperty extrae «clave=valor» de showvminfo.
func enteroProperty(output, key string) int {
	prefijo := key + "="

	for _, line := range strings.Split(output, "\n") {
		value, found := strings.CutPrefix(strings.TrimSpace(line), prefijo)
		if !found {
			continue
		}

		number, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return number
		}
	}

	return 0
}

// parseUsers interpreta la propiedad de usuarios conectados.
func parseUsers(output string) []string {
	const prefijo = "Value:"

	for _, line := range strings.Split(output, "\n") {
		value, found := strings.CutPrefix(strings.TrimSpace(line), prefijo)
		if !found {
			continue
		}

		users := []string{}

		for _, user := range strings.Split(strings.TrimSpace(value), ",") {
			if user != "" {
				users = append(users, user)
			}
		}

		return users
	}

	return nil
}

// Probe implements ports.Connector: la máquina debe estar en ejecución
// para que Guest Additions pueda atenderla.
func (c *Client) Probe(ctx context.Context, target domain.Target) error {
	state, err := c.State(ctx, target)
	if err != nil {
		return err
	}

	if state != domain.StateRunning {
		return fmt.Errorf("guest-additions: la máquina %q no está en ejecución (estado %q)", target.ID, state)
	}

	return nil
}

// Connect implements ports.Connector: devuelve una sesión que ejecuta
// órdenes en la máquina mediante «vboxmanage guestcontrol».
func (c *Client) Connect(ctx context.Context, target domain.Target) (ports.Session, error) {
	if err := c.Probe(ctx, target); err != nil {
		return nil, err
	}

	user, password := c.user, c.pass

	if c.credentials != nil {
		if u, p, ok := c.credentials.Guest(target.ID); ok {
			user, password = u, p
		}
	}

	return &Session{client: c, target: target, user: user, pass: password}, nil
}

// Session implements ports.Session usando «vboxmanage guestcontrol».
type Session struct {
	client *Client
	target domain.Target
	user   string
	pass   string
}

// Run implements ports.Session: ejecuta la orden dentro de la máquina
// invitada pasando cada componente como argumento literal, sin shell.
func (s *Session) Run(ctx context.Context, cmd *domain.Command) (domain.Result, error) {
	args := []string{"guestcontrol", s.target.ID, "run", "--username", s.user, "--password", s.pass}

	for _, entorno := range cmd.Env {
		args = append(args, "--putenv", entorno)
	}

	if cmd.Dir != "" {
		args = append(args, "--cwd", cmd.Dir)
	}

	args = append(args, "--exe", cmd.Path, "--")
	args = append(args, cmd.Args...)

	result, err := s.client.run(ctx, args...)
	if err != nil {
		return result, fmt.Errorf("guest-additions: ejecutar %q: %w", cmd.Path,
			sanitizeError(err, "VBoxManage guestcontrol "+s.target.ID+" run"))
	}

	return result, nil
}

// Put implements ports.Session: transfiere un archivo de la consola a
// la máquina invitada con «vboxmanage guestcontrol copyto».
func (s *Session) Put(ctx context.Context, localPath, remotePath string) error {
	args := []string{
		"guestcontrol", s.target.ID, "copyto",
		"--username", s.user, "--password", s.pass,
		localPath, remotePath,
	}

	if _, err := s.client.run(ctx, args...); err != nil {
		return fmt.Errorf("guest-additions: transferir %q: %w", localPath,
			sanitizeError(err, "VBoxManage guestcontrol "+s.target.ID+" copyto"))
	}

	return nil
}

// Close implements ports.Session: «guestcontrol» no mantiene estado
// entre órdenes, por lo que no hay recursos que liberar.
func (s *Session) Close() error {
	return nil
}

// sanitizeError reemplaza la orden completa por una versión sin
// argumentos sensibles (por ejemplo, la contraseña de Guest Additions).
func sanitizeError(err error, command string) error {
	var exitErr *domain.ExitError
	if errors.As(err, &exitErr) {
		return &domain.ExitError{Code: exitErr.Code, Command: command}
	}

	return err
}

// parseAddress extrae la dirección de la salida de «guestproperty get»,
// cuya última línea tiene la forma «Value: 10.0.2.15».
func parseAddress(output string) (netip.Addr, error) {
	const prefix = "Value:"

	for _, line := range strings.Split(output, "\n") {
		value, found := strings.CutPrefix(strings.TrimSpace(line), prefix)
		if !found {
			continue
		}

		addr, err := netip.ParseAddr(strings.TrimSpace(value))
		if err != nil {
			return netip.Addr{}, fmt.Errorf("el valor %q no es una dirección válida: %w", strings.TrimSpace(value), err)
		}

		return addr, nil
	}

	return netip.Addr{}, fmt.Errorf("la salida no contiene la propiedad %q", guestIPv4Property)
}

// parseState extrae el estado de la salida de «showvminfo
// --machinereadable», cuya línea relevante es «VMState="running"».
func parseState(output string) domain.MachineState {
	const prefix = `VMState="`

	for _, line := range strings.Split(output, "\n") {
		value, found := strings.CutPrefix(strings.TrimSpace(line), prefix)
		if !found {
			continue
		}

		return mapState(strings.TrimSuffix(value, `"`))
	}

	return domain.StateUnknown
}

// parseNetwork extrae el modo de la primera interfaz de la salida de
// «showvminfo --machinereadable», cuya línea relevante es «nic1="nat"».
func parseNetwork(output string) domain.NetworkMode {
	const prefix = `nic1="`

	for _, line := range strings.Split(output, "\n") {
		value, found := strings.CutPrefix(strings.TrimSpace(line), prefix)
		if !found {
			continue
		}

		return mapNetwork(strings.TrimSuffix(value, `"`))
	}

	return domain.NetworkUnknown
}

// mapNetwork traduce el modo de VirtualBox al lenguaje del dominio.
func mapNetwork(raw string) domain.NetworkMode {
	switch raw {
	case "bridged":
		return domain.NetworkBridged
	case "nat", "natnetwork":
		return domain.NetworkNAT
	case "hostonly":
		return domain.NetworkHostOnly
	case "intnet":
		return domain.NetworkInternal
	case "none", "null":
		return domain.NetworkNone
	default:
		return domain.NetworkUnknown
	}
}

// mapState traduce el vocabulario de VirtualBox al lenguaje del dominio.
func mapState(raw string) domain.MachineState {
	switch raw {
	case "running":
		return domain.StateRunning
	case "poweroff", "aborted":
		return domain.StatePoweredOff
	case "paused":
		return domain.StatePaused
	case "saved":
		return domain.StateSaved
	default:
		return domain.StateUnknown
	}
}
