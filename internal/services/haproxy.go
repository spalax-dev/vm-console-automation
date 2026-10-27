// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"

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

// ServerStats reúne las métricas que publica HAProxy para un servidor.
type ServerStats struct {
	State           string
	Weight          int
	Active          bool
	Backup          bool
	CurrentSessions int
	MaxSessions     int
	TotalSessions   int64
	QueueCurrent    int
	QueueMax        int
	BytesIn         int64
	BytesOut        int64
	CheckStatus     string
	CheckCode       int
	CheckDuration   int
	Downtime        int64
	LastChange      int64
	LastSession     int64
	LastCheck       string
}

// Server describe un servidor del back-end de HAProxy.
type Server struct {
	// Address es la dirección IPv4 del servidor.
	Address string
	// Port es el puerto TCP.
	Port int
	// State es el estado informado por HAProxy: up, down o unknown
	// cuando el socket de estadísticas no responde.
	State string
	// Stats son las métricas del socket de estadísticas.
	Stats ServerStats
	// name es el identificador del servidor en la configuración, que
	// usa la tabla de estadísticas para informar su estado.
	name string
}

// HAProxyStatus describe si HAProxy puede administrarse.
type HAProxyStatus struct {
	// Installed indica si el binario de HAProxy existe.
	Installed bool
	// ConfigExists indica si /etc/haproxy/haproxy.cfg existe.
	ConfigExists bool
	// ConfigValid indica si haproxy -c acepta la configuración.
	ConfigValid bool
	// Backend indica si hay al menos un bloque backend.
	Backend bool
	// ServiceActive indica si el servicio haproxy está activo.
	ServiceActive bool
	// Socat indica si está instalado socat para consultar estados.
	Socat bool
	// Port es el puerto configurado en el frontend, o cero si no se pudo leer.
	Port int
	// Problems describe lo que impide administrar el balanceador.
	Problems []string
	// Suggestions indica cómo resolverlo.
	Suggestions []string
}

// defaultHAProxyConfig es la configuración mínima que instala la
// configuración rápida. Deja un back-end vacío listo para recibir
// servidores desde la consola.
func defaultHAProxyConfig(port int) string {
	return fmt.Sprintf(`global
    log /dev/log local0
    log /dev/log local1 notice
    chroot /var/lib/haproxy
    stats socket /run/haproxy/admin.sock mode 660 level admin
    stats timeout 30s
    user haproxy
    group haproxy
    daemon

defaults
    log global
    mode http
    option httplog
    option dontlognull
    timeout connect 5000
    timeout client 50000
    timeout server 50000

frontend vmcons_front
    bind *:%d
    default_backend vmcons_app

backend vmcons_app
    balance roundrobin
`, port)
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
		stats := backendStats(ctx, session)

		for i := range servers {
			if stat, ok := stats[servers[i].name]; ok {
				servers[i].Stats = stat
				servers[i].State = stat.State
			}
		}
	}

	return servers, nil
}

// backendStats consulta el socket de estadísticas y devuelve las métricas
// de cada servidor por su nombre en la configuración. Un error de la
// consulta no interrumpe el listado: se devuelve un mapa vacío y los
// servidores conservan el estado unknown.
func backendStats(ctx context.Context, session ports.Session) map[string]ServerStats {
	cmd := &domain.Command{Path: binSh, Args: []string{"-c", showStatCommand}}

	result, err := session.Run(ctx, cmd)
	if err != nil {
		return map[string]ServerStats{}
	}

	return parseServerStats(result.Stdout)
}

// parseStates traduce el campo status de la salida CSV del socket.
func parseStates(output string) map[string]string {
	stats := parseServerStats(output)
	states := make(map[string]string, len(stats))

	for name := range stats {
		states[name] = stats[name].State
	}

	return states
}

// parseServerStats interpreta las columnas estándar de «show stat».
func parseServerStats(output string) map[string]ServerStats {
	stats := make(map[string]ServerStats)

	for _, line := range strings.Split(output, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 18 {
			continue
		}

		name := strings.TrimSpace(fields[1])
		if name == "" || name == "FRONTEND" || name == "BACKEND" {
			continue
		}

		stat := ServerStats{
			State:           stateUnknown,
			Weight:          intField(fields, 18),
			Active:          intField(fields, 19) > 0,
			Backup:          intField(fields, 20) > 0,
			CurrentSessions: intField(fields, 4),
			MaxSessions:     intField(fields, 5),
			TotalSessions:   int64Field(fields, 7),
			QueueCurrent:    intField(fields, 2),
			QueueMax:        intField(fields, 3),
			BytesIn:         int64Field(fields, 8),
			BytesOut:        int64Field(fields, 9),
			CheckStatus:     checkStatus(fields),
			CheckCode:       intField(fields, 37),
			CheckDuration:   intField(fields, 38),
			Downtime:        int64Field(fields, 24),
			LastChange:      int64Field(fields, 23),
			LastSession:     int64Field(fields, 55),
			LastCheck:       checkMessage(fields),
		}

		if state, ok := haproxyState(fields[17]); ok {
			stat.State = state
		}

		stats[name] = stat
	}

	return stats
}

// intField devuelve un entero de la columna o cero.
func intField(fields []string, index int) int {
	if index >= len(fields) {
		return 0
	}

	value, err := strconv.Atoi(strings.TrimSpace(fields[index]))
	if err != nil {
		return 0
	}

	return value
}

// int64Field devuelve un entero largo de la columna o cero.
func int64Field(fields []string, index int) int64 {
	if index >= len(fields) {
		return 0
	}

	value, err := strconv.ParseInt(strings.TrimSpace(fields[index]), 10, 64)
	if err != nil {
		return 0
	}

	return value
}

// stringField devuelve el texto de la columna o una cadena vacía.
func stringField(fields []string, index int) string {
	if index >= len(fields) {
		return ""
	}

	return strings.TrimSpace(fields[index])
}

// checkStatus devuelve el resultado del último chequeo de salud. HAProxy
// antepone «* » cuando hay un chequeo en curso; ese prefijo no forma
// parte del resultado y se descarta.
func checkStatus(fields []string) string {
	return strings.TrimSpace(strings.TrimPrefix(stringField(fields, 36), "*"))
}

// checkMessage devuelve el contenido del último chequeo. HAProxy publica
// «0» cuando no tiene un mensaje que mostrar, así que se informa como
// vacío para no confundirlo con un resultado real.
func checkMessage(fields []string) string {
	message := stringField(fields, 56)
	if message == "0" {
		return ""
	}

	return message
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

// Status inspecciona HAProxy sin modificar la máquina.
func (h *HAProxy) Status(ctx context.Context) (HAProxyStatus, error) {
	session, err := h.sessions.Open(ctx)
	if err != nil {
		return HAProxyStatus{}, fmt.Errorf("abrir la sesión para revisar HAProxy: %w", err)
	}

	defer closeSession(session)

	return h.inspect(ctx, session)
}

// inspect reúne el estado de HAProxy a partir de la sesión abierta.
func (h *HAProxy) inspect(ctx context.Context, session ports.Session) (HAProxyStatus, error) {
	status := HAProxyStatus{Problems: []string{}, Suggestions: []string{}}

	status.Installed = commandOK(ctx, session, binSh, "-c", "command -v "+serviceHAProxy+" >/dev/null 2>&1")

	config, err := runOutput(ctx, session, binCat, haproxyConfigPath)
	status.ConfigExists = err == nil
	status.ConfigValid = status.Installed && status.ConfigExists &&
		commandOK(ctx, session, binHaproxy, "-c", "-f", haproxyConfigPath)
	_, status.Backend = parseServers(config)
	status.ServiceActive = commandOK(ctx, session, binSystemctl, "is-active", "--quiet", serviceHAProxy)
	status.Socat = commandOK(ctx, session, binSh, "-c", "test -x "+binSocat)
	status.Port = configuredPort(config)

	evaluateHAProxyStatus(&status)

	return status, nil
}

// evaluateHAProxyStatus llena problemas y sugerencias del estado.
func evaluateHAProxyStatus(status *HAProxyStatus) {
	evaluateInstallation(status)

	if status.Installed && status.ConfigExists && !status.Backend {
		status.Problems = append(status.Problems, "La configuración no tiene bloques backend.")
	}

	if status.Installed && !status.ServiceActive {
		status.Problems = append(status.Problems, "El servicio haproxy no está activo.")
	}

	if !status.Socat {
		status.Suggestions = append(status.Suggestions, "Instala socat para ver el estado up/down de los servidores.")
	}

	if len(status.Problems) > 0 {
		status.Suggestions = append(status.Suggestions, "Puedes usar la configuración rápida; se guarda una copia de la configuración anterior.")
	}
}

// evaluateInstallation agrega el problema del binario o del archivo.
func evaluateInstallation(status *HAProxyStatus) {
	switch {
	case !status.Installed:
		status.Problems = append(status.Problems, "HAProxy no está instalado.")
		status.Suggestions = append(status.Suggestions, "Usa la configuración rápida para instalarlo y dejar un back-end listo.")
	case !status.ConfigExists:
		status.Problems = append(status.Problems, "No existe /etc/haproxy/haproxy.cfg.")
	case !status.ConfigValid:
		status.Problems = append(status.Problems, "La configuración de HAProxy no es válida.")
	}
}

// Setup instala HAProxy si falta y deja una configuración mínima con un
// back-end vacío. Si la configuración actual es válida y tiene back-end,
// solo asegura el servicio.
func (h *HAProxy) Setup(ctx context.Context) (Outcome, error) {
	session, err := h.sessions.Open(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("abrir la sesión para configurar HAProxy: %w", err)
	}

	defer closeSession(session)

	status, err := h.inspect(ctx, session)
	if err != nil {
		return Outcome{}, err
	}

	if !status.Installed {
		output, installErr := h.installHAProxy(ctx, session)
		if installErr != nil {
			return output, installErr
		}

		status, err = h.inspect(ctx, session)
		if err != nil {
			return output, err
		}
	}

	if status.ConfigValid && status.Backend {
		return h.ensureService(ctx, session, status)
	}

	return h.replaceConfig(ctx, session, status)
}

// installHAProxy instala el binario y socat con APT.
func (h *HAProxy) installHAProxy(ctx context.Context, session ports.Session) (Outcome, error) {
	cmd := &domain.Command{Path: binSh, Args: []string{
		"-c",
		"DEBIAN_FRONTEND=noninteractive apt-get update -qq && " +
			"DEBIAN_FRONTEND=noninteractive apt-get install -y haproxy socat",
	}}

	result, err := session.Run(ctx, cmd)

	output := Outcome{
		Message:   "HAProxy instalado y configurado.",
		Operation: "apt-get install -y haproxy",
		Output:    result.Stdout,
		Stderr:    result.Stderr,
		Code:      result.ExitCode,
	}

	if err != nil {
		return output, fmt.Errorf("instalar HAProxy: %w", err)
	}

	return output, nil
}

// ensureService activa HAProxy y socat si la configuración ya es válida.
func (h *HAProxy) ensureService(ctx context.Context, session ports.Session, status HAProxyStatus) (Outcome, error) {
	activated := false

	if !status.ServiceActive {
		cmd := &domain.Command{Path: binSystemctl, Args: []string{"enable", "--now", serviceHAProxy}}
		result, err := session.Run(ctx, cmd)

		output := Outcome{
			Message:   "Servicio HAProxy activado.",
			Operation: cmd.ShellLine(),
			Output:    result.Stdout,
			Stderr:    result.Stderr,
			Code:      result.ExitCode,
		}

		if err != nil {
			return output, fmt.Errorf("activar HAProxy: %w", err)
		}

		activated = true
	}

	if !status.Socat {
		socat, err := h.installSocat(ctx, session)
		if err != nil {
			mensaje := "HAProxy ya estaba configurado"
			if activated {
				mensaje = "Servicio HAProxy activado"
			}

			return Outcome{
				Message: mensaje + "; socat no se pudo instalar, los estados se verán como unknown.",
				Stderr:  socat.Stderr,
			}, nil
		}
	}

	if activated {
		return Outcome{Message: "Servicio HAProxy activado."}, nil
	}

	return Outcome{Message: "HAProxy ya estaba configurado."}, nil
}

// replaceConfig guarda la configuración anterior y escribe la mínima.
func (h *HAProxy) replaceConfig(ctx context.Context, session ports.Session, status HAProxyStatus) (Outcome, error) {
	port, err := pickPort(ctx, session)
	if err != nil {
		return Outcome{}, err
	}

	backup := ""
	if status.ConfigExists {
		backup = "cp -a " + domain.Quote(haproxyConfigPath) + " " +
			domain.Quote(haproxyConfigPath+".bak-"+time.Now().Format("20060102-150405")) + " && "
	}

	b64 := base64.StdEncoding.EncodeToString([]byte(defaultHAProxyConfig(port)))
	script := backup +
		"printf '%s' " + domain.Quote(b64) + " | base64 -d > /tmp/haproxy.cfg.vmcons && " +
		binHaproxy + " -c -f /tmp/haproxy.cfg.vmcons && " +
		"mv /tmp/haproxy.cfg.vmcons " + haproxyConfigPath + " && " +
		binSystemctl + " enable " + serviceHAProxy + " && " +
		binSystemctl + " restart " + serviceHAProxy

	cmd := &domain.Command{Path: binSh, Args: []string{"-c", script}}
	result, err := session.Run(ctx, cmd)

	output := Outcome{
		Message:   fmt.Sprintf("HAProxy configurado en el puerto %d con el back-end «vmcons_app».", port),
		Operation: "instalar configuración mínima de HAProxy y reiniciar",
		Output:    result.Stdout,
		Stderr:    result.Stderr,
		Code:      result.ExitCode,
	}

	if err != nil {
		return output, fmt.Errorf("configurar HAProxy: %w", err)
	}

	if !status.Socat {
		if socat, socatErr := h.installSocat(ctx, session); socatErr != nil {
			output.Message += " socat no se pudo instalar; los estados se verán como unknown."
			output.Stderr = socat.Stderr
		}
	}

	return output, nil
}

// configuredPort extrae el puerto del bind del frontend.
func configuredPort(config string) int {
	re := regexp.MustCompile(`(?m)^\s*bind\s+\*:(\d+)`)

	coincidencias := re.FindStringSubmatch(config)
	if len(coincidencias) < 2 {
		return 0
	}

	port, err := strconv.Atoi(coincidencias[1])
	if err != nil {
		return 0
	}

	return port
}

// pickPort devuelve 80 si está libre o lo ocupa HAProxy; si no, el primer
// puerto libre entre 8080 y 8090.
func pickPort(ctx context.Context, session ports.Session) (int, error) {
	candidatos := []int{80}
	for port := 8080; port <= 8090; port++ {
		candidatos = append(candidatos, port)
	}

	for _, port := range candidatos {
		filtro := domain.Quote("sport = :" + strconv.Itoa(port))
		cmd := &domain.Command{Path: binSh, Args: []string{"-c", "ss -ltnpH " + filtro + " 2>/dev/null"}}

		result, err := session.Run(ctx, cmd)
		if err != nil {
			return 0, fmt.Errorf("comprobar el puerto %d: %w", port, err)
		}

		salida := strings.TrimSpace(result.Stdout)

		if salida == "" || strings.Contains(salida, serviceHAProxy) {
			return port, nil
		}
	}

	return 0, errors.New("no hay un puerto libre para HAProxy")
}

// installSocat instala socat para leer el socket de estadísticas.
func (h *HAProxy) installSocat(ctx context.Context, session ports.Session) (Outcome, error) {
	cmd := &domain.Command{Path: binSh, Args: []string{
		"-c",
		"DEBIAN_FRONTEND=noninteractive apt-get install -y socat",
	}}

	result, err := session.Run(ctx, cmd)

	output := Outcome{
		Message:   "socat instalado para consultar el estado de los servidores.",
		Operation: "apt-get install -y socat",
		Output:    result.Stdout,
		Stderr:    result.Stderr,
		Code:      result.ExitCode,
	}

	if err != nil {
		return output, fmt.Errorf("instalar socat: %w", err)
	}

	return output, nil
}

// commandOK ejecuta una orden y devuelve true si termina sin error.
func commandOK(ctx context.Context, session ports.Session, path string, args ...string) bool {
	_, err := session.Run(ctx, &domain.Command{Path: path, Args: args})

	return err == nil
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
			State:   stateUnknown,
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
