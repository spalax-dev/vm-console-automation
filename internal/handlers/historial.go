// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"consola-vm/internal/services"
)

// HistoryService registra y consulta las operaciones de la consola.
type HistoryService interface {
	Register(ctx context.Context, operation *services.Operation) error
	List(ctx context.Context, filter *services.HistoryFilter) ([]services.Operation, error)
}

// History expone GET /api/historial con los filtros del diseño.
type History struct {
	service HistoryService
}

// NewHistory construye el handler del historial.
func NewHistory(service HistoryService) *History {
	return &History{service: service}
}

// operationResponse es una fila del historial en JSON.
type operationResponse struct {
	ID          int64  `json:"id"`
	DateHour    string `json:"fecha_hora"`
	VM          string `json:"vm"`
	Module      string `json:"modulo"`
	Operation   string `json:"operacion"`
	Description string `json:"descripcion"`
	Outcome     string `json:"resultado"`
	Detail      string `json:"detalle,omitempty"`
}

// historyList es el cuerpo de GET /api/historial.
type historyList struct {
	Operations []operationResponse `json:"operaciones"`
}

// ServeHTTP implementa http.Handler.
func (h *History) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	filter, err := filterOfHistory(r)
	if err != nil {
		respondValidacion(w, err)

		return
	}

	operations, err := h.service.List(r.Context(), filter)
	if err != nil {
		slog.Error("consultar el historial", errorKey, err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message:   "No fue posible consultar el historial.",
			Technical: err.Error(),
		})

		return
	}

	response := historyList{Operations: make([]operationResponse, 0, len(operations))}
	for i := range operations {
		response.Operations = append(response.Operations, operationResponse(operations[i]))
	}

	writeJSON(w, http.StatusOK, response)
}

// filterOfHistory valida y traduce los parámetros de la query.
func filterOfHistory(r *http.Request) (*services.HistoryFilter, error) {
	query := r.URL.Query()

	filter := &services.HistoryFilter{
		Text:     query.Get("q"),
		VM:       query.Get("vm"),
		Module:   query.Get("modulo"),
		From:     query.Get("desde"),
		To:       query.Get("hasta"),
		HourFrom: query.Get("hora_desde"),
		HourTo:   query.Get("hora_hasta"),
	}

	if value := query.Get("limite"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 1000 {
			return nil, invalidField("limite", "debe estar entre 1 y 1000")
		}

		filter.Limit = limit
	}

	if err := validateFechas(filter); err != nil {
		return nil, err
	}

	if err := validateHoras(filter); err != nil {
		return nil, err
	}

	return filter, nil
}

// validateFechas comprueba el formato de las fechas del filtro.
func validateFechas(filter *services.HistoryFilter) error {
	for field, value := range map[string]string{"desde": filter.From, "hasta": filter.To} {
		if value == "" {
			continue
		}

		if _, err := time.Parse(time.DateOnly, value); err != nil {
			return invalidField(field, "debe tener el formato AAAA-MM-DD")
		}
	}

	return nil
}

// validateHoras comprueba el formato de las horas del filtro.
func validateHoras(filter *services.HistoryFilter) error {
	for field, value := range map[string]string{"hora_desde": filter.HourFrom, "hora_hasta": filter.HourTo} {
		if value == "" {
			continue
		}

		if _, err := time.Parse("15:04", value); err != nil {
			return invalidField(field, "debe tener el formato HH:MM")
		}
	}

	return nil
}

// readBody lee el JSON de la petición para describir la operación y
// lo restaura para que el handler lo vuelva a leer. Nunca guarda el
// cuerpo completo: solo se usan campos descriptivos.
func readBody(r *http.Request) map[string]any {
	data := map[string]any{}

	if r.Body == nil || !strings.HasPrefix(r.Header.Get(headerContentType), "application/json") {
		return data
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	r.Body = io.NopCloser(bytes.NewReader(body))

	if err != nil {
		return data
	}

	if err := json.Unmarshal(body, &data); err != nil || data == nil {
		data = map[string]any{}
	}

	return data
}

// stateOf devuelve el estado escrito o 200 si el handler no lo fijó.
func stateOf(registration *registrationWriter) int {
	if registration.state == 0 {
		return http.StatusOK
	}

	return registration.state
}

// resultOf traduce el código HTTP a «ok» o «error».
func resultOf(state int) string {
	if state >= http.StatusBadRequest {
		return "error"
	}

	return "ok"
}

// activeName consulta la máquina vinculada sin romper si no hay una.
func activeName(vm func() string) string {
	if vm == nil {
		return ""
	}

	return vm()
}

// machineFrom elige la máquina afectada: la del cuerpo para las
// operaciones de sesión, la de la ruta para el encendido y, si no, la
// máquina activa.
func machineFrom(r *http.Request, data map[string]any, active string) string {
	if strings.HasPrefix(r.URL.Path, "/api/session") {
		if machine, ok := data["vm"].(string); ok && machine != "" {
			return machine
		}
	}

	if strings.HasPrefix(r.URL.Path, "/api/vms/") {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) >= 3 {
			return parts[2]
		}
	}

	return active
}

// completeData agrega la opción de borrado de archivos a la
// descripción.
func completeData(r *http.Request, data map[string]any) {
	if r.URL.Query().Get("files") == "true" {
		data["archivos"] = true
	}
}

// resumenOf arma la operación técnica y la descripción legible.
func resumenOf(r *http.Request, data map[string]any, vmName string) (operation, description string) {
	operation = r.Method + " " + r.Pattern
	if r.Pattern == "" {
		operation = r.Method + " " + r.URL.Path
	}

	if _, exists := data["vm"]; !exists && vmName != "" {
		data["vm"] = vmName
	}

	description = services.DescribeOperation(r.Method, r.Pattern, r.URL.Path, data)

	return operation, description
}

// registrationWriter conserva el código de estado que escribe el handler.
type registrationWriter struct {
	http.ResponseWriter
	state int
}

// WriteHeader registra el primer estado escrito.
func (w *registrationWriter) WriteHeader(code int) {
	if w.state == 0 {
		w.state = code
	}

	w.ResponseWriter.WriteHeader(code)
}

// Write registra 200 si el handler no llamó a WriteHeader.
func (w *registrationWriter) Write(data []byte) (int, error) {
	if w.state == 0 {
		w.state = http.StatusOK
	}

	escritos, err := w.ResponseWriter.Write(data)
	if err != nil {
		return escritos, fmt.Errorf("escribir la respuesta: %w", err)
	}

	return escritos, nil
}

// WithHistory envuelve el mux para registrar las operaciones mutantes.
func WithHistory(next http.Handler, history HistoryService, vm func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		module := services.ModuleOf(r.URL.Path)
		if r.Method == http.MethodGet || module == "" {
			next.ServeHTTP(w, r)

			return
		}

		data := readBody(r)
		completeData(r, data)

		registration := &registrationWriter{ResponseWriter: w}
		next.ServeHTTP(registration, r)

		state := stateOf(registration)
		vmName := machineFrom(r, data, activeName(vm))
		operation, description := resumenOf(r, data, vmName)

		if err := history.Register(context.Background(), &services.Operation{
			VM:          vmName,
			Module:      module,
			Operation:   operation,
			Description: description,
			Outcome:     resultOf(state),
			Detail:      "HTTP " + strconv.Itoa(state),
		}); err != nil {
			slog.Error("registrar la operación", errorKey, err)
		}
	})
}
