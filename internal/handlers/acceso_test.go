// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"net/http"
	"testing"

	"consola-vm/internal/services"
)

// rutaRegistro es la ruta del recurso de registro.
const rutaRegistro = "/api/session/register"

// stubAcceso es un servicio de credenciales falso.
type stubAcceso struct {
	resultado   services.Resultado
	err         error
	usuario     string
	password    string
	registrados int
	olvidados   int
}

// Registrar registra los argumentos y devuelve el resultado.
func (s *stubAcceso) Registrar(_ context.Context, usuario, password string) (services.Resultado, error) {
	s.usuario = usuario
	s.password = password
	s.registrados++

	return s.resultado, s.err
}

// Olvidar registra la llamada.
func (s *stubAcceso) Olvidar() error {
	s.olvidados++

	return s.err
}

// TestAccesoIniciar cubre el guardado correcto de credenciales.
func TestAccesoIniciar(t *testing.T) {
	t.Parallel()

	servicio := &stubAcceso{resultado: services.Resultado{Mensaje: "Credenciales verificadas."}}

	rec := ejecutarCaso(t, NuevoAcceso(servicio).ServeHTTP, rutaRegistro, casoPeticion{
		nombre: "registrar", metodo: http.MethodPost, cuerpo: `{"usuario":"root","password":"secreto"}`,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf(msgCodigo, rec.Code, http.StatusCreated)
	}

	if servicio.registrados != 1 || servicio.usuario != "root" {
		t.Errorf("Registrar() = %+v, want root", servicio)
	}
}

// TestAccesoIniciarRechazado cubre las credenciales inválidas.
func TestAccesoIniciarRechazado(t *testing.T) {
	t.Parallel()

	servicio := &stubAcceso{err: services.ErrCredencialesInvalidas}

	rec := ejecutarCaso(t, NuevoAcceso(servicio).ServeHTTP, rutaRegistro, casoPeticion{
		nombre: "credenciales inválidas", metodo: http.MethodPost, cuerpo: `{"usuario":"root","password":"mala"}`,
	})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(msgCodigo, rec.Code, http.StatusUnauthorized)
	}
}

// TestAccesoIniciarSinSesion cubre la máquina apagada o sin Guest
// Additions.
func TestAccesoIniciarSinSesion(t *testing.T) {
	t.Parallel()

	servicio := &stubAcceso{err: services.ErrSesionNoDisponible}

	rec := ejecutarCaso(t, NuevoAcceso(servicio).ServeHTTP, rutaRegistro, casoPeticion{
		nombre: "sin sesión", metodo: http.MethodPost, cuerpo: `{"usuario":"root","password":"x"}`,
	})

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(msgCodigo, rec.Code, http.StatusServiceUnavailable)
	}
}

// TestAccesoIniciarSinMaquina cubre la consola sin vínculo.
func TestAccesoIniciarSinMaquina(t *testing.T) {
	t.Parallel()

	servicio := &stubAcceso{err: services.ErrSinMaquina}

	rec := ejecutarCaso(t, NuevoAcceso(servicio).ServeHTTP, rutaRegistro, casoPeticion{
		nombre: "sin máquina", metodo: http.MethodPost, cuerpo: `{"usuario":"root","password":"x"}`,
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf(msgCodigo, rec.Code, http.StatusConflict)
	}
}

// TestAccesoValidacion cubre usuario, puerto y método.
func TestAccesoValidacion(t *testing.T) {
	t.Parallel()

	handler := NuevoAcceso(&stubAcceso{}).ServeHTTP
	casos := []casoPeticion{
		{nombre: "usuario inválido", metodo: http.MethodPost, cuerpo: `{"usuario":"Root","password":"x"}`, codigo: http.StatusBadRequest, campo: campoUsuario},
		{nombre: casoCuerpoInvalido, metodo: http.MethodPost, cuerpo: `{`, codigo: http.StatusBadRequest},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodGet, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, rutaRegistro, caso)
		comprobarCaso(t, rec, caso)
	}
}

// TestAccesoOlvidar elimina las credenciales.
func TestAccesoOlvidar(t *testing.T) {
	t.Parallel()

	servicio := &stubAcceso{}

	rec := ejecutarCaso(t, NuevoAcceso(servicio).ServeHTTP, rutaRegistro, casoPeticion{
		nombre: "olvidar", metodo: http.MethodDelete,
	})

	if rec.Code != http.StatusNoContent {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNoContent)
	}

	if servicio.olvidados != 1 {
		t.Errorf("Olvidar() se llamó %d veces, want 1", servicio.olvidados)
	}
}

// TestAccesoRecortaElUsuario comprueba que los espacios del autocompletado
// no invalidan un usuario correcto.
func TestAccesoRecortaElUsuario(t *testing.T) {
	t.Parallel()

	servicio := &stubAcceso{resultado: services.Resultado{Mensaje: "Credenciales verificadas."}}

	rec := ejecutarCaso(t, NuevoAcceso(servicio).ServeHTTP, rutaRegistro, casoPeticion{
		nombre: "usuario con espacios", metodo: http.MethodPost, cuerpo: `{"usuario":" jorge ","password":"clave"}`,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf(msgCodigo, rec.Code, http.StatusCreated)
	}

	if servicio.usuario != usuarioJorge {
		t.Errorf("usuario = %q, want jorge", servicio.usuario)
	}
}
