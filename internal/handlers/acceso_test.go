// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"net/http"
	"testing"

	"consola-vm/internal/services"
)

// rutaLogin es la ruta del recurso de credenciales.
const rutaLogin = "/api/session/login"

// stubAcceso es un servicio de credenciales falso.
type stubAcceso struct {
	resultado services.Resultado
	err       error
	usuario   string
	password  string
	puerto    uint16
	iniciados int
	olvidados int
}

// Iniciar registra los argumentos y devuelve el resultado.
func (s *stubAcceso) Iniciar(_ context.Context, usuario, password string, puerto uint16) (services.Resultado, error) {
	s.usuario = usuario
	s.password = password
	s.puerto = puerto
	s.iniciados++

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

	rec := ejecutarCaso(t, NuevoAcceso(servicio).ServeHTTP, rutaLogin, casoPeticion{
		nombre: "iniciar", metodo: http.MethodPost, cuerpo: `{"usuario":"root","password":"secreto","puerto":22}`,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	if servicio.iniciados != 1 || servicio.usuario != "root" || servicio.puerto != 22 {
		t.Errorf("Iniciar() = %+v, want root/22", servicio)
	}
}

// TestAccesoIniciarRechazado cubre las credenciales inválidas.
func TestAccesoIniciarRechazado(t *testing.T) {
	t.Parallel()

	servicio := &stubAcceso{err: services.ErrCredencialesInvalidas}

	rec := ejecutarCaso(t, NuevoAcceso(servicio).ServeHTTP, rutaLogin, casoPeticion{
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

	rec := ejecutarCaso(t, NuevoAcceso(servicio).ServeHTTP, rutaLogin, casoPeticion{
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

	rec := ejecutarCaso(t, NuevoAcceso(servicio).ServeHTTP, rutaLogin, casoPeticion{
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
		{nombre: "puerto inválido", metodo: http.MethodPost, cuerpo: `{"usuario":"root","password":"x","puerto":70000}`, codigo: http.StatusBadRequest, campo: campoPuerto},
		{nombre: casoCuerpoInvalido, metodo: http.MethodPost, cuerpo: `{`, codigo: http.StatusBadRequest},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodGet, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, rutaLogin, caso)
		comprobarCaso(t, rec, caso)
	}
}

// TestAccesoOlvidar elimina las credenciales.
func TestAccesoOlvidar(t *testing.T) {
	t.Parallel()

	servicio := &stubAcceso{}

	rec := ejecutarCaso(t, NuevoAcceso(servicio).ServeHTTP, rutaLogin, casoPeticion{
		nombre: "olvidar", metodo: http.MethodDelete,
	})

	if rec.Code != http.StatusNoContent {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNoContent)
	}

	if servicio.olvidados != 1 {
		t.Errorf("Olvidar() se llamó %d veces, want 1", servicio.olvidados)
	}
}
