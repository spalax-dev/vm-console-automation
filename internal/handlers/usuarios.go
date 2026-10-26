// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"consola-vm/internal/services"
)

// ServicioUsuarios ejecuta las operaciones sobre las cuentas locales.
type ServicioUsuarios interface {
	// Listar devuelve las cuentas regulares de la máquina.
	Listar(ctx context.Context) ([]services.Usuario, error)
	// Obtener devuelve una cuenta.
	Obtener(ctx context.Context, nombre string) (services.Usuario, error)
	// Crear crea la cuenta con sudo y llaves opcionales.
	Crear(ctx context.Context, nombre string, sudo, llaves bool) (services.Resultado, error)
	// Eliminar elimina la cuenta y, si se pide, sus archivos.
	Eliminar(ctx context.Context, nombre string, archivos bool) (services.Resultado, error)
	// DescargarLlave devuelve la llave privada de la cuenta.
	DescargarLlave(nombre string) (archivo string, contenido []byte, err error)
}

// Usuarios agrupa los recursos de gestión de usuarios (numeral 3.2).
type Usuarios struct {
	servicio ServicioUsuarios
}

// NuevoUsuarios construye el handler de gestión de usuarios.
func NuevoUsuarios(servicio ServicioUsuarios) *Usuarios {
	return &Usuarios{servicio: servicio}
}

// solicitudUsuario es el cuerpo de POST /api/users.
type solicitudUsuario struct {
	Nombre string `json:"nombre"`
	Sudo   bool   `json:"sudo"`
	Llaves bool   `json:"llaves"`
}

// usuarioRespuesta es una cuenta en la respuesta JSON.
type usuarioRespuesta struct {
	Nombre string `json:"nombre"`
	Sudo   bool   `json:"sudo"`
	Llaves bool   `json:"llaves"`
	UID    int    `json:"uid,omitempty"`
	Home   string `json:"home,omitempty"`
}

// listaUsuarios es el cuerpo de GET /api/users.
type listaUsuarios struct {
	Usuarios []usuarioRespuesta `json:"usuarios"`
}

// Coleccion atiende GET y POST /api/users: lista los usuarios o crea
// uno nuevo.
func (h *Usuarios) Coleccion(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	if r.Method == http.MethodGet {
		h.listar(w, r)

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

	resultado, err := h.servicio.Crear(r.Context(), solicitud.Nombre, solicitud.Sudo, solicitud.Llaves)
	if errors.Is(err, services.ErrUsuarioExistente) {
		escribirJSON(w, http.StatusConflict,
			respuestaDeError(resultado, "El usuario «"+solicitud.Nombre+"» ya existe.", err))

		return
	}

	if err != nil {
		responderErrorOperacion(w, resultado, err)

		return
	}

	w.Header().Set(cabeceraLocation, "/api/users/"+solicitud.Nombre)
	escribirResultado(w, http.StatusCreated, resultado)
}

// listar responde la colección de usuarios.
func (h *Usuarios) listar(w http.ResponseWriter, r *http.Request) {
	usuarios, err := h.servicio.Listar(r.Context())
	if err != nil {
		responderErrorOperacion(w, services.Resultado{}, err)

		return
	}

	respuesta := listaUsuarios{Usuarios: make([]usuarioRespuesta, 0, len(usuarios))}
	for _, usuario := range usuarios {
		respuesta.Usuarios = append(respuesta.Usuarios, usuarioRespuesta{
			Nombre: usuario.Nombre,
			Sudo:   usuario.Sudo,
			Llaves: usuario.Llaves,
			UID:    usuario.UID,
			Home:   usuario.Home,
		})
	}

	escribirJSON(w, http.StatusOK, respuesta)
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

	if r.Method == http.MethodDelete {
		h.eliminar(w, r, nombre)

		return
	}

	usuario, err := h.servicio.Obtener(r.Context(), nombre)
	if errors.Is(err, services.ErrUsuarioNoEncontrado) {
		escribirJSON(w, http.StatusNotFound, respuestaError{
			Mensaje: "El usuario «" + nombre + "» no existe.",
			Tecnico: err.Error(),
		})

		return
	}

	if err != nil {
		responderErrorOperacion(w, services.Resultado{}, err)

		return
	}

	escribirJSON(w, http.StatusOK, usuarioRespuesta{
		Nombre: usuario.Nombre,
		Sudo:   usuario.Sudo,
		Llaves: usuario.Llaves,
		UID:    usuario.UID,
		Home:   usuario.Home,
	})
}

// eliminar valida la opción de archivos y elimina la cuenta.
func (h *Usuarios) eliminar(w http.ResponseWriter, r *http.Request, nombre string) {
	archivos, err := booleanoQuery(r, campoFiles)
	if err != nil {
		responderValidacion(w, err)

		return
	}

	resultado, err := h.servicio.Eliminar(r.Context(), nombre, archivos)
	if errors.Is(err, services.ErrUsuarioNoEncontrado) {
		escribirJSON(w, http.StatusNotFound,
			respuestaDeError(resultado, "El usuario «"+nombre+"» no existe.", err))

		return
	}

	if err != nil {
		responderErrorOperacion(w, resultado, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Llave atiende GET /api/users/{name}/key: entrega la llave privada
// generada para el usuario.
func (h *Usuarios) Llave(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet) {
		return
	}

	nombre := r.PathValue(campoName)
	if err := validarUsuario(campoName, nombre); err != nil {
		responderValidacion(w, err)

		return
	}

	archivo, contenido, err := h.servicio.DescargarLlave(nombre)
	if errors.Is(err, services.ErrLlaveUsuarioNoEncontrada) {
		escribirJSON(w, http.StatusNotFound, respuestaError{
			Mensaje: "No hay una llave generada para el usuario «" + nombre + "».",
			Tecnico: err.Error(),
		})

		return
	}

	if err != nil {
		slog.Error("descargar la llave del usuario", claveError, err)
		escribirJSON(w, http.StatusInternalServerError, respuestaError{
			Mensaje: "No fue posible leer la llave del usuario.",
			Tecnico: err.Error(),
		})

		return
	}

	w.Header().Set(cabeceraContentType, "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+archivo+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)

	// #nosec G705 -- es la descarga binaria de una llave, no HTML.
	if _, err := w.Write(contenido); err != nil {
		slog.Error("escribir la llave del usuario", claveError, err)
	}
}
