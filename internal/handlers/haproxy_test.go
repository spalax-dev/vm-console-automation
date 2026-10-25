// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"consola-vm/internal/services"
)

// stubHaproxy es un servicio de HAProxy falso con resultados
// preprogramados.
type stubHaproxy struct {
	lista      []services.Servidor
	resultado  services.Resultado
	err        error
	direccion  string
	puerto     int
	agregados  int
	eliminados int
}

// Listar devuelve la lista preprogramada.
func (s *stubHaproxy) Listar(context.Context) ([]services.Servidor, error) {
	return s.lista, s.err
}

// Agregar registra los argumentos y devuelve el resultado.
func (s *stubHaproxy) Agregar(_ context.Context, direccion string, puerto int) (services.Resultado, error) {
	s.direccion = direccion
	s.puerto = puerto
	s.agregados++

	return s.resultado, s.err
}

// Eliminar registra los argumentos y devuelve el resultado.
func (s *stubHaproxy) Eliminar(_ context.Context, direccion string, puerto int) (services.Resultado, error) {
	s.direccion = direccion
	s.puerto = puerto
	s.eliminados++

	return s.resultado, s.err
}

// TestServidoresListar cubre el listado del back-end.
func TestServidoresListar(t *testing.T) {
	t.Parallel()

	servicio := &stubHaproxy{lista: []services.Servidor{
		{Direccion: ipServidor, Puerto: 8080, Estado: "unknown"},
	}}

	rec := ejecutarCaso(t, NuevoHaproxy(servicio).Servidores, rutaServidores, casoPeticion{
		nombre: casoListar, metodo: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	var cuerpo listaBackend
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if len(cuerpo.Servidores) != 1 || cuerpo.Servidores[0].Puerto != 8080 {
		t.Errorf("servidores = %+v, want 192.168.1.61:8080", cuerpo.Servidores)
	}
}

// TestServidoresAgregar cubre el alta de un servidor.
func TestServidoresAgregar(t *testing.T) {
	t.Parallel()

	servicio := &stubHaproxy{resultado: services.Resultado{Mensaje: "Servidor agregado."}}

	rec := ejecutarCaso(t, NuevoHaproxy(servicio).Servidores, rutaServidores, casoPeticion{
		nombre: "agregar", metodo: http.MethodPost, cuerpo: cuerpoServidor,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf(msgCodigo, rec.Code, http.StatusCreated)
	}

	if got := rec.Header().Get(cabeceraLocation); got != "/api/haproxy/backend/servers/192.168.1.61:8080" {
		t.Errorf("Location = %q, want la ruta del servidor", got)
	}

	if servicio.agregados != 1 || servicio.puerto != 8080 {
		t.Errorf("Agregar() = %+v, want 192.168.1.61:8080", servicio)
	}
}

// TestServidoresDuplicado cubre el servidor que ya existe.
func TestServidoresDuplicado(t *testing.T) {
	t.Parallel()

	servicio := &stubHaproxy{err: services.ErrServidorExistente}

	rec := ejecutarCaso(t, NuevoHaproxy(servicio).Servidores, rutaServidores, casoPeticion{
		nombre: "duplicado", metodo: http.MethodPost, cuerpo: cuerpoServidor,
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf(msgCodigo, rec.Code, http.StatusConflict)
	}
}

// TestServidoresValidacion cubre la dirección y el puerto.
func TestServidoresValidacion(t *testing.T) {
	t.Parallel()

	handler := NuevoHaproxy(&stubHaproxy{}).Servidores
	casos := []casoPeticion{
		{nombre: "hostname", metodo: http.MethodPost, cuerpo: `{"direccion":"localhost","puerto":8080}`, codigo: http.StatusBadRequest, campo: campoDireccion},
		{nombre: "puerto fuera de rango", metodo: http.MethodPost, cuerpo: `{"direccion":"192.168.1.61","puerto":70000}`, codigo: http.StatusBadRequest, campo: campoPuerto},
		{nombre: casoCuerpoInvalido, metodo: http.MethodPost, cuerpo: `{`, codigo: http.StatusBadRequest},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodPut, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, rutaServidores, caso)
		comprobarCaso(t, rec, caso)
	}
}

// TestServidorEliminar cubre la eliminación de un servidor.
func TestServidorEliminar(t *testing.T) {
	t.Parallel()

	servicio := &stubHaproxy{}

	rec := ejecutarCasoConPath(t, NuevoHaproxy(servicio).Servidor, rutaServidor, campoServer, servidorPrueba, casoPeticion{
		nombre: casoEliminar, metodo: http.MethodDelete,
	})

	if rec.Code != http.StatusNoContent {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNoContent)
	}

	if servicio.eliminados != 1 || servicio.direccion != ipServidor || servicio.puerto != 8080 {
		t.Errorf("Eliminar() = %+v, want 192.168.1.61:8080", servicio)
	}
}

// TestServidorNoEncontrado cubre un servidor ausente.
func TestServidorNoEncontrado(t *testing.T) {
	t.Parallel()

	servicio := &stubHaproxy{err: services.ErrServidorNoEncontrado}

	rec := ejecutarCasoConPath(t, NuevoHaproxy(servicio).Servidor, rutaServidor, campoServer, servidorPrueba, casoPeticion{
		nombre: casoInexistente, metodo: http.MethodDelete,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNotFound)
	}
}

// TestServidorValidacion cubre el identificador de la ruta.
func TestServidorValidacion(t *testing.T) {
	t.Parallel()

	handler := NuevoHaproxy(&stubHaproxy{}).Servidor
	casos := []struct {
		casoPeticion
		servidor string
	}{
		{casoPeticion{nombre: "sin puerto", metodo: http.MethodDelete, codigo: http.StatusBadRequest, campo: campoServer}, ipServidor},
		{casoPeticion{nombre: "puerto cero", metodo: http.MethodDelete, codigo: http.StatusBadRequest, campo: campoServer}, "192.168.1.61:0"},
		{casoPeticion{nombre: casoMetodoNoPermitido, metodo: http.MethodPost, codigo: http.StatusMethodNotAllowed}, servidorPrueba},
	}

	for _, caso := range casos {
		rec := ejecutarCasoConPath(t, handler, rutaServidor, campoServer, caso.servidor, caso.casoPeticion)
		comprobarCaso(t, rec, caso.casoPeticion)
	}
}

// TestServidoresError cubre el fallo al agregar un servidor.
func TestServidoresError(t *testing.T) {
	t.Parallel()

	servicio := &stubHaproxy{err: errLectura}

	rec := ejecutarCaso(t, NuevoHaproxy(servicio).Servidores, rutaServidores, casoPeticion{
		nombre: "fallo de alta", metodo: http.MethodPost, cuerpo: cuerpoServidor,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestServidoresListarError cubre el fallo al listar el back-end.
func TestServidoresListarError(t *testing.T) {
	t.Parallel()

	rec := ejecutarCaso(t, NuevoHaproxy(&stubHaproxy{err: errLectura}).Servidores, rutaServidores, casoPeticion{
		nombre: casoListar, metodo: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestServidorError cubre el fallo al eliminar un servidor.
func TestServidorError(t *testing.T) {
	t.Parallel()

	rec := ejecutarCasoConPath(t, NuevoHaproxy(&stubHaproxy{err: errLectura}).Servidor, rutaServidor, campoServer, servidorPrueba, casoPeticion{
		nombre: casoEliminar, metodo: http.MethodDelete,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}
