// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// Valores de systemctl usados en varias órdenes.
const (
	flagNotPager = "--no-pager"
	stateActive  = "active"
)

// ErrUnitNotFound indica que la unidad no existe en la máquina.
var ErrUnitNotFound = errors.New("la unidad no existe")

// DeploymentFile describe un archivo que se transfiere a la máquina.
type DeploymentFile struct {
	// Source es la ruta del archivo en el anfitrión.
	Source string
	// Destination es la ruta absoluta en la máquina.
	Destination string
	// Mode son los permisos en octal.
	Mode string
}

// UnitState describe una unidad de systemd.
type UnitState struct {
	// Unit es el nombre de la unidad, con extensión .service.
	Unit string
	// Active indica si está en ejecución.
	Active bool
	// Enabled indica si arranca con el sistema.
	Enabled bool
	// State es active, inactive, failed o unknown.
	State string
	// Detail es el subestado informado por systemd.
	Detail string
}

// Systemd implementa el caso de uso de gestión de unidades.
type Systemd struct {
	sessions SessionOpener
}

// NewSystemd construye el servicio con el abridor de sesiones dado.
func NewSystemd(sessions SessionOpener) *Systemd {
	return &Systemd{sessions: sessions}
}

// List devuelve las unidades de servicio con su estado.
func (s *Systemd) List(ctx context.Context) ([]UnitState, error) {
	session, err := s.sessions.Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrir la sesión para listar unidades: %w", err)
	}

	defer closeSession(session)

	units, err := runOutput(ctx, session, binSystemctl,
		"list-units", "--type=service", "--all", "--no-legend", "--plain", flagNotPager)
	if err != nil {
		return nil, fmt.Errorf("listar las unidades: %w", err)
	}

	files, err := runOutput(ctx, session, binSystemctl,
		"list-unit-files", "--type=service", "--no-legend", flagNotPager)
	if err != nil {
		return nil, fmt.Errorf("consultar la habilitación de las unidades: %w", err)
	}

	habilitadas := enabledMap(files)
	lists := []UnitState{}

	for _, line := range strings.Split(units, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == "●" {
			fields = fields[1:]
		}

		if len(fields) < 4 || !strings.HasSuffix(fields[0], ".service") {
			continue
		}

		active := fields[2]
		lists = append(lists, UnitState{
			Unit:    fields[0],
			Active:  active == stateActive,
			Enabled: habilitadas[fields[0]],
			State:   mapState(active),
			Detail:  fields[3],
		})
	}

	return lists, nil
}

// Get devuelve el estado de una unidad.
func (s *Systemd) Get(ctx context.Context, unit string) (UnitState, error) {
	session, err := s.sessions.Open(ctx)
	if err != nil {
		return UnitState{}, fmt.Errorf("abrir la sesión para consultar la unidad: %w", err)
	}

	defer closeSession(session)

	output, err := runOutput(ctx, session, binSystemctl, "show", unit,
		"--property=LoadState,ActiveState,UnitFileState,SubState", flagNotPager)
	if err != nil {
		var exitErr *domain.ExitError
		if errors.As(err, &exitErr) && exitErr.Code == 4 {
			return UnitState{}, ErrUnitNotFound
		}

		return UnitState{}, fmt.Errorf("consultar la unidad %q: %w", unit, err)
	}

	properties := parseProperties(output)
	if properties["LoadState"] == "not-found" {
		return UnitState{}, ErrUnitNotFound
	}

	return UnitState{
		Unit:    unit,
		Active:  properties["ActiveState"] == stateActive,
		Enabled: properties["UnitFileState"] == "enabled",
		State:   mapState(properties["ActiveState"]),
		Detail:  properties["SubState"],
	}, nil
}

// Deploy transfiere los archivos de una unidad, ajusta permisos y
// recarga systemd.
func (s *Systemd) Deploy(
	ctx context.Context, unit string, files []DeploymentFile, user, group string,
) (Outcome, error) {
	session, err := s.sessions.Open(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("abrir la sesión para desplegar la unidad: %w", err)
	}

	defer closeSession(session)

	for _, file := range files {
		if err := session.Put(ctx, file.Source, file.Destination); err != nil {
			return Outcome{}, fmt.Errorf("transferir %q: %w", file.Source, err)
		}

		if _, err := session.Run(ctx, &domain.Command{
			Path: binChmod,
			Args: []string{file.Mode, file.Destination},
		}); err != nil {
			return Outcome{}, fmt.Errorf("ajustar permisos de %q: %w", file.Destination, err)
		}

		if user != "" {
			dueno := user
			if group != "" {
				dueno += ":" + group
			}

			if _, err := session.Run(ctx, &domain.Command{
				Path: binChown,
				Args: []string{dueno, file.Destination},
			}); err != nil {
				return Outcome{}, fmt.Errorf("ajustar propietario de %q: %w", file.Destination, err)
			}
		}
	}

	return s.run(ctx, session, "Unidad «"+unit+"» desplegada.", "daemon-reload")
}

// ChangeState activa o desactiva la unidad, o su arranque con el
// sistema.
func (s *Systemd) ChangeState(
	ctx context.Context, unit string, active, enabled *bool,
) (Outcome, error) {
	var args []string

	switch {
	case active != nil && *active:
		args = []string{"start", unit}
	case active != nil:
		args = []string{"stop", unit}
	case enabled != nil && *enabled:
		args = []string{"enable", unit}
	case enabled != nil:
		args = []string{"disable", unit}
	default:
		return Outcome{}, fmt.Errorf("no se indicó un cambio de estado para %q", unit)
	}

	session, err := s.sessions.Open(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("abrir la sesión para cambiar la unidad: %w", err)
	}

	defer closeSession(session)

	return s.run(ctx, session, "Unidad «"+unit+"» actualizada.", args...)
}

// Restart reinicia una unidad.
func (s *Systemd) Restart(ctx context.Context, unit string) (Outcome, error) {
	session, err := s.sessions.Open(ctx)
	if err != nil {
		return Outcome{}, fmt.Errorf("abrir la sesión para reiniciar la unidad: %w", err)
	}

	defer closeSession(session)

	return s.run(ctx, session, "Unidad «"+unit+"» reiniciada.", "restart", unit)
}

// run corre systemctl en la sesión y arma el resultado.
func (s *Systemd) run(
	ctx context.Context, session ports.Session, message string, args ...string,
) (Outcome, error) {
	cmd := &domain.Command{Path: binSystemctl, Args: args}

	result, err := session.Run(ctx, cmd)

	output := Outcome{
		Message:   message,
		Operation: cmd.ShellLine(),
		Output:    result.Stdout,
		Stderr:    result.Stderr,
		Code:      result.ExitCode,
	}

	if err != nil {
		return output, fmt.Errorf("ejecutar systemctl %s: %w", strings.Join(args, " "), err)
	}

	return output, nil
}

// enabledMap interpreta «systemctl list-unit-files».
func enabledMap(output string) map[string]bool {
	habilitadas := map[string]bool{}

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.HasSuffix(fields[0], ".service") {
			continue
		}

		habilitadas[fields[0]] = fields[1] == "enabled"
	}

	return habilitadas
}

// parseProperties interpreta la salida «clave=valor» de systemctl.
func parseProperties(output string) map[string]string {
	properties := map[string]string{}

	for _, line := range strings.Split(output, "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if found {
			properties[key] = value
		}
	}

	return properties
}

// mapState traduce el estado de systemd al del contrato.
func mapState(active string) string {
	switch active {
	case stateActive:
		return stateActive
	case "inactive":
		return "inactive"
	case "failed":
		return "failed"
	default:
		return "unknown"
	}
}
