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

// nombreCookieSesion es la cookie que identifica la sesión de la consola.
const nombreCookieSesion = "vmcons_sesion"

// prefijoAcceso es el prefijo de las rutas que no exigen cookie.
const prefijoAcceso = "/api/auth/"

// reUsuarioConsola valida el usuario de la consola.
var reUsuarioConsola = regexp.MustCompile(`^[a-z_][a-z0-9_-]{2,31}$`)

// Límites de la contraseña de la consola.
const (
	minPasswordConsola = 8
	maxPasswordConsola = 128
)

// claveUsuario es la clave del usuario autenticado en el contexto.
type claveUsuario struct{}

// ServicioConsola autentica a los usuarios de la consola.
type ServicioConsola interface {
	// Configurada informa si ya existe una cuenta.
	Configurada() (bool, error)
	// Configurar crea la primera cuenta.
	Configurar(ctx context.Context, usuario, password string) error
	// Autenticar comprueba las credenciales.
	Autenticar(ctx context.Context, usuario, password string) (bool, error)
	// AbrirSesion emite la cookie de sesión.
	AbrirSesion(usuario string) (string, error)
	// Validar comprueba una cookie y devuelve el usuario.
	Validar(token string) (string, bool)
}

// Auth agrupa los recursos de autenticación de la consola.
type Auth struct {
	servicio ServicioConsola
}

// NuevoAuth construye el handler de autenticación.
func NuevoAuth(servicio ServicioConsola) *Auth {
	return &Auth{servicio: servicio}
}

// credencialesConsola es el cuerpo de acceso y de creación de cuenta.
type credencialesConsola struct {
	Usuario  string `json:"usuario"`
	Password string `json:"password"`
}

// estadoAutenticacion es el cuerpo de GET /api/auth/estado.
type estadoAutenticacion struct {
	Configurada bool   `json:"configurada"`
	Autenticado bool   `json:"autenticado"`
	Usuario     string `json:"usuario,omitempty"`
}

// Estado informa si hay cuenta y si la petición trae una sesión válida.
func (a *Auth) Estado(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet) {
		return
	}

	configurada, err := a.servicio.Configurada()
	if err != nil {
		escribirJSON(w, http.StatusInternalServerError, respuestaError{
			Mensaje: "No fue posible consultar la cuenta de la consola.",
			Tecnico: err.Error(),
		})

		return
	}

	cuerpo := estadoAutenticacion{Configurada: configurada}

	if usuario, ok := a.servicio.Validar(tokenDe(r)); ok {
		cuerpo.Autenticado = true
		cuerpo.Usuario = usuario
	}

	escribirJSON(w, http.StatusOK, cuerpo)
}

// Configurar crea la primera cuenta y deja la sesión iniciada.
func (a *Auth) Configurar(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodPost) {
		return
	}

	credenciales, ok := leerCredenciales(w, r)
	if !ok {
		return
	}

	err := a.servicio.Configurar(r.Context(), credenciales.Usuario, credenciales.Password)

	switch {
	case errors.Is(err, services.ErrConsolaConfigurada):
		escribirJSON(w, http.StatusConflict, respuestaError{
			Mensaje: "La consola ya tiene una cuenta; inicia sesión.",
			Tecnico: err.Error(),
		})

		return
	case errors.Is(err, services.ErrDatosConsola):
		responderValidacion(w, err)

		return
	case err != nil:
		escribirJSON(w, http.StatusInternalServerError, respuestaError{
			Mensaje: "No fue posible crear la cuenta de la consola.",
			Tecnico: err.Error(),
		})

		return
	}

	a.entrar(w, credenciales.Usuario, http.StatusCreated)
}

// Entrar inicia sesión con la cuenta existente.
func (a *Auth) Entrar(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodPost) {
		return
	}

	credenciales, ok := leerCredenciales(w, r)
	if !ok {
		return
	}

	autenticado, err := a.servicio.Autenticar(r.Context(), credenciales.Usuario, credenciales.Password)
	if err != nil {
		escribirJSON(w, http.StatusInternalServerError, respuestaError{
			Mensaje: "No fue posible comprobar las credenciales.",
			Tecnico: err.Error(),
		})

		return
	}

	if !autenticado {
		escribirJSON(w, http.StatusUnauthorized, respuestaError{
			Mensaje: "Usuario o contraseña incorrectos.",
		})

		return
	}

	a.entrar(w, credenciales.Usuario, http.StatusOK)
}

// Salir vence la cookie de sesión.
func (a *Auth) Salir(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodPost) {
		return
	}

	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- la consola se sirve por HTTP local.
		Name:     nombreCookieSesion,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusNoContent)
}

// entrar emite la cookie y responde con el estado de la sesión.
func (a *Auth) entrar(w http.ResponseWriter, usuario string, codigo int) {
	token, err := a.servicio.AbrirSesion(usuario)
	if err != nil {
		escribirJSON(w, http.StatusInternalServerError, respuestaError{
			Mensaje: "No fue posible abrir la sesión de la consola.",
			Tecnico: err.Error(),
		})

		return
	}

	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- la consola se sirve por HTTP local.
		Name:     nombreCookieSesion,
		Value:    token,
		Path:     "/",
		MaxAge:   int(services.VigenciaSesionConsola.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	escribirJSON(w, codigo, estadoAutenticacion{
		Configurada: true,
		Autenticado: true,
		Usuario:     usuario,
	})
}

// leerCredenciales decodifica y valida el cuerpo de acceso. Responde el
// error y devuelve false cuando no es utilizable.
func leerCredenciales(w http.ResponseWriter, r *http.Request) (credencialesConsola, bool) {
	var credenciales credencialesConsola

	if err := decodificarJSON(w, r, &credenciales); err != nil {
		responderValidacion(w, err)

		return credencialesConsola{}, false
	}

	credenciales.Usuario = strings.TrimSpace(credenciales.Usuario)

	if !reUsuarioConsola.MatchString(credenciales.Usuario) {
		responderValidacion(w, campoInvalido(campoUsuario,
			"debe tener entre 3 y 32 caracteres y empezar por minúscula o guion bajo"))

		return credencialesConsola{}, false
	}

	if len(credenciales.Password) < minPasswordConsola || len(credenciales.Password) > maxPasswordConsola {
		responderValidacion(w, campoInvalido(campoPassword,
			"debe tener entre 8 y 128 caracteres"))

		return credencialesConsola{}, false
	}

	return credenciales, true
}

// ConAutenticacion exige la cookie de sesión en la API, salvo en las rutas
// de acceso. Los archivos de la interfaz se sirven sin cookie: la propia
// aplicación muestra el formulario cuando `estado` no está autenticado.
func ConAutenticacion(siguiente http.Handler, servicio ServicioConsola) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !exigeSesion(r.URL.Path) {
			siguiente.ServeHTTP(w, r)

			return
		}

		usuario, ok := servicio.Validar(tokenDe(r))
		if !ok {
			escribirJSON(w, http.StatusUnauthorized, respuestaError{
				Mensaje: "La sesión de la consola no está iniciada.",
			})

			return
		}

		siguiente.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claveUsuario{}, usuario)))
	})
}

// exigeSesion informa si la ruta requiere cookie de sesión.
func exigeSesion(ruta string) bool {
	return strings.HasPrefix(ruta, "/api/") && !strings.HasPrefix(ruta, prefijoAcceso)
}

// tokenDe devuelve el valor de la cookie de sesión.
func tokenDe(r *http.Request) string {
	cookie, err := r.Cookie(nombreCookieSesion)
	if err != nil {
		return ""
	}

	return cookie.Value
}

// UsuarioDe devuelve el usuario autenticado del contexto.
func UsuarioDe(ctx context.Context) string {
	usuario, ok := ctx.Value(claveUsuario{}).(string)
	if !ok {
		return ""
	}

	return usuario
}
