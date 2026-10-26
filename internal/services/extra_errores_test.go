// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// contadorControladores evita registrar dos veces el mismo controlador
// cuando las pruebas se ejecutan con -count.
var contadorControladores atomic.Int64

// registerControladorUnico registra un controlador de prueba con un
// nombre irrepetible y lo devuelve.
func registerControladorUnico(t *testing.T, controlador driver.Driver) string {
	t.Helper()

	name := fmt.Sprintf("vmcons-prueba-%d", contadorControladores.Add(1))
	sql.Register(name, controlador)

	return name
}

// errorRows devuelve el error configurado al recorrer.
type errorRows struct{ err error }

// Columns anuncia las columnas de PRAGMA table_info.
func (f errorRows) Columns() []string {
	return []string{"cid", "name", "type", "notnull", "dflt_value", "pk"}
}

// Close devuelve el error configurado.
func (f errorRows) Close() error { return f.err }

// Next devuelve el error configurado.
func (f errorRows) Next([]driver.Value) error { return f.err }

// valueRows entrega una fila con valores que no encajan en el Scan.
type valueRows struct {
	entregada bool
}

// Columns anuncia las columnas de PRAGMA table_info.
func (f *valueRows) Columns() []string {
	return []string{"cid", "name", "type", "notnull", "dflt_value", "pk"}
}

// Close no hace nada.
func (f *valueRows) Close() error { return nil }

// Next entrega una fila con tipos incompatibles y luego termina.
func (f *valueRows) Next(destination []driver.Value) error {
	if f.entregada {
		return io.EOF
	}

	f.entregada = true

	for i := range destination {
		destination[i] = "texto"
	}

	return nil
}

// unclosedRows termina sin error pero falla al cerrarse.
type unclosedRows struct{}

// Columns anuncia las columnas de PRAGMA table_info.
func (unclosedRows) Columns() []string {
	return []string{"cid", "name", "type", "notnull", "dflt_value", "pk"}
}

// Close devuelve el error del cierre.
func (unclosedRows) Close() error { return errors.New("cierre") }

// Next termina el recorrido sin error.
func (unclosedRows) Next([]driver.Value) error { return io.EOF }

// foundRows entrega la columna buscada y falla al cerrarse.
type foundRows struct{}

// Columns anuncia las columnas de PRAGMA table_info.
func (foundRows) Columns() []string {
	return []string{"cid", "name", "type", "notnull", "dflt_value", "pk"}
}

// Close devuelve el error del cierre.
func (foundRows) Close() error { return errors.New("cierre") }

// Next entrega la columna buscada y luego termina.
func (foundRows) Next(destination []driver.Value) error {
	if destination[0] != nil {
		return io.EOF
	}

	destination[0] = int64(0)
	destination[1] = "descripcion"
	destination[2] = "TEXT"
	destination[3] = int64(0)
	destination[4] = nil
	destination[5] = int64(0)

	return nil
}

// rowsConnector entrega las filas indicadas.
type rowsConnector struct{ rows driver.Rows }

// Connect devuelve una conexión que solo responde consultas.
func (c rowsConnector) Connect(context.Context) (driver.Conn, error) {
	return rowsConnection(c), nil
}

// Driver no se usa.
func (c rowsConnector) Driver() driver.Driver { return nil }

// rowsConnection responde con las filas configuradas.
type rowsConnection struct{ rows driver.Rows }

// Prepare no está soportado.
func (rowsConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("no soportado")
}

// Close no hace nada.
func (rowsConnection) Close() error { return nil }

// Begin no está soportado.
func (rowsConnection) Begin() (driver.Tx, error) { return nil, errors.New("no soportado") }

// QueryContext devuelve las filas configuradas.
func (c rowsConnection) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return c.rows, nil
}

// testDriver entrega el conector registrado o su error.
type testDriver struct {
	connector driver.Connector
	err       error
}

// Open no se usa.
func (d *testDriver) Open(string) (driver.Conn, error) { return nil, errors.New("no soportado") }

// OpenConnector entrega el conector de prueba o el error configurado.
func (d *testDriver) OpenConnector(string) (driver.Connector, error) {
	if d.err != nil {
		return nil, d.err
	}

	return d.connector, nil
}

// TestExistsColumnaWithRowsDefectuosas distingue los errores reales de
// lectura de los del cierre, que solo se registran.
func TestExistsColumnaWithRowsDefectuosas(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		rows        func() driver.Rows
		wantsExists bool
		wantsError  bool
	}{
		{"error al recorrer", func() driver.Rows { return errorRows{err: errors.New("fallo")} }, false, true},
		{"error al leer", func() driver.Rows { return &valueRows{} }, false, true},
		{"error al cerrar se propaga", func() driver.Rows { return unclosedRows{} }, false, true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			name := registerControladorUnico(t, &testDriver{
				connector: rowsConnector{rows: testCase.rows()},
			})

			db, err := sql.Open(name, "")
			if err != nil {
				t.Fatalf("abrir la base de prueba: %v", err)
			}

			t.Cleanup(func() {
				if err := db.Close(); err != nil {
					t.Logf("cerrar la base de prueba: %v", err)
				}
			})

			exists, err := columnExists(db, "operaciones", "otra")
			if (err != nil) != testCase.wantsError {
				t.Fatalf("existeColumna() error = %v, want error=%v", err, testCase.wantsError)
			}

			if exists != testCase.wantsExists {
				t.Errorf("existeColumna() = %v, want %v", exists, testCase.wantsExists)
			}
		})
	}
}

// TestExistsColumnaEncuentraColumna cierra las filas tras el acierto.
func TestExistsColumnaEncuentraColumna(t *testing.T) {
	t.Parallel()

	name := registerControladorUnico(t, &testDriver{
		connector: rowsConnector{rows: foundRows{}},
	})

	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("abrir la base de prueba: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("cerrar la base de prueba: %v", err)
		}
	})

	exists, err := columnExists(db, "credenciales", "descripcion")
	if err != nil || !exists {
		t.Errorf("existeColumna() = %v, %v, want true, nil", exists, err)
	}
}

// TestNewStoreNotOpensBase cubre el fallo de apertura.
func TestNewStoreNotOpensBase(t *testing.T) {
	original := sqlDriver

	t.Cleanup(func() { sqlDriver = original })

	sqlDriver = registerControladorUnico(t, &testDriver{err: errors.New("sin base")})

	directory := t.TempDir()

	if _, err := NewStore(filepath.Join(directory, fileBDTests), filepath.Join(directory, "clave")); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo de apertura")
	}
}

// TestNewStoreAdvierteIfNotAjustaPermissions comprueba que un fallo al
// ajustar los permisos de la base no impide usarla.
func TestNewStoreAdvierteIfNotAjustaPermissions(t *testing.T) {
	original := adjustPermissions

	t.Cleanup(func() { adjustPermissions = original })

	adjustPermissions = func(string, os.FileMode) error { return errors.New("sin permisos") }

	directory := t.TempDir()

	store, err := NewStore(filepath.Join(directory, fileBDTests), filepath.Join(directory, "clave"))
	if err != nil {
		t.Fatalf("NuevoAlmacen() error = %v, want nil", err)
	}

	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("cerrar la base: %v", err)
		}
	})

	if _, err := NewConsole(store).IsConfigured(); err != nil {
		t.Errorf("Configurada() error = %v, want una base utilizable", err)
	}
}

// TestSchemaAntiguoNotMigrable comprueba que la migración de una base
// de solo lectura falla en lugar de continuar con un esquema incompleto.
func TestSchemaAntiguoNotMigrable(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	pathBD := filepath.Join(directory, fileBDTests)
	pathKey := filepath.Join(directory, "clave")

	store, err := NewStore(pathBD, pathKey)
	if err != nil {
		t.Fatalf("NuevoAlmacen() error = %v, want nil", err)
	}

	if _, err := store.db.ExecContext(t.Context(),
		`ALTER TABLE operaciones DROP COLUMN descripcion`); err != nil {
		t.Fatalf("quitar la columna: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("cerrar la base: %v", err)
	}

	db, err := sql.Open("sqlite", pathBD)
	if err != nil {
		t.Fatalf("abrir la base de solo lectura: %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("cerrar la base de prueba: %v", err)
		}
	})

	if _, err := db.ExecContext(t.Context(), "PRAGMA query_only = ON"); err != nil {
		t.Fatalf("marcar la base como solo lectura: %v", err)
	}

	if err := createSchema(db); err == nil {
		t.Error("crearEsquema() error = nil, want el fallo de la migración")
	}
}
