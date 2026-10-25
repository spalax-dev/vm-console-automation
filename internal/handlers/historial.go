// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
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
	ID        int64  `json:"id"`
	FechaHora string `json:"fecha_hora"`
	VM        string `json:"vm"`
	Modulo    string `json:"modulo"`
	Operacion string `json:"operacion"`
	Resultado string `json:"resultado"`
	Detalle   string `json:"detalle,omitempty"`
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
		escribirJSON(w, http.StatusInternalServerError, respuestaOperacion{
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

		registro := &registroWriter{ResponseWriter: w}
		siguiente.ServeHTTP(registro, r)

		estado := registro.estado
		if estado == 0 {
			estado = http.StatusOK
		}

		resultado := "ok"
		if estado >= http.StatusBadRequest {
			resultado = "error"
		}

		operacion := r.Method + " " + r.Pattern
		if r.Pattern == "" {
			operacion = r.Method + " " + r.URL.Path
		}

		nombreVM := ""
		if vm != nil {
			nombreVM = vm()
		}

		if err := historial.Registrar(context.Background(), &services.Operacion{
			VM:        nombreVM,
			Modulo:    modulo,
			Operacion: operacion,
			Resultado: resultado,
			Detalle:   "HTTP " + strconv.Itoa(estado),
		}); err != nil {
			slog.Error("registrar la operación", claveError, err)
		}
	})
}
