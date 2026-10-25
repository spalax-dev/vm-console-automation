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

// ConfiguracionHostname describe el nombre de la máquina y su entrada en
// /etc/hosts.
type ConfiguracionHostname struct {
	// Nombre es el nombre de host actual o deseado.
	Nombre string
	// IP es la dirección IPv4 asociada.
	IP string
	// ActualizarHosts indica si el nombre está registrado en /etc/hosts.
	ActualizarHosts bool
}

// Hostname implementa el caso de uso del nombre de host y /etc/hosts.
type Hostname struct {
	sesiones AbridorSesion
}

// NuevoHostname construye el servicio de nombre de host con el abridor
// de sesiones dado.
func NuevoHostname(sesiones AbridorSesion) *Hostname {
	return &Hostname{sesiones: sesiones}
}

// Obtener lee el nombre, la primera IPv4 y la presencia del nombre en
// /etc/hosts.
func (h *Hostname) Obtener(ctx context.Context) (ConfiguracionHostname, error) {
	sesion, err := h.sesiones.Abrir(ctx)
	if err != nil {
		return ConfiguracionHostname{}, fmt.Errorf("abrir la sesión para leer el nombre de host: %w", err)
	}

	defer cerrarSesion(sesion)

	nombre, err := ejecutarSalida(ctx, sesion, binCat, "/etc/hostname")
	if err != nil {
		return ConfiguracionHostname{}, fmt.Errorf("leer /etc/hostname: %w", err)
	}

	ip, err := obtenerIPv4(ctx, sesion)
	if err != nil {
		return ConfiguracionHostname{}, err
	}

	hosts, err := ejecutarSalida(ctx, sesion, binCat, "/etc/hosts")
	if err != nil {
		return ConfiguracionHostname{}, fmt.Errorf("leer /etc/hosts: %w", err)
	}

	nombre = strings.TrimSpace(nombre)

	return ConfiguracionHostname{
		Nombre:          nombre,
		IP:              ip,
		ActualizarHosts: nombreEnHosts(hosts, nombre),
	}, nil
}

// Escribir cambia el nombre de host y, si se pide, actualiza /etc/hosts.
func (h *Hostname) Escribir(ctx context.Context, config ConfiguracionHostname) (Resultado, error) {
	sesion, err := h.sesiones.Abrir(ctx)
	if err != nil {
		return Resultado{}, fmt.Errorf("abrir la sesión para cambiar el nombre de host: %w", err)
	}

	defer cerrarSesion(sesion)

	guion := "hostnamectl set-hostname " + domain.Quote(config.Nombre)

	if config.ActualizarHosts {
		hosts, err := ejecutarSalida(ctx, sesion, binCat, "/etc/hosts")
		if err != nil {
			return Resultado{}, fmt.Errorf("leer /etc/hosts: %w", err)
		}

		guion += " && printf '%s' " + domain.Quote(contenidoHosts(hosts, config.Nombre, config.IP)) +
			" > /etc/hosts"
	}

	cmd := &domain.Command{Path: binSh, Args: []string{"-c", guion}}

	resultado, err := sesion.Run(ctx, cmd)

	salida := Resultado{
		Mensaje:   "Nombre de host actualizado.",
		Operacion: cmd.ShellLine(),
		Salida:    resultado.Stdout,
		Stderr:    resultado.Stderr,
		Codigo:    resultado.ExitCode,
	}

	if err != nil {
		return salida, fmt.Errorf("actualizar el nombre de host: %w", err)
	}

	return salida, nil
}

// ejecutarSalida ejecuta una orden y devuelve su salida estándar.
func ejecutarSalida(ctx context.Context, sesion ports.Session, path string, args ...string) (string, error) {
	resultado, err := sesion.Run(ctx, &domain.Command{Path: path, Args: args})
	if err != nil {
		return "", fmt.Errorf("ejecutar %s: %w", path, err)
	}

	return resultado.Stdout, nil
}

// obtenerIPv4 devuelve la primera dirección IPv4 de «hostname -I».
func obtenerIPv4(ctx context.Context, sesion ports.Session) (string, error) {
	salida, err := ejecutarSalida(ctx, sesion, "/usr/bin/hostname", "-I")
	if err != nil {
		return "", fmt.Errorf("consultar la dirección IP: %w", err)
	}

	for _, campo := range strings.Fields(salida) {
		direccion, err := netip.ParseAddr(campo)
		if err == nil && direccion.Is4() {
			return direccion.String(), nil
		}
	}

	return "", fmt.Errorf("la máquina no informó una dirección IPv4")
}

// nombreEnHosts informa si el nombre aparece como destino en el
// contenido de /etc/hosts.
func nombreEnHosts(contenido, nombre string) bool {
	for _, linea := range strings.Split(contenido, "\n") {
		campos := strings.Fields(linea)
		if len(campos) >= 2 && slices.Contains(campos[1:], nombre) {
			return true
		}
	}

	return false
}

// contenidoHosts devuelve /etc/hosts sin las líneas que mapean el
// nombre y con la entrada «ip nombre» al final.
func contenidoHosts(contenido, nombre, ip string) string {
	lineas := []string{}

	for _, linea := range strings.Split(contenido, "\n") {
		if linea == "" {
			continue
		}

		campos := strings.Fields(linea)
		if len(campos) >= 2 && slices.Contains(campos[1:], nombre) {
			continue
		}

		lineas = append(lineas, linea)
	}

	lineas = append(lineas, ip+" "+nombre)

	return strings.Join(lineas, "\n") + "\n"
}
