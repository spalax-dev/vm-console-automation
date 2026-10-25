// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"fmt"
	"strings"

	"consola-vm/internal/core/domain"
)

// ConfiguracionIssue describe el banner previo al inicio de sesión.
type ConfiguracionIssue struct {
	// Mensaje es el texto personalizado.
	Mensaje string
	// NombreHost indica si se incluye el nombre de host.
	NombreHost bool
	// Fecha indica si se incluye la fecha.
	Fecha bool
	// Hora indica si se incluye la hora.
	Hora bool
	// IP indica si se incluye la dirección IPv4.
	IP bool
	// Contenido es el texto completo del archivo, solo en las lecturas.
	Contenido string
}

// Issue implementa el caso de uso de personalización de /etc/issue.
type Issue struct {
	sesiones AbridorSesion
}

// NuevoIssue construye el servicio de /etc/issue con el abridor de
// sesiones dado.
func NuevoIssue(sesiones AbridorSesion) *Issue {
	return &Issue{sesiones: sesiones}
}

// Etiquetas del formato de una línea por elemento que escribe la
// consola; usan las secuencias de agetty para cada dato.
const (
	etiquetaHostname = `Hostname: \n`
	etiquetaFecha    = `Fecha: \d`
	etiquetaHora     = `Hora: \t`
	etiquetaIP       = `IP: \4`
)

// Obtener lee /etc/issue y reconstruye el mensaje y las opciones.
func (i *Issue) Obtener(ctx context.Context) (ConfiguracionIssue, error) {
	sesion, err := i.sesiones.Abrir(ctx)
	if err != nil {
		return ConfiguracionIssue{}, fmt.Errorf("abrir la sesión para leer /etc/issue: %w", err)
	}

	defer cerrarSesion(sesion)

	resultado, err := sesion.Run(ctx, &domain.Command{
		Path: "/bin/cat",
		Args: []string{"/etc/issue"},
	})
	if err != nil {
		return ConfiguracionIssue{}, fmt.Errorf("leer /etc/issue: %w", err)
	}

	return parsearIssue(resultado.Stdout), nil
}

// Escribir reemplaza /etc/issue con el mensaje y las opciones dadas.
func (i *Issue) Escribir(ctx context.Context, config ConfiguracionIssue) (Resultado, error) {
	sesion, err := i.sesiones.Abrir(ctx)
	if err != nil {
		return Resultado{}, fmt.Errorf("abrir la sesión para escribir /etc/issue: %w", err)
	}

	defer cerrarSesion(sesion)

	cmd := &domain.Command{
		Path: "/bin/sh",
		Args: []string{
			"-c",
			"printf '%s' " + domain.Quote(construirIssue(config)) +
				" > /etc/issue && chmod 644 /etc/issue",
		},
	}

	resultado, err := sesion.Run(ctx, cmd)

	salida := Resultado{
		Mensaje:   "Entrada del sistema actualizada.",
		Operacion: cmd.ShellLine(),
		Salida:    resultado.Stdout,
		Stderr:    resultado.Stderr,
		Codigo:    resultado.ExitCode,
	}

	if err != nil {
		return salida, fmt.Errorf("escribir /etc/issue: %w", err)
	}

	return salida, nil
}

// construirIssue arma el contenido del archivo con una línea por
// elemento activo.
func construirIssue(config ConfiguracionIssue) string {
	lineas := []string{}

	if strings.TrimSpace(config.Mensaje) != "" {
		lineas = append(lineas, config.Mensaje)
	}

	if config.NombreHost {
		lineas = append(lineas, etiquetaHostname)
	}

	if config.Fecha {
		lineas = append(lineas, etiquetaFecha)
	}

	if config.Hora {
		lineas = append(lineas, etiquetaHora)
	}

	if config.IP {
		lineas = append(lineas, etiquetaIP)
	}

	return strings.Join(lineas, "\n") + "\n"
}

// parsearIssue reconstruye la configuración a partir del contenido del
// archivo. Las líneas que no son del formato de la consola se conservan
// como mensaje personalizado.
func parsearIssue(contenido string) ConfiguracionIssue {
	config := ConfiguracionIssue{Contenido: contenido}
	mensaje := []string{}

	for _, linea := range strings.Split(strings.TrimRight(contenido, "\n"), "\n") {
		switch strings.TrimSpace(linea) {
		case etiquetaHostname:
			config.NombreHost = true
		case etiquetaFecha:
			config.Fecha = true
		case etiquetaHora:
			config.Hora = true
		case etiquetaIP:
			config.IP = true
		default:
			if linea != "" {
				mensaje = append(mensaje, linea)
			}
		}
	}

	config.Mensaje = strings.Join(mensaje, "\n")

	return config
}
