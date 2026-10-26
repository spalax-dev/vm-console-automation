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

// maxBody es el tamaño máximo aceptado para el cuerpo JSON de una
// petición (1 MiB).
const maxBody = 1 << 20

// errorKey es la clave de los mensajes de error en el log.
const errorKey = "error"

// Cabeceras HTTP usadas por los handlers.
const (
	headerContentType = "Content-Type"
	headerLocation    = "Location"
)

// errorResponse es el cuerpo JSON común de todos los errores de la
// consola. OK siempre es false.
//
// Campo solo aparece en los errores de validación. Operation, Codigo,
// Salida y Stderr solo aparecen cuando el fallo ocurrió después de ejecutar
// un comando, de modo que un 400 o un 401 no inventan un código de salida.
// El código de la operación se omite cuando vale cero.
type errorResponse struct {
	OK        bool   `json:"ok"`
	Message   string `json:"mensaje"`
	Field     string `json:"campo,omitempty"`
	Operation string `json:"operacion,omitempty"`
	Code      int    `json:"codigo,omitempty"`
	Output    string `json:"salida,omitempty"`
	Stderr    string `json:"stderr,omitempty"`
	Technical string `json:"tecnico,omitempty"`
}

// fieldError describe un campo que no cumple una regla de validación.
type fieldError struct {
	// Field es el nombre del campo rechazado.
	Field string
	// Reason explica por qué no es válido.
	Reason string
}

// Error implementa la interfaz error con un mensaje comprensible.
func (e *fieldError) Error() string {
	return fmt.Sprintf("el campo %q %s", e.Field, e.Reason)
}

// invalidField construye un campoError.
func invalidField(field, reason string) error {
	return &fieldError{Field: field, Reason: reason}
}

// writeJSON serializa cuerpo como JSON con el código de estado dado.
// Si la serialización falla, solo deja constancia en el log.
func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set(headerContentType, "application/json; charset=utf-8")
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("escribir la respuesta JSON", errorKey, err)
	}
}

// requireMethod comprueba que la petición use uno de los métodos dados.
// Si no, responde 405 con el encabezado Allow y devuelve false.
func requireMethod(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, method := range methods {
		if r.Method == method {
			return true
		}
	}

	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
		Message: "Método no permitido para este recurso.",
	})

	return false
}

// decodeJSON lee el cuerpo JSON de la petición en destino, con
// límite de tamaño y sin admitir campos desconocidos.
func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)

	decodificador := json.NewDecoder(r.Body)
	decodificador.DisallowUnknownFields()

	if err := decodificador.Decode(destination); err != nil {
		return fmt.Errorf("el cuerpo de la petición no es un JSON válido: %w", err)
	}

	return nil
}

// respondValidacion traduce un error de validación a 400, nombrando
// el campo rechazado cuando se conoce.
func respondValidacion(w http.ResponseWriter, err error) {
	var field *fieldError

	if errors.As(err, &field) {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: field.Error(),
			Field:   field.Field,
		})

		return
	}

	writeJSON(w, http.StatusBadRequest, errorResponse{Message: err.Error()})
}

// successResponse es el cuerpo JSON de una operación de módulo
// completada con éxito. Los errores usan respuestaError.
type successResponse struct {
	OK        bool   `json:"ok"`
	Message   string `json:"mensaje"`
	Operation string `json:"operacion,omitempty"`
	Code      int    `json:"codigo"`
	Output    string `json:"salida,omitempty"`
	Stderr    string `json:"stderr,omitempty"`
	Technical string `json:"tecnico,omitempty"`
}

// responseFromError compone el cuerpo de un error a partir del resultado que
// devolvió el servicio y del error correspondiente. Solo el mensaje y el
// detalle técnico son obligatorios: los datos de la operación se omiten si
// el servicio no llegó a ejecutar ningún comando.
func responseFromError(result services.Outcome, message string, err error) errorResponse {
	return errorResponse{
		Message:   message,
		Operation: result.Operation,
		Code:      result.Code,
		Output:    result.Output,
		Stderr:    result.Stderr,
		Technical: err.Error(),
	}
}

// writeResult escribe el resultado de una operación con el código
// HTTP indicado.
func writeResult(w http.ResponseWriter, code int, result services.Outcome) {
	writeJSON(w, code, successResponse{
		OK:        true,
		Message:   result.Message,
		Operation: result.Operation,
		Code:      result.Code,
		Output:    result.Output,
		Stderr:    result.Stderr,
	})
}

// respondErrorOperation traduce el fallo de un caso de uso a la
// respuesta HTTP correspondiente, conservando el detalle técnico.
func respondErrorOperation(w http.ResponseWriter, result services.Outcome, err error) {
	if errors.Is(err, services.ErrSessionUnavailable) {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{
			Message:   "No hay una sesión disponible con la máquina: enciéndela o revisa la red.",
			Technical: err.Error(),
		})

		return
	}

	var exitErr *domain.ExitError
	if errors.As(err, &exitErr) {
		writeJSON(w, http.StatusInternalServerError,
			responseFromError(result, "No fue posible completar la operación.", err))

		return
	}

	slog.Error("ejecutar la operación", errorKey, err)
	writeJSON(w, http.StatusInternalServerError,
		responseFromError(result, "No fue posible completar la operación.", err))
}
