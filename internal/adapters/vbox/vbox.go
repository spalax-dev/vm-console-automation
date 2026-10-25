// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package vbox adapta Oracle VM VirtualBox a los puertos Machine y
// Connector de la consola, hablando exclusivamente con el programa
// VBoxManage. Usa Guest Additions para descubrir la dirección de la
// máquina y para ejecutar órdenes dentro de ella.
package vbox

import (
	"context"
	"fmt"
	"net/netip"
	"regexp"
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// guestIPv4Property es la propiedad de Guest Additions que publica la
// dirección IPv4 de la primera interfaz de red de la máquina invitada.
const guestIPv4Property = "/VirtualBox/GuestInfo/Net/0/V4/IP"

// defaultBinary es el nombre del programa VBoxManage; en Windows,
// exec.LookPath resuelve VBoxManage.exe sin necesidad de la extensión.
const defaultBinary = "VBoxManage"

// Client implementa ports.Machine y ports.Connector sobre VirtualBox.
type Client struct {
	exe  ports.Executor
	bin  string
	user string
	pass string
}

// Option configura un Client.
type Option func(*Client)

// WithBinary establece la ruta o nombre del programa VBoxManage.
func WithBinary(bin string) Option {
	return func(c *Client) {
		c.bin = bin
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
	todas, err := c.exe.Execute(ctx, c.bin, "list", "vms")
	if err != nil {
		return nil, fmt.Errorf("listar las máquinas virtuales: %w", err)
	}

	enMarcha, err := c.exe.Execute(ctx, c.bin, "list", "runningvms")
	if err != nil {
		return nil, fmt.Errorf("listar las máquinas en ejecución: %w", err)
	}

	running := make(map[string]struct{})

	for _, maquina := range parseList(enMarcha.Stdout) {
		running[maquina.Name] = struct{}{}
	}

	maquinas := parseList(todas.Stdout)

	for i := range maquinas {
		if _, activa := running[maquinas[i].Name]; activa {
			maquinas[i].State = domain.StateRunning
		} else {
			maquinas[i].State = domain.StatePoweredOff
		}
	}

	return maquinas, nil
}

// Start implements ports.Power: arranca la máquina en modo sin interfaz
// gráfica, que es el que corresponde a una consola de administración.
func (c *Client) Start(ctx context.Context, target domain.Target) error {
	if _, err := c.exe.Execute(ctx, c.bin, "startvm", target.ID, "--type", "headless"); err != nil {
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
	maquinas := make([]domain.VM, 0, strings.Count(output, "\n"))

	for _, line := range strings.Split(output, "\n") {
		coincidencia := listPattern.FindStringSubmatch(strings.TrimSpace(line))
		if coincidencia == nil {
			continue
		}

		maquinas = append(maquinas, domain.VM{
			Name: coincidencia[1],
			UUID: coincidencia[2],
		})
	}

	return maquinas
}

// Address implements ports.Machine: lee la propiedad que publica Guest
// Additions y devuelve la dirección de la máquina.
func (c *Client) Address(ctx context.Context, target domain.Target) (netip.Addr, error) {
	result, err := c.exe.Execute(ctx, c.bin, "guestproperty", "get", target.ID, guestIPv4Property)
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
	result, err := c.exe.Execute(ctx, c.bin, "showvminfo", target.ID, "--machinereadable")
	if err != nil {
		return domain.StateUnknown, fmt.Errorf("guest-additions: leer el estado de %q: %w", target.ID, err)
	}

	return parseState(result.Stdout), nil
}

// Network implements ports.Machine: interpreta el modo de la primera
// interfaz de red de «showvminfo --machinereadable».
func (c *Client) Network(ctx context.Context, target domain.Target) (domain.NetworkMode, error) {
	result, err := c.exe.Execute(ctx, c.bin, "showvminfo", target.ID, "--machinereadable")
	if err != nil {
		return domain.NetworkUnknown, fmt.Errorf("leer la red de %q: %w", target.ID, err)
	}

	return parseNetwork(result.Stdout), nil
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

	return &Session{client: c, target: target}, nil
}

// Session implements ports.Session usando «vboxmanage guestcontrol».
type Session struct {
	client *Client
	target domain.Target
}

// Run implements ports.Session: ejecuta la orden dentro de la máquina
// invitada pasando cada componente como argumento literal, sin shell.
func (s *Session) Run(ctx context.Context, cmd *domain.Command) (domain.Result, error) {
	args := []string{"guestcontrol", s.target.ID, "run", "--username", s.client.user, "--password", s.client.pass}

	for _, entorno := range cmd.Env {
		args = append(args, "--putenv", entorno)
	}

	if cmd.Dir != "" {
		args = append(args, "--cwd", cmd.Dir)
	}

	args = append(args, "--exe", cmd.Path, "--", cmd.Path)
	args = append(args, cmd.Args...)

	result, err := s.client.exe.Execute(ctx, s.client.bin, args...)
	if err != nil {
		return result, fmt.Errorf("guest-additions: ejecutar %q: %w", cmd.Path, err)
	}

	return result, nil
}

// Put implements ports.Session: transfiere un archivo de la consola a
// la máquina invitada con «vboxmanage guestcontrol copyto».
func (s *Session) Put(ctx context.Context, localPath, remotePath string) error {
	args := []string{
		"guestcontrol", s.target.ID, "copyto",
		"--username", s.client.user, "--password", s.client.pass,
		localPath, remotePath,
	}

	if _, err := s.client.exe.Execute(ctx, s.client.bin, args...); err != nil {
		return fmt.Errorf("guest-additions: transferir %q: %w", localPath, err)
	}

	return nil
}

// Close implements ports.Session: «guestcontrol» no mantiene estado
// entre órdenes, por lo que no hay recursos que liberar.
func (s *Session) Close() error {
	return nil
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
