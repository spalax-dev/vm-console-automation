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
	flagNoPager  = "--no-pager"
	estadoActivo = "active"
)

// ErrUnidadNoEncontrada indica que la unidad no existe en la máquina.
var ErrUnidadNoEncontrada = errors.New("la unidad no existe")

// ArchivoDespliegue describe un archivo que se transfiere a la máquina.
type ArchivoDespliegue struct {
	// Origen es la ruta del archivo en el anfitrión.
	Origen string
	// Destino es la ruta absoluta en la máquina.
	Destino string
	// Modo son los permisos en octal.
	Modo string
}

// EstadoUnidad describe una unidad de systemd.
type EstadoUnidad struct {
	// Unidad es el nombre de la unidad, con extensión .service.
	Unidad string
	// Activo indica si está en ejecución.
	Activo bool
	// Habilitado indica si arranca con el sistema.
	Habilitado bool
	// Estado es active, inactive, failed o unknown.
	Estado string
	// Detalle es el subestado informado por systemd.
	Detalle string
}

// Systemd implementa el caso de uso de gestión de unidades.
type Systemd struct {
	sesiones AbridorSesion
}

// NuevoSystemd construye el servicio con el abridor de sesiones dado.
func NuevoSystemd(sesiones AbridorSesion) *Systemd {
	return &Systemd{sesiones: sesiones}
}

// Listar devuelve las unidades de servicio con su estado.
func (s *Systemd) Listar(ctx context.Context) ([]EstadoUnidad, error) {
	sesion, err := s.sesiones.Abrir(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrir la sesión para listar unidades: %w", err)
	}

	defer cerrarSesion(sesion)

	unidades, err := ejecutarSalida(ctx, sesion, binSystemctl,
		"list-units", "--type=service", "--all", "--no-legend", "--plain", flagNoPager)
	if err != nil {
		return nil, fmt.Errorf("listar las unidades: %w", err)
	}

	archivos, err := ejecutarSalida(ctx, sesion, binSystemctl,
		"list-unit-files", "--type=service", "--no-legend", flagNoPager)
	if err != nil {
		return nil, fmt.Errorf("consultar la habilitación de las unidades: %w", err)
	}

	habilitadas := mapaHabilitacion(archivos)
	lista := []EstadoUnidad{}

	for _, linea := range strings.Split(unidades, "\n") {
		campos := strings.Fields(linea)
		if len(campos) > 0 && campos[0] == "●" {
			campos = campos[1:]
		}

		if len(campos) < 4 || !strings.HasSuffix(campos[0], ".service") {
			continue
		}

		activo := campos[2]
		lista = append(lista, EstadoUnidad{
			Unidad:     campos[0],
			Activo:     activo == estadoActivo,
			Habilitado: habilitadas[campos[0]],
			Estado:     mapearEstado(activo),
			Detalle:    campos[3],
		})
	}

	return lista, nil
}

// Obtener devuelve el estado de una unidad.
func (s *Systemd) Obtener(ctx context.Context, unidad string) (EstadoUnidad, error) {
	sesion, err := s.sesiones.Abrir(ctx)
	if err != nil {
		return EstadoUnidad{}, fmt.Errorf("abrir la sesión para consultar la unidad: %w", err)
	}

	defer cerrarSesion(sesion)

	salida, err := ejecutarSalida(ctx, sesion, binSystemctl, "show", unidad,
		"--property=LoadState,ActiveState,UnitFileState,SubState", flagNoPager)
	if err != nil {
		var exitErr *domain.ExitError
		if errors.As(err, &exitErr) && exitErr.Code == 4 {
			return EstadoUnidad{}, ErrUnidadNoEncontrada
		}

		return EstadoUnidad{}, fmt.Errorf("consultar la unidad %q: %w", unidad, err)
	}

	propiedades := parsearPropiedades(salida)
	if propiedades["LoadState"] == "not-found" {
		return EstadoUnidad{}, ErrUnidadNoEncontrada
	}

	return EstadoUnidad{
		Unidad:     unidad,
		Activo:     propiedades["ActiveState"] == estadoActivo,
		Habilitado: propiedades["UnitFileState"] == "enabled",
		Estado:     mapearEstado(propiedades["ActiveState"]),
		Detalle:    propiedades["SubState"],
	}, nil
}

// Desplegar transfiere los archivos de una unidad, ajusta permisos y
// recarga systemd.
func (s *Systemd) Desplegar(
	ctx context.Context, unidad string, archivos []ArchivoDespliegue, usuario, grupo string,
) (Resultado, error) {
	sesion, err := s.sesiones.Abrir(ctx)
	if err != nil {
		return Resultado{}, fmt.Errorf("abrir la sesión para desplegar la unidad: %w", err)
	}

	defer cerrarSesion(sesion)

	for _, archivo := range archivos {
		if err := sesion.Put(ctx, archivo.Origen, archivo.Destino); err != nil {
			return Resultado{}, fmt.Errorf("transferir %q: %w", archivo.Origen, err)
		}

		if _, err := sesion.Run(ctx, &domain.Command{
			Path: binChmod,
			Args: []string{archivo.Modo, archivo.Destino},
		}); err != nil {
			return Resultado{}, fmt.Errorf("ajustar permisos de %q: %w", archivo.Destino, err)
		}

		if usuario != "" {
			dueno := usuario
			if grupo != "" {
				dueno += ":" + grupo
			}

			if _, err := sesion.Run(ctx, &domain.Command{
				Path: binChown,
				Args: []string{dueno, archivo.Destino},
			}); err != nil {
				return Resultado{}, fmt.Errorf("ajustar propietario de %q: %w", archivo.Destino, err)
			}
		}
	}

	return s.ejecutar(ctx, sesion, "Unidad «"+unidad+"» desplegada.", "daemon-reload")
}

// CambiarEstado activa o desactiva la unidad, o su arranque con el
// sistema.
func (s *Systemd) CambiarEstado(
	ctx context.Context, unidad string, activo, habilitado *bool,
) (Resultado, error) {
	var args []string

	switch {
	case activo != nil && *activo:
		args = []string{"start", unidad}
	case activo != nil:
		args = []string{"stop", unidad}
	case habilitado != nil && *habilitado:
		args = []string{"enable", unidad}
	case habilitado != nil:
		args = []string{"disable", unidad}
	default:
		return Resultado{}, fmt.Errorf("no se indicó un cambio de estado para %q", unidad)
	}

	sesion, err := s.sesiones.Abrir(ctx)
	if err != nil {
		return Resultado{}, fmt.Errorf("abrir la sesión para cambiar la unidad: %w", err)
	}

	defer cerrarSesion(sesion)

	return s.ejecutar(ctx, sesion, "Unidad «"+unidad+"» actualizada.", args...)
}

// Reiniciar reinicia una unidad.
func (s *Systemd) Reiniciar(ctx context.Context, unidad string) (Resultado, error) {
	sesion, err := s.sesiones.Abrir(ctx)
	if err != nil {
		return Resultado{}, fmt.Errorf("abrir la sesión para reiniciar la unidad: %w", err)
	}

	defer cerrarSesion(sesion)

	return s.ejecutar(ctx, sesion, "Unidad «"+unidad+"» reiniciada.", "restart", unidad)
}

// ejecutar corre systemctl en la sesión y arma el resultado.
func (s *Systemd) ejecutar(
	ctx context.Context, sesion ports.Session, mensaje string, args ...string,
) (Resultado, error) {
	cmd := &domain.Command{Path: binSystemctl, Args: args}

	resultado, err := sesion.Run(ctx, cmd)

	salida := Resultado{
		Mensaje:   mensaje,
		Operacion: cmd.ShellLine(),
		Salida:    resultado.Stdout,
		Stderr:    resultado.Stderr,
		Codigo:    resultado.ExitCode,
	}

	if err != nil {
		return salida, fmt.Errorf("ejecutar systemctl %s: %w", strings.Join(args, " "), err)
	}

	return salida, nil
}

// mapaHabilitacion interpreta «systemctl list-unit-files».
func mapaHabilitacion(salida string) map[string]bool {
	habilitadas := map[string]bool{}

	for _, linea := range strings.Split(salida, "\n") {
		campos := strings.Fields(linea)
		if len(campos) < 2 || !strings.HasSuffix(campos[0], ".service") {
			continue
		}

		habilitadas[campos[0]] = campos[1] == "enabled"
	}

	return habilitadas
}

// parsearPropiedades interpreta la salida «clave=valor» de systemctl.
func parsearPropiedades(salida string) map[string]string {
	propiedades := map[string]string{}

	for _, linea := range strings.Split(salida, "\n") {
		clave, valor, encontrado := strings.Cut(strings.TrimSpace(linea), "=")
		if encontrado {
			propiedades[clave] = valor
		}
	}

	return propiedades
}

// mapearEstado traduce el estado de systemd al del contrato.
func mapearEstado(active string) string {
	switch active {
	case estadoActivo:
		return estadoActivo
	case "inactive":
		return "inactive"
	case "failed":
		return "failed"
	default:
		return "unknown"
	}
}
