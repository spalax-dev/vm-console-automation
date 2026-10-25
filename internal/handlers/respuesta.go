// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/services"
)

// maxCuerpo es el tamaño máximo aceptado para el cuerpo JSON de una
// petición (1 MiB).
const maxCuerpo = 1 << 20

// claveError es la clave de los mensajes de error en el log.
const claveError = "error"

// respuestaError es el cuerpo JSON común de los errores de la consola.
type respuestaError struct {
	OK      bool   `json:"ok"`
	Mensaje string `json:"mensaje"`
	Campo   string `json:"campo,omitempty"`
	Tecnico string `json:"tecnico,omitempty"`
}

// campoError describe un campo que no cumple una regla de validación.
type campoError struct {
	// Campo es el nombre del campo rechazado.
	Campo string
	// Motivo explica por qué no es válido.
	Motivo string
}

// Error implementa la interfaz error con un mensaje comprensible.
func (e *campoError) Error() string {
	return fmt.Sprintf("el campo %q %s", e.Campo, e.Motivo)
}

// campoInvalido construye un campoError.
func campoInvalido(campo, motivo string) error {
	return &campoError{Campo: campo, Motivo: motivo}
}

// escribirJSON serializa cuerpo como JSON con el código de estado dado.
// Si la serialización falla, solo deja constancia en el log.
func escribirJSON(w http.ResponseWriter, codigo int, cuerpo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)

	if err := json.NewEncoder(w).Encode(cuerpo); err != nil {
		slog.Error("escribir la respuesta JSON", claveError, err)
	}
}

// exigirMetodo comprueba que la petición use uno de los métodos dados.
// Si no, responde 405 con el encabezado Allow y devuelve false.
func exigirMetodo(w http.ResponseWriter, r *http.Request, metodos ...string) bool {
	for _, metodo := range metodos {
		if r.Method == metodo {
			return true
		}
	}

	w.Header().Set("Allow", strings.Join(metodos, ", "))
	escribirJSON(w, http.StatusMethodNotAllowed, respuestaError{
		Mensaje: "Método no permitido para este recurso.",
	})

	return false
}

// decodificarJSON lee el cuerpo JSON de la petición en destino, con
// límite de tamaño y sin admitir campos desconocidos.
func decodificarJSON(w http.ResponseWriter, r *http.Request, destino any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxCuerpo)

	decodificador := json.NewDecoder(r.Body)
	decodificador.DisallowUnknownFields()

	if err := decodificador.Decode(destino); err != nil {
		return fmt.Errorf("el cuerpo de la petición no es un JSON válido: %w", err)
	}

	return nil
}

// responderValidacion traduce un error de validación a 400, nombrando
// el campo rechazado cuando se conoce.
func responderValidacion(w http.ResponseWriter, err error) {
	var campo *campoError

	if errors.As(err, &campo) {
		escribirJSON(w, http.StatusBadRequest, respuestaError{
			Mensaje: campo.Error(),
			Campo:   campo.Campo,
		})

		return
	}

	escribirJSON(w, http.StatusBadRequest, respuestaError{Mensaje: err.Error()})
}

// respuestaOperacion es el cuerpo JSON de una operación de módulo.
type respuestaOperacion struct {
	OK        bool   `json:"ok"`
	Mensaje   string `json:"mensaje"`
	Operacion string `json:"operacion,omitempty"`
	Codigo    int    `json:"codigo"`
	Salida    string `json:"salida,omitempty"`
	Stderr    string `json:"stderr,omitempty"`
	Tecnico   string `json:"tecnico,omitempty"`
}

// escribirResultado escribe el resultado de una operación con el código
// HTTP indicado.
func escribirResultado(w http.ResponseWriter, codigo int, resultado services.Resultado) {
	escribirJSON(w, codigo, respuestaOperacion{
		OK:        true,
		Mensaje:   resultado.Mensaje,
		Operacion: resultado.Operacion,
		Codigo:    resultado.Codigo,
		Salida:    resultado.Salida,
		Stderr:    resultado.Stderr,
	})
}

// responderErrorOperacion traduce el fallo de un caso de uso a la
// respuesta HTTP correspondiente, conservando el detalle técnico.
func responderErrorOperacion(w http.ResponseWriter, resultado services.Resultado, err error) {
	if errors.Is(err, services.ErrSesionNoDisponible) {
		escribirJSON(w, http.StatusServiceUnavailable, respuestaOperacion{
			Mensaje: "No hay una sesión disponible con la máquina: enciéndela o revisa la red.",
			Tecnico: err.Error(),
		})

		return
	}

	var exitErr *domain.ExitError
	if errors.As(err, &exitErr) {
		escribirJSON(w, http.StatusInternalServerError, respuestaOperacion{
			Mensaje:   "No fue posible completar la operación.",
			Operacion: resultado.Operacion,
			Codigo:    resultado.Codigo,
			Stderr:    resultado.Stderr,
			Tecnico:   err.Error(),
		})

		return
	}

	slog.Error("ejecutar la operación", claveError, err)
	escribirJSON(w, http.StatusInternalServerError, respuestaOperacion{
		Mensaje: "No fue posible completar la operación.",
		Tecnico: err.Error(),
	})
}

// noImplementado responde 501 para una operación validada cuyo caso de
// uso todavía no está conectado con la máquina.
func noImplementado(w http.ResponseWriter, operacion string) {
	escribirJSON(w, http.StatusNotImplemented, respuestaError{
		Mensaje: fmt.Sprintf("La operación «%s» aún no está conectada con la máquina.", operacion),
		Tecnico: "endpoint disponible; falta conectar el caso de uso",
	})
}
