// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
)

// TestHistorialFiltraPorMaquina comprueba el filtro por VM del historial.
func TestHistorialFiltraPorMaquina(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	historial := NuevoHistorial(almacen)
	ctx := context.Background()

	operaciones := []Operacion{
		{VM: "debian-1", Modulo: "paquetes", Operacion: "POST /api/packages", Resultado: "ok"},
		{VM: "debian-2", Modulo: "usuarios", Operacion: "POST /api/users", Resultado: "ok"},
	}

	for i := range operaciones {
		if err := historial.Registrar(ctx, &operaciones[i]); err != nil {
			t.Fatalf("Registrar() error = %v, want nil", err)
		}
	}

	if operaciones[0].FechaHora == "" {
		t.Error("Registrar() no fijó la fecha y la hora")
	}

	lista, err := historial.Listar(ctx, &FiltroHistorial{VM: "debian-1"})
	if err != nil {
		t.Fatalf("Listar() error = %v, want nil", err)
	}

	if len(lista) != 1 || lista[0].VM != "debian-1" {
		t.Errorf("Listar() = %+v, want una operación de debian-1", lista)
	}
}

// TestHistorialSinFiltro lista todas las operaciones.
func TestHistorialSinFiltro(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	historial := NuevoHistorial(almacen)
	ctx := context.Background()

	if err := historial.Registrar(ctx, &Operacion{VM: "debian-1", Modulo: "paquetes"}); err != nil {
		t.Fatalf("Registrar() error = %v, want nil", err)
	}

	lista, err := historial.Listar(ctx, nil)
	if err != nil {
		t.Fatalf("Listar() error = %v, want nil", err)
	}

	if len(lista) != 1 {
		t.Errorf("Listar() = %d operaciones, want 1", len(lista))
	}
}

// TestHistorialConBaseCerrada propaga los errores de SQLite.
func TestHistorialConBaseCerrada(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	historial := NuevoHistorial(almacen)
	ctx := context.Background()

	if err := almacen.db.Close(); err != nil {
		t.Fatalf("cerrar la base: %v", err)
	}

	if err := historial.Registrar(ctx, &Operacion{}); err == nil {
		t.Error("Registrar() error = nil, want el fallo de SQLite")
	}

	if _, err := historial.Listar(ctx, nil); err == nil {
		t.Error("Listar() error = nil, want el fallo de SQLite")
	}
}

// TestModuloDeRutas reconoce todos los módulos y descarta el resto.
func TestModuloDeRutas(t *testing.T) {
	t.Parallel()

	casos := map[string]string{
		"/api/issue":       "issue",
		"/api/hostname":    "hostname",
		"/api/packages":    "paquetes",
		"/api/session":     "sesion",
		"/api/desconocido": "",
		"/otra/ruta":       "",
	}

	for ruta, esperado := range casos {
		if modulo := ModuloDe(ruta); modulo != esperado {
			t.Errorf("ModuloDe(%q) = %q, want %q", ruta, modulo, esperado)
		}
	}
}

// TestHistorialConFilasRotas cubre los errores al recorrer las filas.
func TestHistorialConFilasRotas(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre      string
		filas       func() driver.Rows
		quiereError bool
	}{
		{"recorrido", func() driver.Rows { return filasConError{err: errors.New("fallo")} }, true},
		{"cierre", func() driver.Rows { return filasQueNoCierran{} }, true},
		{"lectura", func() driver.Rows { return &filasConValores{} }, true},
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

			operaciones, err := (&Historial{db: db}).Listar(context.Background(), nil)
			if (err != nil) != caso.quiereError {
				t.Fatalf("Listar() error = %v, want error=%v", err, caso.quiereError)
			}

			if !caso.quiereError && len(operaciones) != 0 {
				t.Errorf("Listar() devolvió %d operaciones, want 0", len(operaciones))
			}
		})
	}
}
