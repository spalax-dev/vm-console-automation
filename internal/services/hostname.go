// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// HostnameConfig describe el nombre de la máquina y su entrada en
// /etc/hosts.
type HostnameConfig struct {
	// Name es el nombre de host actual o deseado.
	Name string
	// IP es la dirección IPv4 asociada.
	IP string
	// UpdateHosts indica si el nombre está registrado en /etc/hosts.
	UpdateHosts bool
}

// Hostname implementa el caso de uso del nombre de host y /etc/hosts.
type Hostname struct {
	sessions SessionOpener
}

// NewHostname construye el servicio de nombre de host con el abridor
// de sesiones dado.
func NewHostname(sessions SessionOpener) *Hostname {
	return &Hostname{sessions: sessions}
}

// Get lee el nombre, la primera IPv4 y la presencia del nombre en
// /etc/hosts.
func (h *Hostname) Get(ctx context.Context) (HostnameConfig, error) {
	session, err := h.sessions.Open(ctx)
	if err != nil {
		return HostnameConfig{}, fmt.Errorf("abrir la sesión para leer el nombre de host: %w", err)
	}

	defer closeSession(session)

	name, err := runOutput(ctx, session, binCat, "/etc/hostname")
	if err != nil {
		return HostnameConfig{}, fmt.Errorf("leer /etc/hostname: %w", err)
	}

	ip, err := getIPv4(ctx, session)
	if err != nil {
		return HostnameConfig{}, err
	}

	hosts, err := runOutput(ctx, session, binCat, "/etc/hosts")
	if err != nil {
		return HostnameConfig{}, fmt.Errorf("leer /etc/hosts: %w", err)
	}

	name = strings.TrimSpace(name)

	return HostnameConfig{
		Name:        name,
		IP:          ip,
		UpdateHosts: nameInHosts(hosts, name),
	}, nil
}

// Write cambia el nombre de host y, si se pide, actualiza /etc/hosts.
func (h *Hostname) Write(ctx context.Context, config HostnameConfig) (Outcome, error) {
	session, err := h.sessions.Open(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("abrir la sesión para cambiar el nombre de host: %w", err)
	}

	defer closeSession(session)

	guion := "hostnamectl set-hostname " + domain.Quote(config.Name)

	if config.UpdateHosts {
		hosts, err := runOutput(ctx, session, binCat, "/etc/hosts")
		if err != nil {
			return Outcome{}, fmt.Errorf("leer /etc/hosts: %w", err)
		}

		guion += " && printf '%s' " + domain.Quote(hostsContent(hosts, config.Name, config.IP)) +
			" > /etc/hosts"
	}

	cmd := &domain.Command{Path: binSh, Args: []string{"-c", guion}}

	result, err := session.Run(ctx, cmd)

	output := Outcome{
		Message:   "Nombre de host actualizado.",
		Operation: cmd.ShellLine(),
		Output:    result.Stdout,
		Stderr:    result.Stderr,
		Code:      result.ExitCode,
	}

	if err != nil {
		return output, fmt.Errorf("actualizar el nombre de host: %w", err)
	}

	return output, nil
}

// runOutput ejecuta una orden y devuelve su salida estándar.
func runOutput(ctx context.Context, session ports.Session, path string, args ...string) (string, error) {
	result, err := session.Run(ctx, &domain.Command{Path: path, Args: args})
	if err != nil {
		return "", fmt.Errorf("ejecutar %s: %w", path, err)
	}

	return result.Stdout, nil
}

// getIPv4 devuelve la primera dirección IPv4 de «hostname -I».
func getIPv4(ctx context.Context, session ports.Session) (string, error) {
	output, err := runOutput(ctx, session, "/usr/bin/hostname", "-I")
	if err != nil {
		return "", fmt.Errorf("consultar la dirección IP: %w", err)
	}

	for _, field := range strings.Fields(output) {
		address, err := netip.ParseAddr(field)
		if err == nil && address.Is4() {
			return address.String(), nil
		}
	}

	return "", fmt.Errorf("la máquina no informó una dirección IPv4")
}

// nameInHosts informa si el nombre aparece como destino en el
// contenido de /etc/hosts.
func nameInHosts(content, name string) bool {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && slices.Contains(fields[1:], name) {
			return true
		}
	}

	return false
}

// hostsContent devuelve /etc/hosts sin las líneas que mapean el
// nombre y con la entrada «ip nombre» al final.
func hostsContent(content, name, ip string) string {
	lines := []string{}

	for _, line := range strings.Split(content, "\n") {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 && slices.Contains(fields[1:], name) {
			continue
		}

		lines = append(lines, line)
	}

	lines = append(lines, ip+" "+name)

	return strings.Join(lines, "\n") + "\n"
}
