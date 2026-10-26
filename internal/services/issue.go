// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"fmt"
	"strings"

	"consola-vm/internal/core/domain"
)

// IssueConfig describe el banner previo al inicio de sesión.
type IssueConfig struct {
	// Message es el texto personalizado.
	Message string
	// NameHost indica si se incluye el nombre de host.
	NameHost bool
	// Date indica si se incluye la fecha.
	Date bool
	// Hour indica si se incluye la hora.
	Hour bool
	// IP indica si se incluye la dirección IPv4.
	IP bool
	// Content es el texto completo del archivo, solo en las lecturas.
	Content string
}

// Issue implementa el caso de uso de personalización de /etc/issue.
type Issue struct {
	sessions SessionOpener
}

// NewIssue construye el servicio de /etc/issue con el abridor de
// sesiones dado.
func NewIssue(sessions SessionOpener) *Issue {
	return &Issue{sessions: sessions}
}

// Etiquetas del formato de una línea por elemento que escribe la
// consola; usan las secuencias de agetty para cada dato.
const (
	etiquetaHostname = `Hostname: \n`
	etiquetaDate     = `Fecha: \d`
	etiquetaHour     = `Hora: \t`
	etiquetaIP       = `IP: \4`
)

// Get lee /etc/issue y reconstruye el mensaje y las opciones.
func (i *Issue) Get(ctx context.Context) (IssueConfig, error) {
	session, err := i.sessions.Open(ctx)
	if err != nil {
		return IssueConfig{}, fmt.Errorf("abrir la sesión para leer /etc/issue: %w", err)
	}

	defer closeSession(session)

	result, err := session.Run(ctx, &domain.Command{
		Path: binCat,
		Args: []string{"/etc/issue"},
	})
	if err != nil {
		return IssueConfig{}, fmt.Errorf("leer /etc/issue: %w", err)
	}

	return parseIssue(result.Stdout), nil
}

// Write reemplaza /etc/issue con el mensaje y las opciones dadas.
func (i *Issue) Write(ctx context.Context, config IssueConfig) (Outcome, error) {
	session, err := i.sessions.Open(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("abrir la sesión para escribir /etc/issue: %w", err)
	}

	defer closeSession(session)

	cmd := &domain.Command{
		Path: binSh,
		Args: []string{
			"-c",
			"printf '%s' " + domain.Quote(buildIssue(config)) +
				" > /etc/issue && chmod 644 /etc/issue",
		},
	}

	result, err := session.Run(ctx, cmd)

	output := Outcome{
		Message:   "Entrada del sistema actualizada.",
		Operation: cmd.ShellLine(),
		Output:    result.Stdout,
		Stderr:    result.Stderr,
		Code:      result.ExitCode,
	}

	if err != nil {
		return output, fmt.Errorf("escribir /etc/issue: %w", err)
	}

	return output, nil
}

// buildIssue arma el contenido del archivo con una línea por
// elemento activo.
func buildIssue(config IssueConfig) string {
	lines := []string{}

	if strings.TrimSpace(config.Message) != "" {
		lines = append(lines, config.Message)
	}

	if config.NameHost {
		lines = append(lines, etiquetaHostname)
	}

	if config.Date {
		lines = append(lines, etiquetaDate)
	}

	if config.Hour {
		lines = append(lines, etiquetaHour)
	}

	if config.IP {
		lines = append(lines, etiquetaIP)
	}

	return strings.Join(lines, "\n") + "\n"
}

// parseIssue reconstruye la configuración a partir del contenido del
// archivo. Las líneas que no son del formato de la consola se conservan
// como mensaje personalizado.
func parseIssue(content string) IssueConfig {
	config := IssueConfig{Content: content}
	message := []string{}

	for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
		switch strings.TrimSpace(line) {
		case etiquetaHostname:
			config.NameHost = true
		case etiquetaDate:
			config.Date = true
		case etiquetaHour:
			config.Hour = true
		case etiquetaIP:
			config.IP = true
		default:
			if line != "" {
				message = append(message, line)
			}
		}
	}

	config.Message = strings.Join(message, "\n")

	return config
}
