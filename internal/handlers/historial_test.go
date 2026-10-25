// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"consola-vm/internal/services"
)

// stubHistorial es un historial falso con resultados preprogramados.
type stubHistorial struct {
	lista       []services.Operacion
	err         error
	filtro      services.FiltroHistorial
	registradas []services.Operacion
}

// Listar devuelve la lista preprogramada.
func (s *stubHistorial) Listar(_ context.Context, filtro *services.FiltroHistorial) ([]services.Operacion, error) {
	s.filtro = *filtro

	return s.lista, s.err
}

// Registrar acumula las operaciones registradas.
func (s *stubHistorial) Registrar(_ context.Context, operacion *services.Operacion) error {
	s.registradas = append(s.registradas, *operacion)

	return s.err
}

// TestHistorialListar cubre la consulta y los filtros.
func TestHistorialListar(t *testing.T) {
	t.Parallel()

	servicio := &stubHistorial{lista: []services.Operacion{
		{ID: 1, FechaHora: "2026-10-25T10:00:00Z", VM: maquinaDebian, Modulo: "paquetes", Operacion: "POST /api/packages", Resultado: "ok"},
	}}

	ruta := "/api/historial?q=nano&vm=debian-1&modulo=paquetes&desde=2026-10-25&hasta=2026-10-26&hora_desde=08:00&hora_hasta=18:00&limite=50"

	rec := ejecutarCaso(t, NuevoHistorial(servicio).ServeHTTP, ruta, casoPeticion{
		nombre: casoListar, metodo: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	if servicio.filtro.Texto != "nano" || servicio.filtro.VM != maquinaDebian || servicio.filtro.Limite != 50 {
		t.Errorf("filtro = %+v, want los parámetros de la query", servicio.filtro)
	}

	var cuerpo listaHistorial
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if len(cuerpo.Operaciones) != 1 || cuerpo.Operaciones[0].VM != maquinaDebian {
		t.Errorf("operaciones = %+v, want debian-1", cuerpo.Operaciones)
	}
}

// TestHistorialValidacion cubre fechas, horas, límite y método.
func TestHistorialValidacion(t *testing.T) {
	t.Parallel()

	handler := NuevoHistorial(&stubHistorial{}).ServeHTTP

	casos := []struct {
		casoPeticion
		ruta string
	}{
		{casoPeticion{nombre: "fecha inválida", metodo: http.MethodGet, codigo: http.StatusBadRequest, campo: "desde"}, "/api/historial?desde=25-10-2026"},
		{casoPeticion{nombre: "hora inválida", metodo: http.MethodGet, codigo: http.StatusBadRequest, campo: "hora_hasta"}, "/api/historial?hora_hasta=25:00"},
		{casoPeticion{nombre: "límite inválido", metodo: http.MethodGet, codigo: http.StatusBadRequest, campo: "limite"}, "/api/historial?limite=0"},
		{casoPeticion{nombre: casoMetodoNoPermitido, metodo: http.MethodPost, codigo: http.StatusMethodNotAllowed}, "/api/historial"},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, caso.ruta, caso.casoPeticion)
		comprobarCaso(t, rec, caso.casoPeticion)
	}
}

// TestConHistorialRegistraMutaciones comprueba que solo se registran las
// operaciones que no son GET.
func TestConHistorialRegistraMutaciones(t *testing.T) {
	t.Parallel()

	servicio := &stubHistorial{}
	handler := ConHistorial(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
		servicio,
		func() string { return maquinaDebian },
	)

	post := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/packages", http.NoBody)
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, post)

	if len(servicio.registradas) != 1 {
		t.Fatalf("Registrar() se llamó %d veces, want 1", len(servicio.registradas))
	}

	if servicio.registradas[0].Resultado != "ok" || servicio.registradas[0].VM != maquinaDebian {
		t.Errorf("operación = %+v, want ok en debian-1", servicio.registradas[0])
	}

	get := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/packages", http.NoBody)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, get)

	if len(servicio.registradas) != 1 {
		t.Errorf("una consulta GET se registró en el historial")
	}
}
