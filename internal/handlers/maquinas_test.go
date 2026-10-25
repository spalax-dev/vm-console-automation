// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/services"
)

// Valores repetidos en las pruebas del inventario.
const (
	rutaMaquinas  = "/api/vms"
	maquinaDebian = "debian-1"
)

// stubMaquinas implementa ServicioMaquinas para las pruebas.
type stubMaquinas struct {
	maquinas    []services.MaquinaVirtual
	filtro      string
	err         error
	encendida   string
	errEncender error
}

// Listar devuelve el inventario preprogramado y registra el filtro.
func (s *stubMaquinas) Listar(_ context.Context, filtro string) ([]services.MaquinaVirtual, error) {
	s.filtro = filtro

	return s.maquinas, s.err
}

// Encender registra la máquina y devuelve el error preprogramado.
func (s *stubMaquinas) Encender(_ context.Context, identificador string) (services.MaquinaVirtual, error) {
	s.encendida = identificador

	if s.errEncender != nil {
		return services.MaquinaVirtual{}, s.errEncender
	}

	return services.MaquinaVirtual{Nombre: identificador, Estado: domain.StateRunning, Activa: true}, nil
}

// Vincular devuelve la primera máquina o el error preprogramado.
func (s *stubMaquinas) Vincular(_ context.Context, _ string) (services.MaquinaVirtual, error) {
	if s.err != nil {
		return services.MaquinaVirtual{}, s.err
	}

	return s.maquinas[0], nil
}

// TestMaquinasListaRespondeJSON comprueba el contrato de GET /api/vms.
func TestMaquinasListaRespondeJSON(t *testing.T) {
	t.Parallel()

	servicio := &stubMaquinas{maquinas: []services.MaquinaVirtual{
		{Nombre: maquinaDebian, UUID: "uuid-1", Estado: domain.StateRunning, Activa: true},
	}}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaMaquinas+"?q=deb", http.NoBody)
	rec := httptest.NewRecorder()

	NuevoMaquinas(servicio).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	if servicio.filtro != "deb" {
		t.Errorf("filtro = %q, want %q", servicio.filtro, "deb")
	}

	var body struct {
		Maquinas []map[string]any `json:"maquinas"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if len(body.Maquinas) != 1 {
		t.Fatalf("len(maquinas) = %d, want 1", len(body.Maquinas))
	}

	if body.Maquinas[0]["nombre"] != maquinaDebian {
		t.Errorf("nombre = %v, want %q", body.Maquinas[0]["nombre"], maquinaDebian)
	}

	if body.Maquinas[0]["activa"] != true {
		t.Errorf("activa = %v, want true", body.Maquinas[0]["activa"])
	}
}

// TestMaquinasRechazaMetodosNoGET comprueba que el inventario es de
// solo lectura.
func TestMaquinasRechazaMetodosNoGET(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, rutaMaquinas, http.NoBody)
	rec := httptest.NewRecorder()

	NuevoMaquinas(&stubMaquinas{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(msgCodigo, rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestMaquinasErrorDelHipervisorResponde500 comprueba que un fallo de
// VBoxManage no se confunde con una lista vacía.
func TestMaquinasErrorDelHipervisorResponde500(t *testing.T) {
	t.Parallel()

	servicio := &stubMaquinas{err: errors.New("VBoxManage no está disponible")}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaMaquinas, http.NoBody)
	rec := httptest.NewRecorder()

	NuevoMaquinas(servicio).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestSesionVinculaMaquina comprueba PUT /api/session.
func TestSesionVinculaMaquina(t *testing.T) {
	t.Parallel()

	vinculo := &stubVinculacion{maquina: services.MaquinaVirtual{Nombre: maquinaDebian, Estado: domain.StateRunning, Activa: true}}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, rutaSesion,
		strings.NewReader(`{"vm":"debian-1"}`))
	rec := httptest.NewRecorder()

	NuevoSesion(&stubDescubridor{}, ConVinculador(vinculo)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	if vinculo.vinculada != maquinaDebian {
		t.Errorf("vm vinculada = %q, want %q", vinculo.vinculada, maquinaDebian)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if body["nombre"] != maquinaDebian {
		t.Errorf("nombre = %v, want %q", body["nombre"], maquinaDebian)
	}
}

// TestSesionVinculaMaquinaInexistenteResponde404 comprueba el error de
// una máquina que no está en el inventario.
func TestSesionVinculaMaquinaInexistenteResponde404(t *testing.T) {
	t.Parallel()

	vinculo := &stubVinculacion{err: services.ErrMaquinaNoEncontrada}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, rutaSesion,
		strings.NewReader(`{"vm":"fantasma"}`))
	rec := httptest.NewRecorder()

	NuevoSesion(&stubDescubridor{}, ConVinculador(vinculo)).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNotFound)
	}
}

// TestSesionVinculaSinNombreResponde400 comprueba la validación del
// campo obligatorio.
func TestSesionVinculaSinNombreResponde400(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, rutaSesion,
		strings.NewReader(`{"vm":"  "}`))
	rec := httptest.NewRecorder()

	NuevoSesion(&stubDescubridor{}, ConVinculador(&stubVinculacion{})).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(msgCodigo, rec.Code, http.StatusBadRequest)
	}
}

// TestSesionVinculaCuerpoInvalidoResponde400 comprueba el rechazo de un
// JSON que no cumple el esquema.
func TestSesionVinculaCuerpoInvalidoResponde400(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, rutaSesion,
		strings.NewReader(`{"maquina":maquinaDebian}`))
	rec := httptest.NewRecorder()

	NuevoSesion(&stubDescubridor{}, ConVinculador(&stubVinculacion{})).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(msgCodigo, rec.Code, http.StatusBadRequest)
	}
}

// TestSesionSinVinculadorRechazaPUT evita aceptar una vinculación que
// no se puede aplicar.
func TestSesionSinVinculadorRechazaPUT(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, rutaSesion,
		strings.NewReader(`{"vm":maquinaDebian}`))
	rec := httptest.NewRecorder()

	NuevoSesion(&stubDescubridor{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(msgCodigo, rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestSesionPublicaLaMaquinaVinculada comprueba el campo vm de GET.
func TestSesionPublicaLaMaquinaVinculada(t *testing.T) {
	t.Parallel()

	desc := &stubDescubridor{info: services.InfoSesion{Conector: conectorSSH, Estado: domain.StateRunning}}
	vinculo := &stubVinculacion{activa: maquinaDebian}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaSesion, http.NoBody)
	rec := httptest.NewRecorder()

	NuevoSesion(desc, ConVinculador(vinculo)).ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if body["vm"] != maquinaDebian {
		t.Errorf("vm = %v, want %q", body["vm"], maquinaDebian)
	}
}

// stubVinculacion implementa VinculadorSesion.
type stubVinculacion struct {
	maquina   services.MaquinaVirtual
	vinculada string
	activa    string
	err       error
}

// Vincular registra el identificador y devuelve la máquina o el error.
func (s *stubVinculacion) Vincular(_ context.Context, id string) (services.MaquinaVirtual, error) {
	s.vinculada = id

	if s.err != nil {
		return services.MaquinaVirtual{}, s.err
	}

	return s.maquina, nil
}

// Activa devuelve la máquina vinculada.
func (s *stubVinculacion) Activa() string { return s.activa }

// TestMaquinasEnciendeRespondeLaMaquina comprueba POST
// /api/vms/{vm}/poweron.
func TestMaquinasEnciendeRespondeLaMaquina(t *testing.T) {
	t.Parallel()

	servicio := &stubMaquinas{}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, rutaMaquinas+"/debian-1/poweron", http.NoBody)
	req.SetPathValue("vm", "debian-1")

	rec := httptest.NewRecorder()

	NuevoMaquinas(servicio).Encender(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	if servicio.encendida != maquinaDebian {
		t.Errorf("encendida = %q, want %q", servicio.encendida, maquinaDebian)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if body["estado"] != "running" {
		t.Errorf("estado = %v, want %q", body["estado"], "running")
	}
}

// TestMaquinasEnciendeUnaMaquinaYaEncendidaResponde409 comprueba el
// conflicto de estado.
func TestMaquinasEnciendeUnaMaquinaYaEncendidaResponde409(t *testing.T) {
	t.Parallel()

	servicio := &stubMaquinas{errEncender: services.ErrMaquinaEncendida}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, rutaMaquinas+"/debian-1/poweron", http.NoBody)
	req.SetPathValue("vm", maquinaDebian)

	rec := httptest.NewRecorder()

	NuevoMaquinas(servicio).Encender(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf(msgCodigo, rec.Code, http.StatusConflict)
	}
}

// TestMaquinasEnciendeUnaMaquinaDesconocidaResponde404 comprueba el
// recurso inexistente.
func TestMaquinasEnciendeUnaMaquinaDesconocidaResponde404(t *testing.T) {
	t.Parallel()

	servicio := &stubMaquinas{errEncender: services.ErrMaquinaNoEncontrada}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, rutaMaquinas+"/fantasma/poweron", http.NoBody)
	req.SetPathValue("vm", "fantasma")

	rec := httptest.NewRecorder()

	NuevoMaquinas(servicio).Encender(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNotFound)
	}
}

// TestMaquinasEnciendeRechazaMetodosNoPOST comprueba el verbo.
func TestMaquinasEnciendeRechazaMetodosNoPOST(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaMaquinas+"/debian-1/poweron", http.NoBody)
	req.SetPathValue("vm", maquinaDebian)

	rec := httptest.NewRecorder()

	NuevoMaquinas(&stubMaquinas{}).Encender(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(msgCodigo, rec.Code, http.StatusMethodNotAllowed)
	}
}
