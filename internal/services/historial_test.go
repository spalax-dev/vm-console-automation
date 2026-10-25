// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"testing"
)

// nuevoHistorialDePrueba arma el historial sobre una base temporal.
func nuevoHistorialDePrueba(t *testing.T) *Historial {
	t.Helper()

	almacen, _, _ := nuevaBaseDePrueba(t)

	return NuevoHistorial(almacen)
}

// registrarEjemplos inserta tres operaciones con fechas conocidas.
func registrarEjemplos(t *testing.T, historial *Historial) {
	t.Helper()

	operaciones := []Operacion{
		{FechaHora: "2026-10-25T10:00:00Z", VM: nombreDebian, Modulo: "paquetes", Operacion: "POST /api/packages", Resultado: "ok"},
		{FechaHora: "2026-10-25T12:30:00Z", VM: nombreDebian, Modulo: "usuarios", Operacion: "POST /api/users", Resultado: "ok"},
		{FechaHora: "2026-10-26T09:15:00Z", VM: "ubuntu-lab", Modulo: moduloSystemd, Operacion: "POST /api/units/{unit}/restarts", Resultado: "error", Detalle: "HTTP 500"},
	}

	for i := range operaciones {
		if err := historial.Registrar(context.Background(), &operaciones[i]); err != nil {
			t.Fatalf("Registrar() error = %v", err)
		}
	}
}

// TestHistorialListarOrdenaYCunta comprueba el orden descendente.
func TestHistorialListarOrdenaYCunta(t *testing.T) {
	t.Parallel()

	historial := nuevoHistorialDePrueba(t)
	registrarEjemplos(t, historial)

	operaciones, err := historial.Listar(context.Background(), &FiltroHistorial{})
	if err != nil {
		t.Fatalf("Listar() error = %v", err)
	}

	if len(operaciones) != 3 {
		t.Fatalf("Listar() devolvió %d, want 3", len(operaciones))
	}

	if operaciones[0].VM != "ubuntu-lab" {
		t.Errorf("primera operación = %+v, want la más reciente", operaciones[0])
	}
}

// TestHistorialFiltros comprueba texto, VM, módulo, fecha, hora y límite.
func TestHistorialFiltros(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre    string
		filtro    FiltroHistorial
		esperados int
	}{
		{"texto", FiltroHistorial{Texto: "users"}, 1},
		{"vm", FiltroHistorial{VM: nombreDebian}, 2},
		{"módulo", FiltroHistorial{Modulo: moduloSystemd}, 1},
		{"fecha", FiltroHistorial{Desde: "2026-10-26", Hasta: "2026-10-26"}, 1},
		{"hora", FiltroHistorial{HoraDesde: "12:00", HoraHasta: "13:00"}, 1},
		{"límite", FiltroHistorial{Limite: 1}, 1},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			historial := nuevoHistorialDePrueba(t)
			registrarEjemplos(t, historial)

			operaciones, err := historial.Listar(context.Background(), &caso.filtro)
			if err != nil {
				t.Fatalf("Listar() error = %v", err)
			}

			if len(operaciones) != caso.esperados {
				t.Errorf("Listar(%+v) devolvió %d, want %d", caso.filtro, len(operaciones), caso.esperados)
			}
		})
	}
}

// TestModuloDeRuta comprueba la traducción de rutas a módulos.
func TestModuloDeRuta(t *testing.T) {
	t.Parallel()

	casos := map[string]string{
		"/api/packages/nano":                   "paquetes",
		"/api/keys/root":                       "llaves",
		"/api/users/jorge/key":                 "usuarios",
		"/api/units/catalogo.service/restarts": moduloSystemd,
		"/api/haproxy/backend/servers":         "haproxy",
		"/api/session":                         "",
	}

	for ruta, esperado := range casos {
		if obtenido := ModuloDe(ruta); obtenido != esperado {
			t.Errorf("ModuloDe(%q) = %q, want %q", ruta, obtenido, esperado)
		}
	}
}
