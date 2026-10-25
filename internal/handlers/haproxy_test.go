// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"net/http"
	"testing"
)

// TestServidores cubre el listado y el alta de servidores del back-end.
func TestServidores(t *testing.T) {
	t.Parallel()

	handler := NuevoHaproxy().Servidores
	casos := []casoPeticion{
		{nombre: casoListar, metodo: http.MethodGet, codigo: http.StatusNotImplemented},
		{nombre: "agregar válido", metodo: http.MethodPost, cuerpo: `{"direccion":"192.168.1.61","puerto":8080}`, codigo: http.StatusNotImplemented},
		{nombre: "agregar con hostname", metodo: http.MethodPost, cuerpo: `{"direccion":"localhost","puerto":8080}`, codigo: http.StatusBadRequest, campo: campoDireccion},
		{nombre: "agregar con puerto fuera de rango", metodo: http.MethodPost, cuerpo: `{"direccion":"192.168.1.61","puerto":70000}`, codigo: http.StatusBadRequest, campo: campoPuerto},
		{nombre: casoCuerpoInvalido, metodo: http.MethodPost, cuerpo: `{`, codigo: http.StatusBadRequest},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodPut, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, "/api/haproxy/backend/servers", caso)
		comprobarCaso(t, rec, caso)
	}
}

// TestServidor cubre la eliminación de un servidor identificado como
// «direccion:puerto».
func TestServidor(t *testing.T) {
	t.Parallel()

	handler := NuevoHaproxy().Servidor
	casos := []struct {
		casoPeticion
		servidor string
	}{
		{casoPeticion{nombre: casoEliminar, metodo: http.MethodDelete, codigo: http.StatusNotImplemented}, servidorPrueba},
		{casoPeticion{nombre: "sin puerto", metodo: http.MethodDelete, codigo: http.StatusBadRequest, campo: campoServer}, "192.168.1.61"},
		{casoPeticion{nombre: "puerto cero", metodo: http.MethodDelete, codigo: http.StatusBadRequest, campo: campoServer}, "192.168.1.61:0"},
		{casoPeticion{nombre: casoMetodoNoPermitido, metodo: http.MethodPost, codigo: http.StatusMethodNotAllowed}, servidorPrueba},
	}

	for _, caso := range casos {
		rec := ejecutarCasoConPath(t, handler, "/api/haproxy/backend/servers/x", "server", caso.servidor, caso.casoPeticion)
		comprobarCaso(t, rec, caso.casoPeticion)
	}
}
