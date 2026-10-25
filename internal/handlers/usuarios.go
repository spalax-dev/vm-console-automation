// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import "net/http"

// Usuarios agrupa los recursos de gestión de usuarios (numeral 3.2).
type Usuarios struct{}

// NuevoUsuarios construye el handler de gestión de usuarios.
func NuevoUsuarios() *Usuarios {
	return &Usuarios{}
}

// solicitudUsuario es el cuerpo de POST /api/users.
type solicitudUsuario struct {
	Nombre string `json:"nombre"`
	Sudo   bool   `json:"sudo"`
	Llaves bool   `json:"llaves"`
}

// Coleccion atiende GET y POST /api/users: lista los usuarios o crea
// uno nuevo.
func (h *Usuarios) Coleccion(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	if r.Method == http.MethodGet {
		noImplementado(w, "listar los usuarios")

		return
	}

	var solicitud solicitudUsuario
	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarUsuario(campoNombre, solicitud.Nombre); err != nil {
		responderValidacion(w, err)

		return
	}

	if solicitud.Nombre == "root" {
		responderValidacion(w, campoInvalido(campoNombre, "ya existe; use otro nombre de usuario"))

		return
	}

	noImplementado(w, "crear el usuario "+solicitud.Nombre)
}

// Usuario atiende GET y DELETE /api/users/{name}: consulta o elimina la
// cuenta indicada en la ruta.
func (h *Usuarios) Usuario(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet, http.MethodDelete) {
		return
	}

	nombre := r.PathValue(campoName)
	if err := validarUsuario(campoName, nombre); err != nil {
		responderValidacion(w, err)

		return
	}

	if r.Method == http.MethodGet {
		noImplementado(w, "consultar el usuario "+nombre)

		return
	}

	if _, err := booleanoQuery(r, campoFiles); err != nil {
		responderValidacion(w, err)

		return
	}

	noImplementado(w, "eliminar el usuario "+nombre)
}

// Llave atiende GET /api/users/{name}/key: entrega la llave privada
// generada para el usuario indicado.
func (h *Usuarios) Llave(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet) {
		return
	}

	nombre := r.PathValue(campoName)
	if err := validarUsuario(campoName, nombre); err != nil {
		responderValidacion(w, err)

		return
	}

	noImplementado(w, "descargar la llave privada de "+nombre)
}
