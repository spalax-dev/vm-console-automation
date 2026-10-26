// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"database/sql"
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
	registrarControlador(t, "historial-error", conectorDeFilas{filas: filasConError{err: errors.New("fallo")}})
	registrarControlador(t, "historial-sin-cierre", conectorDeFilas{filas: filasQueNoCierran{}})
	registrarControlador(t, "historial-con-valores", conectorDeFilas{filas: &filasConValores{}})

	casos := []struct {
		nombre      string
		controlador string
	}{
		{"recorrido", "historial-error"},
		{"cierre", "historial-sin-cierre"},
		{"lectura", "historial-con-valores"},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			db, err := sql.Open(caso.controlador, "")
			if err != nil {
				t.Fatalf("abrir la base de prueba: %v", err)
			}

			t.Cleanup(func() {
				if err := db.Close(); err != nil {
					t.Logf("cerrar la base de prueba: %v", err)
				}
			})

			if _, err := (&Historial{db: db}).Listar(context.Background(), nil); err == nil && caso.nombre != "cierre" {
				t.Errorf("Listar() error = nil, want el fallo de las filas (%s)", caso.nombre)
			}
		})
	}
}
