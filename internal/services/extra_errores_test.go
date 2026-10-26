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

// registrarControladorUnico registra un controlador de prueba con un
// nombre irrepetible y lo devuelve.
func registrarControladorUnico(t *testing.T, controlador driver.Driver) string {
	t.Helper()

	nombre := fmt.Sprintf("vmcons-prueba-%d", contadorControladores.Add(1))
	sql.Register(nombre, controlador)

	return nombre
}

// filasConError devuelve el error configurado al recorrer.
type filasConError struct{ err error }

// Columns anuncia las columnas de PRAGMA table_info.
func (f filasConError) Columns() []string {
	return []string{"cid", "name", "type", "notnull", "dflt_value", "pk"}
}

// Close devuelve el error configurado.
func (f filasConError) Close() error { return f.err }

// Next devuelve el error configurado.
func (f filasConError) Next([]driver.Value) error { return f.err }

// filasConValores entrega una fila con valores que no encajan en el Scan.
type filasConValores struct {
	entregada bool
}

// Columns anuncia las columnas de PRAGMA table_info.
func (f *filasConValores) Columns() []string {
	return []string{"cid", "name", "type", "notnull", "dflt_value", "pk"}
}

// Close no hace nada.
func (f *filasConValores) Close() error { return nil }

// Next entrega una fila con tipos incompatibles y luego termina.
func (f *filasConValores) Next(destino []driver.Value) error {
	if f.entregada {
		return io.EOF
	}

	f.entregada = true

	for i := range destino {
		destino[i] = "texto"
	}

	return nil
}

// filasQueNoCierran termina sin error pero falla al cerrarse.
type filasQueNoCierran struct{}

// Columns anuncia las columnas de PRAGMA table_info.
func (filasQueNoCierran) Columns() []string {
	return []string{"cid", "name", "type", "notnull", "dflt_value", "pk"}
}

// Close devuelve el error del cierre.
func (filasQueNoCierran) Close() error { return errors.New("cierre") }

// Next termina el recorrido sin error.
func (filasQueNoCierran) Next([]driver.Value) error { return io.EOF }

// filasEncontradas entrega la columna buscada y falla al cerrarse.
type filasEncontradas struct{}

// Columns anuncia las columnas de PRAGMA table_info.
func (filasEncontradas) Columns() []string {
	return []string{"cid", "name", "type", "notnull", "dflt_value", "pk"}
}

// Close devuelve el error del cierre.
func (filasEncontradas) Close() error { return errors.New("cierre") }

// Next entrega la columna buscada y luego termina.
func (filasEncontradas) Next(destino []driver.Value) error {
	if destino[0] != nil {
		return io.EOF
	}

	destino[0] = int64(0)
	destino[1] = "descripcion"
	destino[2] = "TEXT"
	destino[3] = int64(0)
	destino[4] = nil
	destino[5] = int64(0)

	return nil
}

// conectorDeFilas entrega las filas indicadas.
type conectorDeFilas struct{ filas driver.Rows }

// Connect devuelve una conexión que solo responde consultas.
func (c conectorDeFilas) Connect(context.Context) (driver.Conn, error) {
	return conexionDeFilas(c), nil
}

// Driver no se usa.
func (c conectorDeFilas) Driver() driver.Driver { return nil }

// conexionDeFilas responde con las filas configuradas.
type conexionDeFilas struct{ filas driver.Rows }

// Prepare no está soportado.
func (conexionDeFilas) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("no soportado")
}

// Close no hace nada.
func (conexionDeFilas) Close() error { return nil }

// Begin no está soportado.
func (conexionDeFilas) Begin() (driver.Tx, error) { return nil, errors.New("no soportado") }

// QueryContext devuelve las filas configuradas.
func (c conexionDeFilas) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return c.filas, nil
}

// driverConector entrega el conector registrado o su error.
type driverConector struct {
	conector driver.Connector
	err      error
}

// Open no se usa.
func (d *driverConector) Open(string) (driver.Conn, error) { return nil, errors.New("no soportado") }

// OpenConnector entrega el conector de prueba o el error configurado.
func (d *driverConector) OpenConnector(string) (driver.Connector, error) {
	if d.err != nil {
		return nil, d.err
	}

	return d.conector, nil
}

// TestExisteColumnaConFilasDefectuosas distingue los errores reales de
// lectura de los del cierre, que solo se registran.
func TestExisteColumnaConFilasDefectuosas(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre       string
		filas        func() driver.Rows
		quiereExiste bool
		quiereError  bool
	}{
		{"error al recorrer", func() driver.Rows { return filasConError{err: errors.New("fallo")} }, false, true},
		{"error al leer", func() driver.Rows { return &filasConValores{} }, false, true},
		{"error al cerrar se propaga", func() driver.Rows { return filasQueNoCierran{} }, false, true},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			nombre := registrarControladorUnico(t, &driverConector{
				conector: conectorDeFilas{filas: caso.filas()},
			})

			db, err := sql.Open(nombre, "")
			if err != nil {
				t.Fatalf("abrir la base de prueba: %v", err)
			}

			t.Cleanup(func() {
				if err := db.Close(); err != nil {
					t.Logf("cerrar la base de prueba: %v", err)
				}
			})

			existe, err := existeColumna(db, "operaciones", "otra")
			if (err != nil) != caso.quiereError {
				t.Fatalf("existeColumna() error = %v, want error=%v", err, caso.quiereError)
			}

			if existe != caso.quiereExiste {
				t.Errorf("existeColumna() = %v, want %v", existe, caso.quiereExiste)
			}
		})
	}
}

// TestExisteColumnaEncuentraLaColumna cierra las filas tras el acierto.
func TestExisteColumnaEncuentraLaColumna(t *testing.T) {
	t.Parallel()

	nombre := registrarControladorUnico(t, &driverConector{
		conector: conectorDeFilas{filas: filasEncontradas{}},
	})

	db, err := sql.Open(nombre, "")
	if err != nil {
		t.Fatalf("abrir la base de prueba: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("cerrar la base de prueba: %v", err)
		}
	})

	existe, err := existeColumna(db, "credenciales", "descripcion")
	if err != nil || !existe {
		t.Errorf("existeColumna() = %v, %v, want true, nil", existe, err)
	}
}

// TestNuevoAlmacenNoAbreLaBase cubre el fallo de apertura.
func TestNuevoAlmacenNoAbreLaBase(t *testing.T) {
	original := controladorSQL

	t.Cleanup(func() { controladorSQL = original })

	controladorSQL = registrarControladorUnico(t, &driverConector{err: errors.New("sin base")})

	directorio := t.TempDir()

	if _, err := NuevoAlmacen(filepath.Join(directorio, archivoBDPrueba), filepath.Join(directorio, "clave")); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo de apertura")
	}
}

// TestNuevoAlmacenAdvierteSiNoAjustaPermisos comprueba que un fallo al
// ajustar los permisos de la base no impide usarla.
func TestNuevoAlmacenAdvierteSiNoAjustaPermisos(t *testing.T) {
	original := ajustarPermisos

	t.Cleanup(func() { ajustarPermisos = original })

	ajustarPermisos = func(string, os.FileMode) error { return errors.New("sin permisos") }

	directorio := t.TempDir()

	almacen, err := NuevoAlmacen(filepath.Join(directorio, archivoBDPrueba), filepath.Join(directorio, "clave"))
	if err != nil {
		t.Fatalf("NuevoAlmacen() error = %v, want nil", err)
	}

	t.Cleanup(func() {
		if err := almacen.Close(); err != nil {
			t.Errorf("cerrar la base: %v", err)
		}
	})

	if _, err := NuevoConsola(almacen).Configurada(); err != nil {
		t.Errorf("Configurada() error = %v, want una base utilizable", err)
	}
}

// TestEsquemaAntiguoNoMigrable comprueba que la migración de una base
// de solo lectura falla en lugar de continuar con un esquema incompleto.
func TestEsquemaAntiguoNoMigrable(t *testing.T) {
	t.Parallel()

	directorio := t.TempDir()
	rutaBD := filepath.Join(directorio, archivoBDPrueba)
	rutaClave := filepath.Join(directorio, "clave")

	almacen, err := NuevoAlmacen(rutaBD, rutaClave)
	if err != nil {
		t.Fatalf("NuevoAlmacen() error = %v, want nil", err)
	}

	if _, err := almacen.db.ExecContext(t.Context(),
		`ALTER TABLE operaciones DROP COLUMN descripcion`); err != nil {
		t.Fatalf("quitar la columna: %v", err)
	}

	if err := almacen.Close(); err != nil {
		t.Fatalf("cerrar la base: %v", err)
	}

	db, err := sql.Open("sqlite", rutaBD)
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

	if err := crearEsquema(db); err == nil {
		t.Error("crearEsquema() error = nil, want el fallo de la migración")
	}
}
