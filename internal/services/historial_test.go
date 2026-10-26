// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"fmt"
	"testing"
)

// msgListError es el formato de error de List.
const msgListError = "Listar() error = %v"

// modulePackages es el nombre del módulo de paquetes.
const modulePackages = "paquetes"

// newHistoryOfTests arma el historial sobre una base temporal.
func newHistoryOfTests(t *testing.T) *History {
	t.Helper()

	store, _, _ := newTestDB(t)

	return NewHistory(store)
}

// registerEjemplos inserta tres operaciones con fechas conocidas.
func registerEjemplos(t *testing.T, history *History) {
	t.Helper()

	operations := []Operation{
		{DateHour: "2026-10-25T10:00:00Z", VM: testMachineName, Module: modulePackages, Operation: "POST /api/packages", Description: descriptionNano, Outcome: "ok"},
		{DateHour: "2026-10-25T12:30:00Z", VM: testMachineName, Module: "usuarios", Operation: "POST /api/users", Description: "Crear el usuario «jorge»", Outcome: "ok"},
		{DateHour: "2026-10-26T09:15:00Z", VM: "ubuntu-lab", Module: moduleSystemd, Operation: "POST /api/units/{unit}/restarts", Description: "Reiniciar la unidad «catalogo.service»", Outcome: "error", Detail: "HTTP 500"},
	}

	for i := range operations {
		if err := history.Register(context.Background(), &operations[i]); err != nil {
			t.Fatalf("Registrar() error = %v", err)
		}
	}
}

// TestHistoryListOrdenaYCunta comprueba el orden descendente.
func TestHistoryListOrdenaYCunta(t *testing.T) {
	t.Parallel()

	history := newHistoryOfTests(t)
	registerEjemplos(t, history)

	operations, err := history.List(context.Background(), &HistoryFilter{})
	if err != nil {
		t.Fatalf(msgListError, err)
	}

	if len(operations) != 3 {
		t.Fatalf("Listar() devolvió %d, want 3", len(operations))
	}

	if operations[0].VM != "ubuntu-lab" {
		t.Errorf("primera operación = %+v, want la más reciente", operations[0])
	}

	if operations[0].Description != "Reiniciar la unidad «catalogo.service»" {
		t.Errorf("Descripcion = %q, want el resumen guardado", operations[0].Description)
	}
}

// TestHistoryFilters comprueba texto, VM, módulo, fecha, hora y límite.
func TestHistoryFilters(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		filter   HistoryFilter
		expected int
	}{
		{"texto", HistoryFilter{Text: "users"}, 1},
		{"descripción", HistoryFilter{Text: "nano"}, 1},
		{"vm", HistoryFilter{VM: testMachineName}, 2},
		{"módulo", HistoryFilter{Module: moduleSystemd}, 1},
		{"fecha", HistoryFilter{From: "2026-10-26", To: "2026-10-26"}, 1},
		{"hora", HistoryFilter{HourFrom: "12:00", HourTo: "13:00"}, 1},
		{"límite", HistoryFilter{Limit: 1}, 1},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			history := newHistoryOfTests(t)
			registerEjemplos(t, history)

			operations, err := history.List(context.Background(), &testCase.filter)
			if err != nil {
				t.Fatalf(msgListError, err)
			}

			if len(operations) != testCase.expected {
				t.Errorf("Listar(%+v) devolvió %d, want %d", testCase.filter, len(operations), testCase.expected)
			}
		})
	}
}

// TestModuleOfPath comprueba la traducción de rutas a módulos.
func TestModuleOfPath(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"/api/packages/nano":                   modulePackages,
		"/api/keys/root":                       "llaves",
		"/api/users/jorge/key":                 "usuarios",
		"/api/units/catalogo.service/restarts": moduleSystemd,
		"/api/haproxy/backend/servers":         "haproxy",
		pathSession:                            "sesion",
		"/api/vms/ubuntu-lab/poweron":          "maquinas",
	}

	for path, expected := range testCases {
		if obtained := ModuleOf(path); obtained != expected {
			t.Errorf("ModuloDe(%q) = %q, want %q", path, obtained, expected)
		}
	}
}

// TestHistoryListLimit comprueba que el servicio acota el número de
// filas y usa el valor por omisión cuando el límite no es utilizable.
func TestHistoryListLimit(t *testing.T) {
	t.Parallel()

	history := newHistoryOfTests(t)

	for i := 0; i < limitHistory+5; i++ {
		if err := history.Register(context.Background(), &Operation{
			VM:        fmt.Sprintf("vm-%d", i),
			Module:    modulePackages,
			Operation: "POST /api/packages",
			Outcome:   "ok",
		}); err != nil {
			t.Fatalf("Registrar() error = %v, want nil", err)
		}
	}

	testCases := []struct {
		name     string
		limit    int
		expected int
	}{
		{"cero usa el valor por omisión", 0, limitHistory},
		{"negativo usa el valor por omisión", -1, limitHistory},
		{"excesivo usa el valor por omisión", 5000, limitHistory},
		{"uno respeta el límite", 1, 1},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			operations, err := history.List(context.Background(), &HistoryFilter{Limit: testCase.limit})
			if err != nil {
				t.Fatalf(msgListError, err)
			}

			if len(operations) != testCase.expected {
				t.Errorf("Listar(Limite: %d) devolvió %d, want %d", testCase.limit, len(operations), testCase.expected)
			}
		})
	}
}
