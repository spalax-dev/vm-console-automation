// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"net/http"
	"testing"
)

// TestUsuarios cubre la colección de usuarios.
func TestUsuarios(t *testing.T) {
	t.Parallel()

	handler := NuevoUsuarios().Coleccion
	casos := []casoPeticion{
		{nombre: casoListar, metodo: http.MethodGet, codigo: http.StatusNotImplemented},
		{nombre: "crear válido", metodo: http.MethodPost, cuerpo: `{"nombre":"jorge","sudo":true,"llaves":true}`, codigo: http.StatusNotImplemented},
		{nombre: "crear con mayúsculas", metodo: http.MethodPost, cuerpo: `{"nombre":"José","sudo":false}`, codigo: http.StatusBadRequest, campo: campoNombre},
		{nombre: "crear root", metodo: http.MethodPost, cuerpo: `{"nombre":"root","sudo":true}`, codigo: http.StatusBadRequest, campo: campoNombre},
		{nombre: casoCuerpoInvalido, metodo: http.MethodPost, cuerpo: `{`, codigo: http.StatusBadRequest},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodPut, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, "/api/users", caso)
		comprobarCaso(t, rec, caso)
	}
}

// TestUsuario cubre la consulta y eliminación de un usuario por su
// nombre, con la opción de borrar sus archivos.
func TestUsuario(t *testing.T) {
	t.Parallel()

	handler := NuevoUsuarios().Usuario
	casos := []struct {
		casoPeticion
		ruta   string
		nombre string
	}{
		{casoPeticion{nombre: casoConsultar, metodo: http.MethodGet, codigo: http.StatusNotImplemented}, rutaUsuarioJorge, usuarioJorge},
		{casoPeticion{nombre: casoEliminar, metodo: http.MethodDelete, codigo: http.StatusNotImplemented}, rutaUsuarioJorge, usuarioJorge},
		{casoPeticion{nombre: "eliminar con archivos", metodo: http.MethodDelete, codigo: http.StatusNotImplemented}, rutaUsuarioJorge + "?files=true", usuarioJorge},
		{casoPeticion{nombre: "archivos inválido", metodo: http.MethodDelete, codigo: http.StatusBadRequest, campo: campoFiles}, rutaUsuarioJorge + "?files=quizá", usuarioJorge},
		{casoPeticion{nombre: casoNombreInvalido, metodo: http.MethodGet, codigo: http.StatusBadRequest, campo: campoName}, "/api/users/Jorge", usuarioJorgeMayus},
		{casoPeticion{nombre: casoMetodoNoPermitido, metodo: http.MethodPost, codigo: http.StatusMethodNotAllowed}, rutaUsuarioJorge, usuarioJorge},
	}

	for _, caso := range casos {
		rec := ejecutarCasoConPath(t, handler, caso.ruta, campoName, caso.nombre, caso.casoPeticion)
		comprobarCaso(t, rec, caso.casoPeticion)
	}
}

// TestUsuarioLlave cubre la descarga de la llave privada de un usuario.
func TestUsuarioLlave(t *testing.T) {
	t.Parallel()

	handler := NuevoUsuarios().Llave
	casos := []struct {
		casoPeticion
		nombre string
	}{
		{casoPeticion{nombre: "descargar", metodo: http.MethodGet, codigo: http.StatusNotImplemented}, usuarioJorge},
		{casoPeticion{nombre: casoNombreInvalido, metodo: http.MethodGet, codigo: http.StatusBadRequest, campo: campoName}, usuarioJorgeMayus},
		{casoPeticion{nombre: casoMetodoNoPermitido, metodo: http.MethodPost, codigo: http.StatusMethodNotAllowed}, usuarioJorge},
	}

	for _, caso := range casos {
		rec := ejecutarCasoConPath(t, handler, "/api/users/jorge/key", "name", caso.nombre, caso.casoPeticion)
		comprobarCaso(t, rec, caso.casoPeticion)
	}
}
