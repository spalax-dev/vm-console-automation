// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"consola-vm/internal/services"
)

// sessionCookieName es la cookie que identifica la sesión de la consola.
const sessionCookieName = "vmcons_sesion"

// authPrefix es el prefijo de las rutas que no exigen cookie.
const authPrefix = "/api/auth/"

// consoleUserPattern valida el usuario de la consola.
var consoleUserPattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]{2,31}$`)

// Límites de la contraseña de la consola.
const (
	minPasswordConsole = 8
	maxPasswordConsole = 128
)

// userContextKey es la clave del usuario autenticado en el contexto.
type userContextKey struct{}

// ConsoleService autentica a los usuarios de la consola.
type ConsoleService interface {
	// IsConfigured informa si ya existe una cuenta.
	IsConfigured() (bool, error)
	// Configure crea la primera cuenta.
	Configure(ctx context.Context, user, password string) error
	// Authenticate comprueba las credenciales.
	Authenticate(ctx context.Context, user, password string) (bool, error)
	// OpenSession emite la cookie de sesión.
	OpenSession(user string) (string, error)
	// Validate comprueba una cookie y devuelve el usuario.
	Validate(token string) (string, bool)
}

// Auth agrupa los recursos de autenticación de la consola.
type Auth struct {
	service ConsoleService
}

// NewAuth construye el handler de autenticación.
func NewAuth(service ConsoleService) *Auth {
	return &Auth{service: service}
}

// credentialsConsole es el cuerpo de acceso y de creación de cuenta.
type credentialsConsole struct {
	User     string `json:"usuario"`
	Password string `json:"password"`
}

// authState es el cuerpo de GET /api/auth/estado.
type authState struct {
	IsConfigured bool   `json:"configurada"`
	Autenticado  bool   `json:"autenticado"`
	User         string `json:"usuario,omitempty"`
}

// State informa si hay cuenta y si la petición trae una sesión válida.
func (a *Auth) State(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	configurada, err := a.service.IsConfigured()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:   "No fue posible consultar la cuenta de la consola.",
			Technical: err.Error(),
		})

		return
	}

	body := authState{IsConfigured: configurada}

	if user, ok := a.service.Validate(tokenFrom(r)); ok {
		body.Autenticado = true
		body.User = user
	}

	writeJSON(w, http.StatusOK, body)
}

// Configure crea la primera cuenta y deja la sesión iniciada.
func (a *Auth) Configure(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	credentials, ok := readCredentials(w, r)
	if !ok {
		return
	}

	err := a.service.Configure(r.Context(), credentials.User, credentials.Password)

	switch {
	case errors.Is(err, services.ErrConsoleConfigured):
		writeJSON(w, http.StatusConflict, errorResponse{
			Message:   "La consola ya tiene una cuenta; inicia sesión.",
			Technical: err.Error(),
		})

		return
	case errors.Is(err, services.ErrConsoleData):
		respondValidacion(w, err)

		return
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:   "No fue posible crear la cuenta de la consola.",
			Technical: err.Error(),
		})

		return
	}

	a.login(w, credentials.User, http.StatusCreated)
}

// Login inicia sesión con la cuenta existente.
func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	credentials, ok := readCredentials(w, r)
	if !ok {
		return
	}

	autenticado, err := a.service.Authenticate(r.Context(), credentials.User, credentials.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:   "No fue posible comprobar las credenciales.",
			Technical: err.Error(),
		})

		return
	}

	if !autenticado {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "Usuario o contraseña incorrectos.",
		})

		return
	}

	a.login(w, credentials.User, http.StatusOK)
}

// Logout vence la cookie de sesión.
func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- la consola se sirve por HTTP local.
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusNoContent)
}

// login emite la cookie y responde con el estado de la sesión.
func (a *Auth) login(w http.ResponseWriter, user string, code int) {
	token, err := a.service.OpenSession(user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:   "No fue posible abrir la sesión de la consola.",
			Technical: err.Error(),
		})

		return
	}

	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- la consola se sirve por HTTP local.
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(services.SessionLifetime.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, code, authState{
		IsConfigured: true,
		Autenticado:  true,
		User:         user,
	})
}

// readCredentials decodifica y valida el cuerpo de acceso. Responde el
// error y devuelve false cuando no es utilizable.
func readCredentials(w http.ResponseWriter, r *http.Request) (credentialsConsole, bool) {
	var credentials credentialsConsole

	if err := decodeJSON(w, r, &credentials); err != nil {
		respondValidacion(w, err)

		return credentialsConsole{}, false
	}

	credentials.User = strings.TrimSpace(credentials.User)

	if !consoleUserPattern.MatchString(credentials.User) {
		respondValidacion(w, invalidField(userField,
			"debe tener entre 3 y 32 caracteres y empezar por minúscula o guion bajo"))

		return credentialsConsole{}, false
	}

	if len(credentials.Password) < minPasswordConsole || len(credentials.Password) > maxPasswordConsole {
		respondValidacion(w, invalidField(passwordField,
			"debe tener entre 8 y 128 caracteres"))

		return credentialsConsole{}, false
	}

	return credentials, true
}

// WithAuthentication exige la cookie de sesión en la API, salvo en las rutas
// de acceso. Los archivos de la interfaz se sirven sin cookie: la propia
// aplicación muestra el formulario cuando `estado` no está autenticado.
func WithAuthentication(next http.Handler, service ConsoleService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requiresSession(r.URL.Path) {
			next.ServeHTTP(w, r)

			return
		}

		user, ok := service.Validate(tokenFrom(r))
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{
				Message: "La sesión de la consola no está iniciada.",
			})

			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey{}, user)))
	})
}

// requiresSession informa si la ruta requiere cookie de sesión.
func requiresSession(path string) bool {
	return strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, authPrefix)
}

// tokenFrom devuelve el valor de la cookie de sesión.
func tokenFrom(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}

	return cookie.Value
}

// UserFrom devuelve el usuario autenticado del contexto.
func UserFrom(ctx context.Context) string {
	user, ok := ctx.Value(userContextKey{}).(string)
	if !ok {
		return ""
	}

	return user
}
