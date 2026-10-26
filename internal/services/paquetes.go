// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// SessionOpener obtiene una sesión con la máquina administrada.
type SessionOpener interface {
	// Open devuelve una sesión lista para ejecutar órdenes.
	Open(ctx context.Context) (ports.Session, error)
}

// Package describe un paquete instalado en la máquina.
type Package struct {
	// Name es el nombre del paquete.
	Name string
	// Version es la versión instalada.
	Version string
	// State es la presencia del paquete; siempre «installed» en los
	// resultados del listado.
	State string
}

// Outcome resume el desenlace de una operación ejecutada en la
// máquina, con la información necesaria para notificar y diagnosticar.
type Outcome struct {
	// Message es el texto comprensible para el usuario.
	Message string
	// Operation es la orden ejecutada.
	Operation string
	// Output es la salida estándar relevante.
	Output string
	// Stderr es la salida de error de la orden.
	Stderr string
	// Code es el código de salida del proceso remoto.
	Code int
}

// Packages implementa el caso de uso de gestión de paquetes APT.
type Packages struct {
	sessions SessionOpener
}

// NewPackages construye el servicio de paquetes con el abridor de
// sesiones dado.
func NewPackages(sessions SessionOpener) *Packages {
	return &Packages{sessions: sessions}
}

// Install instala el paquete indicado con APT.
func (p *Packages) Install(ctx context.Context, name string) (Outcome, error) {
	cmd := &domain.Command{
		Path: "/usr/bin/apt-get",
		Args: []string{"install", "-y", name},
		Env:  []string{"DEBIAN_FRONTEND=noninteractive"},
	}

	return p.run(ctx, "Paquete «"+name+"» instalado.", "instalar el paquete «"+name+"»", cmd)
}

// Delete elimina el paquete indicado con APT.
func (p *Packages) Delete(ctx context.Context, name string) (Outcome, error) {
	cmd := &domain.Command{
		Path: "/usr/bin/apt-get",
		Args: []string{"remove", "-y", name},
		Env:  []string{"DEBIAN_FRONTEND=noninteractive"},
	}

	return p.run(ctx, "Paquete «"+name+"» eliminado.", "eliminar el paquete «"+name+"»", cmd)
}

// List devuelve los paquetes instalados en la máquina.
func (p *Packages) List(ctx context.Context) ([]Package, error) {
	session, err := p.sessions.Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrir la sesión para listar los paquetes: %w", err)
	}

	defer closeSession(session)

	cmd := &domain.Command{
		Path: "/usr/bin/dpkg-query",
		Args: []string{"-W", "-f=${Package}\t${Version}\t${Status}\n"},
	}

	result, err := session.Run(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("listar los paquetes instalados: %w", err)
	}

	return parsePackages(result.Stdout), nil
}

// run abre una sesión, ejecuta la orden y la cierra, traduciendo el
// resultado del dominio.
func (p *Packages) run(
	ctx context.Context, messageSuccess, contextoFailure string, cmd *domain.Command,
) (Outcome, error) {
	session, err := p.sessions.Open(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("abrir la sesión para ejecutar la orden: %w", err)
	}

	defer closeSession(session)

	result, err := session.Run(ctx, cmd)

	output := Outcome{
		Message:   messageSuccess,
		Operation: cmd.ShellLine(),
		Output:    result.Stdout,
		Stderr:    result.Stderr,
		Code:      result.ExitCode,
	}

	if err != nil {
		return output, fmt.Errorf("%s: %w", contextoFailure, err)
	}

	return output, nil
}

// closeSession cierra la sesión y deja constancia si el cierre falla.
func closeSession(session ports.Session) {
	if session == nil {
		return
	}

	if err := session.Close(); err != nil {
		slog.Warn("cerrar la sesión con la máquina", logErrorKey, err)
	}
}

// parsePackages extrae los paquetes instalados de la salida de
// dpkg-query, cuyas columnas son nombre, versión y estado.
func parsePackages(output string) []Package {
	packages := []Package{}

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 3 || fields[2] != "install ok installed" {
			continue
		}

		packages = append(packages, Package{
			Name:    fields[0],
			Version: fields[1],
			State:   "installed",
		})
	}

	return packages
}
