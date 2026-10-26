// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"fmt"
	"testing"
)

// msgListarError es el formato de error de Listar.
const msgListarError = "Listar() error = %v"

// moduloPaquetes es el nombre del módulo de paquetes.
const moduloPaquetes = "paquetes"

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
		{FechaHora: "2026-10-25T10:00:00Z", VM: nombreDebian, Modulo: moduloPaquetes, Operacion: "POST /api/packages", Descripcion: descripcionNano, Resultado: "ok"},
		{FechaHora: "2026-10-25T12:30:00Z", VM: nombreDebian, Modulo: "usuarios", Operacion: "POST /api/users", Descripcion: "Crear el usuario «jorge»", Resultado: "ok"},
		{FechaHora: "2026-10-26T09:15:00Z", VM: "ubuntu-lab", Modulo: moduloSystemd, Operacion: "POST /api/units/{unit}/restarts", Descripcion: "Reiniciar la unidad «catalogo.service»", Resultado: "error", Detalle: "HTTP 500"},
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
		t.Fatalf(msgListarError, err)
	}

	if len(operaciones) != 3 {
		t.Fatalf("Listar() devolvió %d, want 3", len(operaciones))
	}

	if operaciones[0].VM != "ubuntu-lab" {
		t.Errorf("primera operación = %+v, want la más reciente", operaciones[0])
	}

	if operaciones[0].Descripcion != "Reiniciar la unidad «catalogo.service»" {
		t.Errorf("Descripcion = %q, want el resumen guardado", operaciones[0].Descripcion)
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
		{"descripción", FiltroHistorial{Texto: "nano"}, 1},
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
				t.Fatalf(msgListarError, err)
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
		"/api/packages/nano":                   moduloPaquetes,
		"/api/keys/root":                       "llaves",
		"/api/users/jorge/key":                 "usuarios",
		"/api/units/catalogo.service/restarts": moduloSystemd,
		"/api/haproxy/backend/servers":         "haproxy",
		rutaSesion:                             "sesion",
		"/api/vms/ubuntu-lab/poweron":          "maquinas",
	}

	for ruta, esperado := range casos {
		if obtenido := ModuloDe(ruta); obtenido != esperado {
			t.Errorf("ModuloDe(%q) = %q, want %q", ruta, obtenido, esperado)
		}
	}
}

// TestHistorialListarLimite comprueba que el servicio acota el número de
// filas y usa el valor por omisión cuando el límite no es utilizable.
func TestHistorialListarLimite(t *testing.T) {
	t.Parallel()

	historial := nuevoHistorialDePrueba(t)

	for i := 0; i < limiteHistorial+5; i++ {
		if err := historial.Registrar(context.Background(), &Operacion{
			VM:        fmt.Sprintf("vm-%d", i),
			Modulo:    moduloPaquetes,
			Operacion: "POST /api/packages",
			Resultado: "ok",
		}); err != nil {
			t.Fatalf("Registrar() error = %v, want nil", err)
		}
	}

	casos := []struct {
		nombre    string
		limite    int
		esperados int
	}{
		{"cero usa el valor por omisión", 0, limiteHistorial},
		{"negativo usa el valor por omisión", -1, limiteHistorial},
		{"excesivo usa el valor por omisión", 5000, limiteHistorial},
		{"uno respeta el límite", 1, 1},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			operaciones, err := historial.Listar(context.Background(), &FiltroHistorial{Limite: caso.limite})
			if err != nil {
				t.Fatalf(msgListarError, err)
			}

			if len(operaciones) != caso.esperados {
				t.Errorf("Listar(Limite: %d) devolvió %d, want %d", caso.limite, len(operaciones), caso.esperados)
			}
		})
	}
}
