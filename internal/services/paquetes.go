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

// AbridorSesion obtiene una sesión con la máquina administrada.
type AbridorSesion interface {
	// Abrir devuelve una sesión lista para ejecutar órdenes.
	Abrir(ctx context.Context) (ports.Session, error)
}

// Paquete describe un paquete instalado en la máquina.
type Paquete struct {
	// Nombre es el nombre del paquete.
	Nombre string
	// Version es la versión instalada.
	Version string
	// Estado es la presencia del paquete; siempre «installed» en los
	// resultados del listado.
	Estado string
}

// Resultado describe el resultado de una operación ejecutada en la
// máquina, con la información necesaria para notificar y diagnosticar.
type Resultado struct {
	// Mensaje es el texto comprensible para el usuario.
	Mensaje string
	// Operacion es la orden ejecutada.
	Operacion string
	// Salida es la salida estándar relevante.
	Salida string
	// Stderr es la salida de error de la orden.
	Stderr string
	// Codigo es el código de salida del proceso remoto.
	Codigo int
}

// Paquetes implementa el caso de uso de gestión de paquetes APT.
type Paquetes struct {
	sesiones AbridorSesion
}

// NuevoPaquetes construye el servicio de paquetes con el abridor de
// sesiones dado.
func NuevoPaquetes(sesiones AbridorSesion) *Paquetes {
	return &Paquetes{sesiones: sesiones}
}

// Instalar instala el paquete indicado con APT.
func (p *Paquetes) Instalar(ctx context.Context, nombre string) (Resultado, error) {
	cmd := &domain.Command{
		Path: "/usr/bin/apt-get",
		Args: []string{"install", "-y", nombre},
		Env:  []string{"DEBIAN_FRONTEND=noninteractive"},
	}

	return p.ejecutar(ctx, "Paquete «"+nombre+"» instalado.", "instalar el paquete «"+nombre+"»", cmd)
}

// Eliminar elimina el paquete indicado con APT.
func (p *Paquetes) Eliminar(ctx context.Context, nombre string) (Resultado, error) {
	cmd := &domain.Command{
		Path: "/usr/bin/apt-get",
		Args: []string{"remove", "-y", nombre},
		Env:  []string{"DEBIAN_FRONTEND=noninteractive"},
	}

	return p.ejecutar(ctx, "Paquete «"+nombre+"» eliminado.", "eliminar el paquete «"+nombre+"»", cmd)
}

// Listar devuelve los paquetes instalados en la máquina.
func (p *Paquetes) Listar(ctx context.Context) ([]Paquete, error) {
	sesion, err := p.sesiones.Abrir(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrir la sesión para listar los paquetes: %w", err)
	}

	defer cerrarSesion(sesion)

	cmd := &domain.Command{
		Path: "/usr/bin/dpkg-query",
		Args: []string{"-W", "-f=${Package}\t${Version}\t${Status}\n"},
	}

	resultado, err := sesion.Run(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("listar los paquetes instalados: %w", err)
	}

	return parsearPaquetes(resultado.Stdout), nil
}

// ejecutar abre una sesión, ejecuta la orden y la cierra, traduciendo el
// resultado del dominio.
func (p *Paquetes) ejecutar(
	ctx context.Context, mensajeExito, contextoFallo string, cmd *domain.Command,
) (Resultado, error) {
	sesion, err := p.sesiones.Abrir(ctx)
	if err != nil {
		return Resultado{}, fmt.Errorf("abrir la sesión para ejecutar la orden: %w", err)
	}

	defer cerrarSesion(sesion)

	resultado, err := sesion.Run(ctx, cmd)

	salida := Resultado{
		Mensaje:   mensajeExito,
		Operacion: cmd.ShellLine(),
		Salida:    resultado.Stdout,
		Stderr:    resultado.Stderr,
		Codigo:    resultado.ExitCode,
	}

	if err != nil {
		return salida, fmt.Errorf("%s: %w", contextoFallo, err)
	}

	return salida, nil
}

// cerrarSesion cierra la sesión y deja constancia si el cierre falla.
func cerrarSesion(sesion ports.Session) {
	if err := sesion.Close(); err != nil {
		slog.Warn("cerrar la sesión con la máquina", "error", err)
	}
}

// parsearPaquetes extrae los paquetes instalados de la salida de
// dpkg-query, cuyas columnas son nombre, versión y estado.
func parsearPaquetes(salida string) []Paquete {
	paquetes := []Paquete{}

	for _, linea := range strings.Split(salida, "\n") {
		campos := strings.Split(linea, "\t")
		if len(campos) < 3 || campos[2] != "install ok installed" {
			continue
		}

		paquetes = append(paquetes, Paquete{
			Nombre:  campos[0],
			Version: campos[1],
			Estado:  "installed",
		})
	}

	return paquetes
}
