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

// UsersService ejecuta las operaciones sobre las cuentas locales.
type UsersService interface {
	// List devuelve las cuentas regulares de la máquina.
	List(ctx context.Context) ([]services.User, error)
	// Get devuelve una cuenta.
	Get(ctx context.Context, name string) (services.User, error)
	// Create crea la cuenta con sudo y llaves opcionales.
	Create(ctx context.Context, name string, sudo, keys bool) (services.Outcome, error)
	// Delete elimina la cuenta y, si se pide, sus archivos.
	Delete(ctx context.Context, name string, files bool) (services.Outcome, error)
	// DownloadKey devuelve la llave privada de la cuenta.
	DownloadKey(name string) (file string, content []byte, err error)
}

// Users agrupa los recursos de gestión de usuarios (numeral 3.2).
type Users struct {
	service UsersService
}

// NewUsers construye el handler de gestión de usuarios.
func NewUsers(service UsersService) *Users {
	return &Users{service: service}
}

// userRequest es el cuerpo de POST /api/users.
type userRequest struct {
	Name string `json:"nombre"`
	Sudo bool   `json:"sudo"`
	Keys bool   `json:"llaves"`
}

// userResponse es una cuenta en la respuesta JSON.
type userResponse struct {
	Name string `json:"nombre"`
	Sudo bool   `json:"sudo"`
	Keys bool   `json:"llaves"`
	UID  int    `json:"uid,omitempty"`
	Home string `json:"home,omitempty"`
}

// userList es el cuerpo de GET /api/users.
type userList struct {
	Users []userResponse `json:"usuarios"`
}

// Collection atiende GET y POST /api/users: lista los usuarios o crea
// uno nuevo.
func (h *Users) Collection(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	if r.Method == http.MethodGet {
		h.list(w, r)

		return
	}

	var request userRequest
	if err := decodeJSON(w, r, &request); err != nil {
		respondValidacion(w, err)

		return
	}

	if err := validateUser(nameField, request.Name); err != nil {
		respondValidacion(w, err)

		return
	}

	if request.Name == "root" {
		respondValidacion(w, invalidField(nameField, "ya existe; use otro nombre de usuario"))

		return
	}

	result, err := h.service.Create(r.Context(), request.Name, request.Sudo, request.Keys)
	if errors.Is(err, services.ErrUserExists) {
		writeJSON(w, http.StatusConflict,
			responseFromError(result, "El usuario «"+request.Name+"» ya existe.", err))

		return
	}

	if err != nil {
		respondErrorOperation(w, result, err)

		return
	}

	w.Header().Set(headerLocation, "/api/users/"+request.Name)
	writeResult(w, http.StatusCreated, result)
}

// list responde la colección de usuarios.
func (h *Users) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.List(r.Context())
	if err != nil {
		respondErrorOperation(w, services.Outcome{}, err)

		return
	}

	response := userList{Users: make([]userResponse, 0, len(users))}
	for _, user := range users {
		response.Users = append(response.Users, userResponse{
			Name: user.Name,
			Sudo: user.Sudo,
			Keys: user.Keys,
			UID:  user.UID,
			Home: user.Home,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// User atiende GET y DELETE /api/users/{name}: consulta o elimina la
// cuenta indicada en la ruta.
func (h *Users) User(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodDelete) {
		return
	}

	name := r.PathValue(nameParam)
	if err := validateUser(nameParam, name); err != nil {
		respondValidacion(w, err)

		return
	}

	if r.Method == http.MethodDelete {
		h.delete(w, r, name)

		return
	}

	user, err := h.service.Get(r.Context(), name)
	if errors.Is(err, services.ErrUserNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{
			Message:   "El usuario «" + name + "» no existe.",
			Technical: err.Error(),
		})

		return
	}

	if err != nil {
		respondErrorOperation(w, services.Outcome{}, err)

		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		Name: user.Name,
		Sudo: user.Sudo,
		Keys: user.Keys,
		UID:  user.UID,
		Home: user.Home,
	})
}

// delete valida la opción de archivos y elimina la cuenta.
func (h *Users) delete(w http.ResponseWriter, r *http.Request, name string) {
	files, err := booleanoQuery(r, filesParam)
	if err != nil {
		respondValidacion(w, err)

		return
	}

	result, err := h.service.Delete(r.Context(), name, files)
	if errors.Is(err, services.ErrUserNotFound) {
		writeJSON(w, http.StatusNotFound,
			responseFromError(result, "El usuario «"+name+"» no existe.", err))

		return
	}

	if err != nil {
		respondErrorOperation(w, result, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Key atiende GET /api/users/{name}/key: entrega la llave privada
// generada para el usuario.
func (h *Users) Key(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	name := r.PathValue(nameParam)
	if err := validateUser(nameParam, name); err != nil {
		respondValidacion(w, err)

		return
	}

	file, content, err := h.service.DownloadKey(name)
	if errors.Is(err, services.ErrUserKeyNotFound) {
		writeJSON(w, http.StatusNotFound, errorResponse{
			Message:   "No hay una llave generada para el usuario «" + name + "».",
			Technical: err.Error(),
		})

		return
	}

	if err != nil {
		slog.Error("descargar la llave del usuario", errorKey, err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:   "No fue posible leer la llave del usuario.",
			Technical: err.Error(),
		})

		return
	}

	w.Header().Set(headerContentType, "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+file+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)

	// #nosec G705 -- es la descarga binaria de una llave, no HTML.
	if _, err := w.Write(content); err != nil {
		slog.Error("escribir la llave del usuario", errorKey, err)
	}
}
