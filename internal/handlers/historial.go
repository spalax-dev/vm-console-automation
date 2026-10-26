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

// ServicioHistorial registra y consulta las operaciones de la consola.
type ServicioHistorial interface {
	Registrar(ctx context.Context, operacion *services.Operacion) error
	Listar(ctx context.Context, filtro *services.FiltroHistorial) ([]services.Operacion, error)
}

// Historial expone GET /api/historial con los filtros del diseño.
type Historial struct {
	servicio ServicioHistorial
}

// NuevoHistorial construye el handler del historial.
func NuevoHistorial(servicio ServicioHistorial) *Historial {
	return &Historial{servicio: servicio}
}

// operacionRespuesta es una fila del historial en JSON.
type operacionRespuesta struct {
	ID          int64  `json:"id"`
	FechaHora   string `json:"fecha_hora"`
	VM          string `json:"vm"`
	Modulo      string `json:"modulo"`
	Operacion   string `json:"operacion"`
	Descripcion string `json:"descripcion"`
	Resultado   string `json:"resultado"`
	Detalle     string `json:"detalle,omitempty"`
}

// listaHistorial es el cuerpo de GET /api/historial.
type listaHistorial struct {
	Operaciones []operacionRespuesta `json:"operaciones"`
}

// ServeHTTP implementa http.Handler.
func (h *Historial) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet) {
		return
	}

	filtro, err := filtroDeHistorial(r)
	if err != nil {
		responderValidacion(w, err)

		return
	}

	operaciones, err := h.servicio.Listar(r.Context(), filtro)
	if err != nil {
		slog.Error("consultar el historial", claveError, err)
		escribirJSON(w, http.StatusInternalServerError, respuestaError{
			Mensaje: "No fue posible consultar el historial.",
			Tecnico: err.Error(),
		})

		return
	}

	respuesta := listaHistorial{Operaciones: make([]operacionRespuesta, 0, len(operaciones))}
	for i := range operaciones {
		respuesta.Operaciones = append(respuesta.Operaciones, operacionRespuesta(operaciones[i]))
	}

	escribirJSON(w, http.StatusOK, respuesta)
}

// filtroDeHistorial valida y traduce los parámetros de la query.
func filtroDeHistorial(r *http.Request) (*services.FiltroHistorial, error) {
	query := r.URL.Query()

	filtro := &services.FiltroHistorial{
		Texto:     query.Get("q"),
		VM:        query.Get("vm"),
		Modulo:    query.Get("modulo"),
		Desde:     query.Get("desde"),
		Hasta:     query.Get("hasta"),
		HoraDesde: query.Get("hora_desde"),
		HoraHasta: query.Get("hora_hasta"),
	}

	if valor := query.Get("limite"); valor != "" {
		limite, err := strconv.Atoi(valor)
		if err != nil || limite < 1 || limite > 1000 {
			return nil, campoInvalido("limite", "debe estar entre 1 y 1000")
		}

		filtro.Limite = limite
	}

	if err := validarFechas(filtro); err != nil {
		return nil, err
	}

	if err := validarHoras(filtro); err != nil {
		return nil, err
	}

	return filtro, nil
}

// validarFechas comprueba el formato de las fechas del filtro.
func validarFechas(filtro *services.FiltroHistorial) error {
	for campo, valor := range map[string]string{"desde": filtro.Desde, "hasta": filtro.Hasta} {
		if valor == "" {
			continue
		}

		if _, err := time.Parse(time.DateOnly, valor); err != nil {
			return campoInvalido(campo, "debe tener el formato AAAA-MM-DD")
		}
	}

	return nil
}

// validarHoras comprueba el formato de las horas del filtro.
func validarHoras(filtro *services.FiltroHistorial) error {
	for campo, valor := range map[string]string{"hora_desde": filtro.HoraDesde, "hora_hasta": filtro.HoraHasta} {
		if valor == "" {
			continue
		}

		if _, err := time.Parse("15:04", valor); err != nil {
			return campoInvalido(campo, "debe tener el formato HH:MM")
		}
	}

	return nil
}

// leerCuerpo lee el JSON de la petición para describir la operación y
// lo restaura para que el handler lo vuelva a leer. Nunca guarda el
// cuerpo completo: solo se usan campos descriptivos.
func leerCuerpo(r *http.Request) map[string]any {
	datos := map[string]any{}

	if r.Body == nil || !strings.HasPrefix(r.Header.Get(cabeceraContentType), "application/json") {
		return datos
	}

	cuerpo, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	r.Body = io.NopCloser(bytes.NewReader(cuerpo))

	if err != nil {
		return datos
	}

	if err := json.Unmarshal(cuerpo, &datos); err != nil || datos == nil {
		datos = map[string]any{}
	}

	return datos
}

// estadoDe devuelve el estado escrito o 200 si el handler no lo fijó.
func estadoDe(registro *registroWriter) int {
	if registro.estado == 0 {
		return http.StatusOK
	}

	return registro.estado
}

// resultadoDe traduce el código HTTP a «ok» o «error».
func resultadoDe(estado int) string {
	if estado >= http.StatusBadRequest {
		return "error"
	}

	return "ok"
}

// nombreActivo consulta la máquina vinculada sin romper si no hay una.
func nombreActivo(vm func() string) string {
	if vm == nil {
		return ""
	}

	return vm()
}

// maquinaDe elige la máquina afectada: la del cuerpo para las
// operaciones de sesión, la de la ruta para el encendido y, si no, la
// máquina activa.
func maquinaDe(r *http.Request, datos map[string]any, activa string) string {
	if strings.HasPrefix(r.URL.Path, "/api/session") {
		if maquina, ok := datos["vm"].(string); ok && maquina != "" {
			return maquina
		}
	}

	if strings.HasPrefix(r.URL.Path, "/api/vms/") {
		partes := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(partes) >= 3 {
			return partes[2]
		}
	}

	return activa
}

// completarDatos agrega la opción de borrado de archivos a la
// descripción.
func completarDatos(r *http.Request, datos map[string]any) {
	if r.URL.Query().Get("files") == "true" {
		datos["archivos"] = true
	}
}

// resumenDe arma la operación técnica y la descripción legible.
func resumenDe(r *http.Request, datos map[string]any, nombreVM string) (operacion, descripcion string) {
	operacion = r.Method + " " + r.Pattern
	if r.Pattern == "" {
		operacion = r.Method + " " + r.URL.Path
	}

	if _, existe := datos["vm"]; !existe && nombreVM != "" {
		datos["vm"] = nombreVM
	}

	descripcion = services.DescribirOperacion(r.Method, r.Pattern, r.URL.Path, datos)

	return operacion, descripcion
}

// registroWriter conserva el código de estado que escribe el handler.
type registroWriter struct {
	http.ResponseWriter
	estado int
}

// WriteHeader registra el primer estado escrito.
func (w *registroWriter) WriteHeader(codigo int) {
	if w.estado == 0 {
		w.estado = codigo
	}

	w.ResponseWriter.WriteHeader(codigo)
}

// Write registra 200 si el handler no llamó a WriteHeader.
func (w *registroWriter) Write(datos []byte) (int, error) {
	if w.estado == 0 {
		w.estado = http.StatusOK
	}

	escritos, err := w.ResponseWriter.Write(datos)
	if err != nil {
		return escritos, fmt.Errorf("escribir la respuesta: %w", err)
	}

	return escritos, nil
}

// ConHistorial envuelve el mux para registrar las operaciones mutantes.
func ConHistorial(siguiente http.Handler, historial ServicioHistorial, vm func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		modulo := services.ModuloDe(r.URL.Path)
		if r.Method == http.MethodGet || modulo == "" {
			siguiente.ServeHTTP(w, r)

			return
		}

		datos := leerCuerpo(r)
		completarDatos(r, datos)

		registro := &registroWriter{ResponseWriter: w}
		siguiente.ServeHTTP(registro, r)

		estado := estadoDe(registro)
		nombreVM := maquinaDe(r, datos, nombreActivo(vm))
		operacion, descripcion := resumenDe(r, datos, nombreVM)

		if err := historial.Registrar(context.Background(), &services.Operacion{
			VM:          nombreVM,
			Modulo:      modulo,
			Operacion:   operacion,
			Descripcion: descripcion,
			Resultado:   resultadoDe(estado),
			Detalle:     "HTTP " + strconv.Itoa(estado),
		}); err != nil {
			slog.Error("registrar la operación", claveError, err)
		}
	})
}
