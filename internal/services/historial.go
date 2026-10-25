// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Operacion es una entrada del historial de la consola.
type Operacion struct {
	// ID es el identificador incremental.
	ID int64
	// FechaHora es la marca temporal en RFC 3339 (UTC).
	FechaHora string
	// VM es la máquina sobre la que se actuó.
	VM string
	// Modulo identifica el módulo (paquetes, usuarios, systemd, etc.).
	Modulo string
	// Operacion es el método y la ruta ejecutados.
	Operacion string
	// Descripcion es el resumen legible de la operación.
	Descripcion string
	// Resultado es «ok» o «error».
	Resultado string
	// Detalle conserva información técnica breve.
	Detalle string
}

// FiltroHistorial reúne los filtros de la consulta del historial.
type FiltroHistorial struct {
	Texto     string
	VM        string
	Modulo    string
	Desde     string
	Hasta     string
	HoraDesde string
	HoraHasta string
	Limite    int
}

// limiteHistorial es el número máximo de filas por consulta.
const limiteHistorial = 200

// Historial registra las operaciones y las consulta desde SQLite.
type Historial struct {
	db *sql.DB
}

// NuevoHistorial construye el historial sobre la base de la consola.
func NuevoHistorial(almacen *Almacen) *Historial {
	return &Historial{db: almacen.db}
}

// Registrar agrega una operación al historial.
func (h *Historial) Registrar(ctx context.Context, operacion *Operacion) error {
	if operacion.FechaHora == "" {
		operacion.FechaHora = time.Now().UTC().Format(time.RFC3339)
	}

	_, err := h.db.ExecContext(ctx, `
		INSERT INTO operaciones (fecha_hora, vm, modulo, operacion, descripcion, resultado, detalle)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		operacion.FechaHora, operacion.VM, operacion.Modulo, operacion.Operacion,
		operacion.Descripcion, operacion.Resultado, operacion.Detalle,
	)
	if err != nil {
		return fmt.Errorf("registrar la operación: %w", err)
	}

	return nil
}

// Listar devuelve las operaciones que cumplen el filtro, de la más
// reciente a la más antigua.
func (h *Historial) Listar(ctx context.Context, filtro *FiltroHistorial) ([]Operacion, error) {
	if filtro == nil {
		filtro = &FiltroHistorial{}
	}

	consulta, argumentos := consultaHistorial(filtro)

	filas, err := h.db.QueryContext(ctx, consulta, argumentos...)
	if err != nil {
		return nil, fmt.Errorf("consultar el historial: %w", err)
	}

	defer func() {
		if err := filas.Close(); err != nil {
			slog.Warn("cerrar el historial", claveErrorLog, err)
		}
	}()

	operaciones := []Operacion{}

	for filas.Next() {
		var operacion Operacion

		if err := filas.Scan(
			&operacion.ID, &operacion.FechaHora, &operacion.VM, &operacion.Modulo,
			&operacion.Operacion, &operacion.Descripcion, &operacion.Resultado, &operacion.Detalle,
		); err != nil {
			return nil, fmt.Errorf("leer una operación: %w", err)
		}

		operaciones = append(operaciones, operacion)
	}

	if err := filas.Err(); err != nil {
		return nil, fmt.Errorf("recorrer el historial: %w", err)
	}

	return operaciones, nil
}

// consultaHistorial arma el SQL y los argumentos según los filtros.
func consultaHistorial(filtro *FiltroHistorial) (consulta string, argumentos []any) {
	consulta = `SELECT id, fecha_hora, vm, modulo, operacion, descripcion, resultado, detalle FROM operaciones WHERE 1 = 1`
	argumentos = []any{}

	if filtro.Texto != "" {
		consulta += ` AND (vm LIKE ? OR modulo LIKE ? OR operacion LIKE ? OR descripcion LIKE ? OR detalle LIKE ?)`
		like := "%" + filtro.Texto + "%"
		argumentos = append(argumentos, like, like, like, like, like)
	}

	if filtro.VM != "" {
		consulta += ` AND vm = ?`

		argumentos = append(argumentos, filtro.VM)
	}

	if filtro.Modulo != "" {
		consulta += ` AND modulo = ?`

		argumentos = append(argumentos, filtro.Modulo)
	}

	if filtro.Desde != "" {
		consulta += ` AND substr(fecha_hora, 1, 10) >= ?`

		argumentos = append(argumentos, filtro.Desde)
	}

	if filtro.Hasta != "" {
		consulta += ` AND substr(fecha_hora, 1, 10) <= ?`

		argumentos = append(argumentos, filtro.Hasta)
	}

	if filtro.HoraDesde != "" {
		consulta += ` AND substr(fecha_hora, 12, 5) >= ?`

		argumentos = append(argumentos, filtro.HoraDesde)
	}

	if filtro.HoraHasta != "" {
		consulta += ` AND substr(fecha_hora, 12, 5) <= ?`

		argumentos = append(argumentos, filtro.HoraHasta)
	}

	limite := filtro.Limite
	if limite <= 0 || limite > 1000 {
		limite = limiteHistorial
	}

	consulta += ` ORDER BY id DESC LIMIT ?`

	argumentos = append(argumentos, limite)

	return consulta, argumentos
}

// ModuloDe ruta devuelve el nombre del módulo a partir de la ruta HTTP.
func ModuloDe(ruta string) string {
	switch {
	case strings.HasPrefix(ruta, "/api/packages"):
		return "paquetes"
	case strings.HasPrefix(ruta, "/api/keys"):
		return "llaves"
	case strings.HasPrefix(ruta, "/api/issue"):
		return "issue"
	case strings.HasPrefix(ruta, "/api/hostname"):
		return "hostname"
	case strings.HasPrefix(ruta, "/api/users"):
		return "usuarios"
	case strings.HasPrefix(ruta, "/api/units"):
		return "systemd"
	case strings.HasPrefix(ruta, "/api/haproxy"):
		return "haproxy"
	case strings.HasPrefix(ruta, "/api/session"):
		return "sesion"
	case strings.HasPrefix(ruta, "/api/vms"):
		return "maquinas"
	default:
		return ""
	}
}
