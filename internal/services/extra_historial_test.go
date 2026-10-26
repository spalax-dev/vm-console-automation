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

// TestHistoryFiltraByMachine comprueba el filtro por VM del historial.
func TestHistoryFiltraByMachine(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	history := NewHistory(store)
	ctx := context.Background()

	operations := []Operation{
		{VM: "vm-1", Module: "paquetes", Operation: "POST /api/packages", Outcome: "ok"},
		{VM: "vm-2", Module: "usuarios", Operation: "POST /api/users", Outcome: "ok"},
	}

	for i := range operations {
		if err := history.Register(ctx, &operations[i]); err != nil {
			t.Fatalf("Registrar() error = %v, want nil", err)
		}
	}

	if operations[0].DateHour == "" {
		t.Error("Registrar() no fijó la fecha y la hora")
	}

	lists, err := history.List(ctx, &HistoryFilter{VM: "vm-1"})
	if err != nil {
		t.Fatalf("Listar() error = %v, want nil", err)
	}

	if len(lists) != 1 || lists[0].VM != "vm-1" {
		t.Errorf("Listar() = %+v, want una operación de vm-1", lists)
	}
}

// TestHistoryWithoutFilter lista todas las operaciones.
func TestHistoryWithoutFilter(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	history := NewHistory(store)
	ctx := context.Background()

	if err := history.Register(ctx, &Operation{VM: "vm-1", Module: "paquetes"}); err != nil {
		t.Fatalf("Registrar() error = %v, want nil", err)
	}

	lists, err := history.List(ctx, nil)
	if err != nil {
		t.Fatalf("Listar() error = %v, want nil", err)
	}

	if len(lists) != 1 {
		t.Errorf("Listar() = %d operaciones, want 1", len(lists))
	}
}

// TestHistoryWithBaseClosed propaga los errores de SQLite.
func TestHistoryWithBaseClosed(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	history := NewHistory(store)
	ctx := context.Background()

	if err := store.db.Close(); err != nil {
		t.Fatalf("cerrar la base: %v", err)
	}

	if err := history.Register(ctx, &Operation{}); err == nil {
		t.Error("Registrar() error = nil, want el fallo de SQLite")
	}

	if _, err := history.List(ctx, nil); err == nil {
		t.Error("Listar() error = nil, want el fallo de SQLite")
	}
}

// TestModuleOfRutas reconoce todos los módulos y descarta el resto.
func TestModuleOfRutas(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"/api/issue":       "issue",
		"/api/hostname":    "hostname",
		"/api/packages":    "paquetes",
		"/api/session":     "sesion",
		"/api/desconocido": "",
		"/otra/ruta":       "",
	}

	for path, expected := range testCases {
		if module := ModuleOf(path); module != expected {
			t.Errorf("ModuloDe(%q) = %q, want %q", path, module, expected)
		}
	}
}

// TestHistoryWithRowsRotas cubre los errores al recorrer las filas.
func TestHistoryWithRowsRotas(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		rows       func() driver.Rows
		wantsError bool
	}{
		{"recorrido", func() driver.Rows { return errorRows{err: errors.New("fallo")} }, true},
		{"cierre", func() driver.Rows { return unclosedRows{} }, true},
		{"lectura", func() driver.Rows { return &valueRows{} }, true},
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

			operations, err := (&History{db: db}).List(context.Background(), nil)
			if (err != nil) != testCase.wantsError {
				t.Fatalf("Listar() error = %v, want error=%v", err, testCase.wantsError)
			}

			if !testCase.wantsError && len(operations) != 0 {
				t.Errorf("Listar() devolvió %d operaciones, want 0", len(operations))
			}
		})
	}
}
