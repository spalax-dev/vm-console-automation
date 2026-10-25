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
)

// maxCuerpo es el tamaño máximo aceptado para el cuerpo JSON de una
// petición (1 MiB).
const maxCuerpo = 1 << 20

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
		slog.Error("escribir la respuesta JSON", "error", err)
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

// noImplementado responde 501 para una operación validada cuyo caso de
// uso todavía no está conectado con la máquina.
func noImplementado(w http.ResponseWriter, operacion string) {
	escribirJSON(w, http.StatusNotImplemented, respuestaError{
		Mensaje: fmt.Sprintf("La operación «%s» aún no está conectada con la máquina.", operacion),
		Tecnico: "endpoint disponible; falta conectar el caso de uso",
	})
}
