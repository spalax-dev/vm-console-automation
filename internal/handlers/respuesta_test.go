// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/services"
)

// errFallo simula un error genérico sin comando asociado.
var errFallo = errors.New("fallo interno")

// Órdenes usadas en las pruebas del cuerpo de error.
const (
	operacionCrearJorge   = "useradd -m jorge"
	operacionGuestControl = "VBoxManage guestcontrol ejecutar"
)

// cuerpoDeRespuesta decodifica el cuerpo JSON de una respuesta grabada.
func cuerpoDeRespuesta(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var cuerpo map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	return cuerpo
}

// comprobarSinClaves verifica que el cuerpo JSON no incluya las claves
// dadas, para que un error no emita datos de una operación que no ocurrió.
func comprobarSinClaves(t *testing.T, cuerpo map[string]any, claves ...string) {
	t.Helper()

	for _, clave := range claves {
		if valor, existe := cuerpo[clave]; existe {
			t.Errorf("la respuesta no debe incluir %q y vale %v", clave, valor)
		}
	}
}

// TestRespuestaValidacionOmiteDatosDeOperacion comprueba que un 400 de
// validación nombre el campo rechazado y no invente datos de operación.
func TestRespuestaValidacionOmiteDatosDeOperacion(t *testing.T) {
	t.Parallel()

	rec := ejecutarCaso(t, NuevoUsuarios(&stubUsuarios{}).Coleccion, rutaUsuarios, casoPeticion{
		nombre: "nombre inválido", metodo: http.MethodPost,
		cuerpo: `{"nombre":"-mal-","sudo":false,"llaves":false}`,
	})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(msgCodigoCuerpo, rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	cuerpo := cuerpoDeRespuesta(t, rec)

	if cuerpo["campo"] != campoNombre {
		t.Errorf("campo = %v, want %q", cuerpo["campo"], campoNombre)
	}

	comprobarSinClaves(t, cuerpo, "operacion", "codigo", "salida", "stderr")
}

// TestRespuestaAutenticacionSinCodigo comprueba que los rechazos de la
// autenticación no emitan un código de salida inventado.
func TestRespuestaAutenticacionSinCodigo(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre    string
		servicio  *stubConsola
		manejador func(*Auth) http.HandlerFunc
		esperado  int
	}{
		{
			nombre:    "credenciales incorrectas",
			servicio:  &stubConsola{autenticado: false},
			manejador: func(a *Auth) http.HandlerFunc { return a.Entrar },
			esperado:  http.StatusUnauthorized,
		},
		{
			nombre:    "cuenta existente",
			servicio:  &stubConsola{configurarErr: services.ErrConsolaConfigurada},
			manejador: func(a *Auth) http.HandlerFunc { return a.Configurar },
			esperado:  http.StatusConflict,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			rec := ejecutarCaso(t, caso.manejador(NuevoAuth(caso.servicio)), rutaAuthLogin, casoPeticion{
				nombre: caso.nombre, metodo: http.MethodPost, cuerpo: cuerpoCredenciales,
			})

			if rec.Code != caso.esperado {
				t.Fatalf(msgCodigoCuerpo, rec.Code, caso.esperado, rec.Body.String())
			}

			cuerpo := cuerpoDeRespuesta(t, rec)

			if cuerpo["ok"] != false {
				t.Errorf("ok = %v, want false", cuerpo["ok"])
			}

			if cuerpo["mensaje"] == "" || cuerpo["mensaje"] == nil {
				t.Errorf("la respuesta debe incluir un mensaje")
			}

			comprobarSinClaves(t, cuerpo, "codigo", "operacion", "salida", "stderr")
		})
	}
}

// TestRespuestaErrorDeOperacionConservaLaOrden comprueba que el fallo de un
// comando conserve la operación, el código, la salida y el detalle técnico.
func TestRespuestaErrorDeOperacionConservaLaOrden(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{
		resultado: services.Resultado{
			Operacion: operacionCrearJorge,
			Codigo:    9,
			Salida:    "creando la cuenta",
			Stderr:    "useradd: el usuario ya existe",
		},
		err: &domain.ExitError{Code: 9, Command: operacionCrearJorge},
	}

	rec := ejecutarCaso(t, NuevoUsuarios(servicio).Coleccion, rutaUsuarios, casoPeticion{
		nombre: "fallo de useradd", metodo: http.MethodPost,
		cuerpo: `{"nombre":"jorge","sudo":true,"llaves":false}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigoCuerpo, rec.Code, http.StatusInternalServerError, rec.Body.String())
	}

	cuerpo := cuerpoDeRespuesta(t, rec)

	if cuerpo["ok"] != false {
		t.Errorf("ok = %v, want false", cuerpo["ok"])
	}

	if cuerpo["mensaje"] != "No fue posible completar la operación." {
		t.Errorf("mensaje = %v, want el mensaje genérico de la operación", cuerpo["mensaje"])
	}

	if cuerpo["operacion"] != operacionCrearJorge {
		t.Errorf("operacion = %v, want %q", cuerpo["operacion"], operacionCrearJorge)
	}

	if cuerpo["codigo"] != float64(9) {
		t.Errorf("codigo = %v, want 9", cuerpo["codigo"])
	}

	if cuerpo["salida"] != "creando la cuenta" {
		t.Errorf("salida = %v, want la salida del comando", cuerpo["salida"])
	}

	if cuerpo["stderr"] != "useradd: el usuario ya existe" {
		t.Errorf("stderr = %v, want el stderr del comando", cuerpo["stderr"])
	}

	if tecnico, ok := cuerpo["tecnico"].(string); !ok || tecnico == "" {
		t.Errorf("la respuesta debe incluir el detalle técnico, tecnico = %v", cuerpo["tecnico"])
	}

	comprobarSinClaves(t, cuerpo, "campo")
}

// TestRespuestaCredencialesInvalidasConservaElCodigo comprueba que un 401
// con un comando ejecutado sí incluya su código de salida.
func TestRespuestaCredencialesInvalidasConservaElCodigo(t *testing.T) {
	t.Parallel()

	servicio := &stubAcceso{
		resultado: services.Resultado{
			Operacion: operacionGuestControl,
			Codigo:    5,
			Stderr:    "VBoxManage: error",
		},
		err: services.ErrCredencialesInvalidas,
	}

	rec := ejecutarCaso(t, NuevoAcceso(servicio).ServeHTTP, rutaRegistro, casoPeticion{
		nombre: "credenciales rechazadas", metodo: http.MethodPost,
		cuerpo: `{"usuario":"root","password":"secreto"}`,
	})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(msgCodigoCuerpo, rec.Code, http.StatusUnauthorized, rec.Body.String())
	}

	cuerpo := cuerpoDeRespuesta(t, rec)

	if cuerpo["operacion"] != operacionGuestControl {
		t.Errorf("operacion = %v, want %q", cuerpo["operacion"], operacionGuestControl)
	}

	if cuerpo["codigo"] != float64(5) {
		t.Errorf("codigo = %v, want 5", cuerpo["codigo"])
	}

	if cuerpo["stderr"] != "VBoxManage: error" {
		t.Errorf("stderr = %v, want el stderr del comando", cuerpo["stderr"])
	}
}

// TestRespuestaDeErrorOmiteLaOperacionVacia comprueba que un resultado sin
// comando no aporte campos a la respuesta.
func TestRespuestaDeErrorOmiteLaOperacionVacia(t *testing.T) {
	t.Parallel()

	cuerpo := respuestaDeError(services.Resultado{}, "no fue posible", errFallo)

	if cuerpo.OK {
		t.Errorf("OK = true, want false")
	}

	if cuerpo.Mensaje != "no fue posible" {
		t.Errorf("Mensaje = %q, want %q", cuerpo.Mensaje, "no fue posible")
	}

	if cuerpo.Tecnico != errFallo.Error() {
		t.Errorf("Tecnico = %q, want %q", cuerpo.Tecnico, errFallo.Error())
	}

	if cuerpo.Operacion != "" || cuerpo.Codigo != 0 || cuerpo.Salida != "" || cuerpo.Stderr != "" {
		t.Errorf("el resultado vacío no debe aportar datos de la operación: %+v", cuerpo)
	}
}
