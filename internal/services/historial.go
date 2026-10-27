// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Operation es una entrada del historial de la consola.
type Operation struct {
	// ID es el identificador incremental.
	ID int64
	// DateHour es la marca temporal en RFC 3339 (UTC).
	DateHour string
	// VM es la máquina sobre la que se actuó.
	VM string
	// Module identifica el módulo (paquetes, usuarios, systemd, etc.).
	Module string
	// Operation es el método y la ruta ejecutados.
	Operation string
	// Description es el resumen legible de la operación.
	Description string
	// Outcome es «ok» o «error».
	Outcome string
	// Detail conserva información técnica breve.
	Detail string
}

// HistoryFilter reúne los filtros de la consulta del historial.
type HistoryFilter struct {
	// Text busca en la máquina, el módulo, la operación, la descripción y
	// el detalle.
	Text string
	// VM filtra por el nombre de la máquina.
	VM string
	// Module filtra por el módulo que originó la operación.
	Module string
	// From acota por la fecha inicial (AAAA-MM-DD), inclusive.
	From string
	// To acota por la fecha final (AAAA-MM-DD), inclusive.
	To string
	// HourFrom acota por la hora inicial del día (HH:MM), inclusive.
	HourFrom string
	// HourTo acota por la hora final del día (HH:MM), inclusive.
	HourTo string
	// Limit es el máximo de filas; cero usa el límite por omisión.
	Limit int
}

// limitHistory es el número máximo de filas por consulta.
const limitHistory = 200

// History registra las operaciones y las consulta desde SQLite.
type History struct {
	db *sql.DB
}

// NewHistory construye el historial sobre la base de la consola.
func NewHistory(store *Store) *History {
	return &History{db: store.db}
}

// Register agrega una operación al historial.
func (h *History) Register(ctx context.Context, operation *Operation) error {
	if operation.DateHour == "" {
		operation.DateHour = time.Now().UTC().Format(time.RFC3339)
	}

	_, err := h.db.ExecContext(ctx, `
		INSERT INTO operaciones (fecha_hora, vm, modulo, operacion, descripcion, resultado, detalle)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		operation.DateHour, operation.VM, operation.Module, operation.Operation,
		operation.Description, operation.Outcome, operation.Detail,
	)
	if err != nil {
		return fmt.Errorf("registrar la operación: %w", err)
	}

	return nil
}

// List devuelve las operaciones que cumplen el filtro, de la más
// reciente a la más antigua.
func (h *History) List(ctx context.Context, filter *HistoryFilter) ([]Operation, error) {
	if filter == nil {
		filter = &HistoryFilter{}
	}

	consulta, arguments := consultaHistory(filter)

	rows, err := h.db.QueryContext(ctx, consulta, arguments...)
	if err != nil {
		return nil, fmt.Errorf("consultar el historial: %w", err)
	}

	defer func() {
		// El cierre tras recorrer las filas no aporta información: el error
		// del recorrido ya se devolvió arriba.
		_ = rows.Close() //nolint:errcheck // el cierre no cambia el resultado.
	}()

	operations := []Operation{}

	for rows.Next() {
		var operation Operation

		if err := rows.Scan(
			&operation.ID, &operation.DateHour, &operation.VM, &operation.Module,
			&operation.Operation, &operation.Description, &operation.Outcome, &operation.Detail,
		); err != nil {
			return nil, fmt.Errorf("leer una operación: %w", err)
		}

		operations = append(operations, operation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer el historial: %w", err)
	}

	return operations, nil
}

// consultaHistory arma el SQL y los argumentos según los filtros.
func consultaHistory(filter *HistoryFilter) (consulta string, arguments []any) {
	consulta = `SELECT id, fecha_hora, vm, modulo, operacion, descripcion, resultado, detalle FROM operaciones WHERE 1 = 1`
	arguments = []any{}

	if filter.Text != "" {
		consulta += ` AND (vm LIKE ? OR modulo LIKE ? OR operacion LIKE ? OR descripcion LIKE ? OR detalle LIKE ?)`
		like := "%" + filter.Text + "%"
		arguments = append(arguments, like, like, like, like, like)
	}

	if filter.VM != "" {
		consulta += ` AND vm = ?`

		arguments = append(arguments, filter.VM)
	}

	if filter.Module != "" {
		consulta += ` AND modulo = ?`

		arguments = append(arguments, filter.Module)
	}

	if filter.From != "" {
		consulta += ` AND substr(fecha_hora, 1, 10) >= ?`

		arguments = append(arguments, filter.From)
	}

	if filter.To != "" {
		consulta += ` AND substr(fecha_hora, 1, 10) <= ?`

		arguments = append(arguments, filter.To)
	}

	if filter.HourFrom != "" {
		consulta += ` AND substr(fecha_hora, 12, 5) >= ?`

		arguments = append(arguments, filter.HourFrom)
	}

	if filter.HourTo != "" {
		consulta += ` AND substr(fecha_hora, 12, 5) <= ?`

		arguments = append(arguments, filter.HourTo)
	}

	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = limitHistory
	}

	consulta += ` ORDER BY id DESC LIMIT ?`

	arguments = append(arguments, limit)

	return consulta, arguments
}

// ModuleOf ruta devuelve el nombre del módulo a partir de la ruta HTTP.
func ModuleOf(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/packages"):
		return "paquetes"
	case strings.HasPrefix(path, "/api/keys"):
		return "llaves"
	case strings.HasPrefix(path, pathIssue):
		return "issue"
	case strings.HasPrefix(path, "/api/hostname"):
		return "hostname"
	case strings.HasPrefix(path, "/api/users"):
		return "usuarios"
	case strings.HasPrefix(path, "/api/units"):
		return "systemd"
	case strings.HasPrefix(path, "/api/haproxy"):
		return serviceHAProxy
	case strings.HasPrefix(path, "/api/session"):
		return "sesion"
	case strings.HasPrefix(path, "/api/vms"):
		return "maquinas"
	default:
		return ""
	}
}
