// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// haproxyConfigPath es la configuración de HAProxy en la máquina.
const haproxyConfigPath = "/etc/haproxy/haproxy.cfg"

// haproxySocketPath es el socket de estadísticas del balanceador; la
// configuración por omisión de Debian y sus derivados suele definirlo en el
// bloque global.
const haproxySocketPath = "/run/haproxy/admin.sock"

// binSocat consulta el socket de estadísticas dentro de la máquina.
const binSocat = "/usr/bin/socat"

// showStatCommand pide la tabla de estadísticas por el socket.
const showStatCommand = `printf 'show stat\n' | ` + binSocat + ` - UNIX-CONNECT:` + haproxySocketPath

// msgReadConfig es el contexto de error al leer la configuración.
const msgReadConfig = "leer la configuración de HAProxy: %w"

// ErrBackendNotFound indica que la configuración no tiene un bloque
// backend.
var ErrBackendNotFound = errors.New("no se encontró un back-end en la configuración")

// ErrServerExists indica que el servidor ya está en el back-end.
var ErrServerExists = errors.New("el servidor ya está en el back-end")

// ErrServerNotFound indica que el servidor no está en el back-end.
var ErrServerNotFound = errors.New("el servidor no está en el back-end")

// Server describe un servidor del back-end de HAProxy.
type Server struct {
	// Address es la dirección IPv4 del servidor.
	Address string
	// Port es el puerto TCP.
	Port int
	// State es el estado informado por HAProxy: up, down o unknown
	// cuando el socket de estadísticas no responde.
	State string
	// name es el identificador del servidor en la configuración, que
	// usa la tabla de estadísticas para informar su estado.
	name string
}

// HAProxy implementa el caso de uso del back-end de HAProxy.
type HAProxy struct {
	sessions SessionOpener
}

// NewHAProxy construye el servicio con el abridor de sesiones dado.
func NewHAProxy(sessions SessionOpener) *HAProxy {
	return &HAProxy{sessions: sessions}
}

// List devuelve los servidores del primer bloque backend.
func (h *HAProxy) List(ctx context.Context) ([]Server, error) {
	session, err := h.sessions.Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrir la sesión para listar el back-end: %w", err)
	}

	defer closeSession(session)

	config, err := runOutput(ctx, session, binCat, haproxyConfigPath)
	if err != nil {
		return nil, fmt.Errorf(msgReadConfig, err)
	}

	servers, found := parseServers(config)
	if !found {
		return nil, ErrBackendNotFound
	}

	// El estado de cada servidor se toma del socket de estadísticas. Si la
	// consulta falla, el listado sigue siendo válido con estado unknown.
	if len(servers) > 0 {
		states := backendStates(ctx, session)

		for i := range servers {
			if state, ok := states[servers[i].name]; ok {
				servers[i].State = state
			}
		}
	}

	return servers, nil
}

// backendStates consulta el socket de estadísticas y devuelve el estado
// de cada servidor por su nombre en la configuración. Un error de la
// consulta no interrumpe el listado: se devuelve un mapa vacío y los
// servidores conservan el estado unknown.
func backendStates(ctx context.Context, session ports.Session) map[string]string {
	cmd := &domain.Command{Path: binSh, Args: []string{"-c", showStatCommand}}

	result, err := session.Run(ctx, cmd)
	if err != nil {
		return map[string]string{}
	}

	return parseStates(result.Stdout)
}

// parseStates traduce el campo status de la salida CSV del socket de
// estadísticas a up, down o unknown. Las filas incompletas o con estados
// que la consola no distingue se ignoran.
func parseStates(output string) map[string]string {
	states := make(map[string]string)

	for _, line := range strings.Split(output, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 18 {
			continue
		}

		if state, ok := haproxyState(fields[17]); ok {
			states[fields[1]] = state
		}
	}

	return states
}

// haproxyState traduce el estado de HAProxy al vocabulario del contrato.
// Los estados de mantenimiento y de drenaje se informan como down porque
// el servidor no recibe tráfico nuevo.
func haproxyState(state string) (string, bool) {
	switch state {
	case "UP":
		return "up", true
	case "DOWN", "MAINT", "DRAIN", "NOLB":
		return "down", true
	default:
		return "", false
	}
}

// Add agrega un servidor al back-end y recarga HAProxy.
func (h *HAProxy) Add(ctx context.Context, address string, port int) (Outcome, error) {
	session, err := h.sessions.Open(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("abrir la sesión para agregar el servidor: %w", err)
	}

	defer closeSession(session)

	config, err := runOutput(ctx, session, binCat, haproxyConfigPath)
	if err != nil {
		return Outcome{}, fmt.Errorf(msgReadConfig, err)
	}

	updated, err := addServer(config, address, port)
	if err != nil {
		return Outcome{}, err
	}

	return h.write(ctx, session, updated, "Servidor agregado al back-end.")
}

// Delete elimina un servidor del back-end y recarga HAProxy.
func (h *HAProxy) Delete(ctx context.Context, address string, port int) (Outcome, error) {
	session, err := h.sessions.Open(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("abrir la sesión para eliminar el servidor: %w", err)
	}

	defer closeSession(session)

	config, err := runOutput(ctx, session, binCat, haproxyConfigPath)
	if err != nil {
		return Outcome{}, fmt.Errorf(msgReadConfig, err)
	}

	updated, err := removeServer(config, address, port)
	if err != nil {
		return Outcome{}, err
	}

	return h.write(ctx, session, updated, "Servidor eliminado del back-end.")
}

// write valida la configuración propuesta y, solo si es correcta, la
// instala y recarga HAProxy.
func (h *HAProxy) write(
	ctx context.Context, session ports.Session, config, message string,
) (Outcome, error) {
	// HAProxy rechaza la configuración si la última línea no termina en
	// salto de línea; al agregar un servidor al último backend la
	// concatenación puede perderlo.
	if !strings.HasSuffix(config, "\n") {
		config += "\n"
	}

	b64 := base64.StdEncoding.EncodeToString([]byte(config))
	guion := "printf '%s' " + domain.Quote(b64) + " | base64 -d > /tmp/haproxy.cfg.vmcons && " +
		binHaproxy + " -c -f /tmp/haproxy.cfg.vmcons && " +
		"mv /tmp/haproxy.cfg.vmcons " + haproxyConfigPath + " && " +
		binSystemctl + " reload haproxy"

	cmd := &domain.Command{Path: binSh, Args: []string{"-c", guion}}

	result, err := session.Run(ctx, cmd)

	output := Outcome{
		Message:   message,
		Operation: cmd.ShellLine(),
		Output:    result.Stdout,
		Stderr:    result.Stderr,
		Code:      result.ExitCode,
	}

	if err != nil {
		return output, fmt.Errorf("validar y recargar HAProxy: %w", err)
	}

	return output, nil
}

// parseServers extrae los servidores del primer bloque backend.
func parseServers(config string) (servers []Server, found bool) {
	servers = []Server{}

	start, end := backendLimits(strings.Split(config, "\n"))
	if start < 0 {
		return servers, false
	}

	for _, line := range strings.Split(config, "\n")[start+1 : end] {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != "server" {
			continue
		}

		destination, err := netip.ParseAddrPort(fields[2])
		if err != nil {
			continue
		}

		servers = append(servers, Server{
			Address: destination.Addr().String(),
			Port:    int(destination.Port()),
			State:   "unknown",
			name:    fields[1],
		})
	}

	return servers, true
}

// addServer inserta la línea del servidor al final del primer
// bloque backend.
func addServer(config, address string, port int) (string, error) {
	lines := strings.Split(config, "\n")

	start, end := backendLimits(lines)
	if start < 0 {
		return "", ErrBackendNotFound
	}

	for _, line := range lines[start+1 : end] {
		if serverMatches(line, address, port) {
			return "", ErrServerExists
		}
	}

	name := "srv_" + strings.NewReplacer(".", "_", ":", "_").Replace(address) + "_" + strconv.Itoa(port)
	serverLine := "    server " + name + " " + address + ":" + strconv.Itoa(port) + " check"

	updated := make([]string, 0, len(lines)+1)
	updated = append(updated, lines[:end]...)
	updated = append(updated, serverLine)
	updated = append(updated, lines[end:]...)

	return strings.Join(updated, "\n"), nil
}

// removeServer elimina la línea del servidor del primer bloque
// backend.
func removeServer(config, address string, port int) (string, error) {
	lines := strings.Split(config, "\n")

	start, end := backendLimits(lines)
	if start < 0 {
		return "", ErrBackendNotFound
	}

	updated := make([]string, 0, len(lines))
	updated = append(updated, lines[:start+1]...)
	found := false

	for _, line := range lines[start+1 : end] {
		if serverMatches(line, address, port) {
			found = true

			continue
		}

		updated = append(updated, line)
	}

	if !found {
		return "", ErrServerNotFound
	}

	updated = append(updated, lines[end:]...)

	return strings.Join(updated, "\n"), nil
}

// serverMatches informa si la línea es un servidor con el destino
// indicado.
func serverMatches(line, address string, port int) bool {
	fields := strings.Fields(line)
	if len(fields) < 3 || fields[0] != "server" {
		return false
	}

	destination, err := netip.ParseAddrPort(fields[2])
	if err != nil {
		return false
	}

	return destination.Addr().String() == address && int(destination.Port()) == port
}

// backendLimits devuelve el inicio y el fin del primer bloque backend.
func backendLimits(lines []string) (start, end int) {
	start = -1

	for i, line := range lines {
		limpia := strings.TrimSpace(line)
		if limpia == "" || strings.HasPrefix(limpia, "#") {
			continue
		}

		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}

		fields := strings.Fields(limpia)
		if fields[0] == "backend" {
			if start == -1 {
				start = i
			} else {
				return start, i
			}

			continue
		}

		if start != -1 {
			return start, i
		}
	}

	if start != -1 {
		return start, len(lines)
	}

	return -1, -1
}
