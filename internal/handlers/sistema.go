// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package handlers expone los endpoints HTTP de la consola de
// administración: preparación del sistema, gestión de usuarios,
// gestión de servicios systemd y balanceo de carga con HAProxy. Los
// handlers solo enrutan, validan y traducen a JSON; los casos de uso
// viven en internal/services.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"consola-vm/internal/services"
)

// ServicioPaquetes ejecuta las operaciones de APT sobre la máquina.
type ServicioPaquetes interface {
	// Listar devuelve los paquetes instalados.
	Listar(ctx context.Context) ([]services.Paquete, error)
	// Instalar instala el paquete indicado.
	Instalar(ctx context.Context, nombre string) (services.Resultado, error)
	// Eliminar elimina el paquete indicado.
	Eliminar(ctx context.Context, nombre string) (services.Resultado, error)
}

// ServicioLlaves genera y entrega las llaves SSH de root.
type ServicioLlaves interface {
	// Generar crea el par, instala la pública y deshabilita el acceso
	// por contraseña.
	Generar(ctx context.Context) (services.Resultado, error)
	// Descargar devuelve el nombre y el contenido de la llave privada.
	Descargar() (nombre string, contenido []byte, err error)
}

// ServicioIssue lee y reemplaza /etc/issue.
type ServicioIssue interface {
	// Obtener devuelve la configuración actual del banner.
	Obtener(ctx context.Context) (services.ConfiguracionIssue, error)
	// Escribir reemplaza el banner con la configuración dada.
	Escribir(ctx context.Context, config services.ConfiguracionIssue) (services.Resultado, error)
}

// Sistema agrupa los recursos de preparación del sistema (numeral 3.1
// del enunciado).
type Sistema struct {
	paquetes ServicioPaquetes
	llaves   ServicioLlaves
	issue    ServicioIssue
}

// NuevoSistema construye el handler de preparación del sistema con los
// servicios dados.
func NuevoSistema(paquetes ServicioPaquetes, llaves ServicioLlaves, issue ServicioIssue) *Sistema {
	return &Sistema{paquetes: paquetes, llaves: llaves, issue: issue}
}

// paqueteRespuesta es un paquete en la respuesta JSON.
type paqueteRespuesta struct {
	Nombre  string `json:"nombre"`
	Version string `json:"version"`
	Estado  string `json:"estado"`
}

// listaPaquetes es el cuerpo de GET /api/packages.
type listaPaquetes struct {
	Paquetes []paqueteRespuesta `json:"paquetes"`
}

// solicitudPaquete es el cuerpo de POST /api/packages.
type solicitudPaquete struct {
	Nombre string `json:"nombre"`
}

// solicitudIssue es el cuerpo de PUT /api/issue.
type solicitudIssue struct {
	Mensaje    string `json:"mensaje"`
	NombreHost bool   `json:"nombre_host"`
	Fecha      bool   `json:"fecha"`
	Hora       bool   `json:"hora"`
	IP         bool   `json:"ip"`
}

// solicitudHostname es el cuerpo de PUT /api/hostname.
type solicitudHostname struct {
	Nombre          string `json:"nombre"`
	IP              string `json:"ip"`
	ActualizarHosts bool   `json:"actualizar_hosts"`
}

// Paquetes atiende GET y POST /api/packages: lista los paquetes
// instalados o instala uno nuevo.
func (h *Sistema) Paquetes(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	if r.Method == http.MethodGet {
		h.listarPaquetes(w, r)

		return
	}

	var solicitud solicitudPaquete
	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarPaquete(campoNombre, solicitud.Nombre); err != nil {
		responderValidacion(w, err)

		return
	}

	resultado, err := h.paquetes.Instalar(r.Context(), solicitud.Nombre)
	if err != nil {
		responderErrorOperacion(w, resultado, err)

		return
	}

	w.Header().Set("Location", "/api/packages/"+solicitud.Nombre)
	escribirResultado(w, http.StatusCreated, resultado)
}

// listarPaquetes responde la colección de paquetes instalados.
func (h *Sistema) listarPaquetes(w http.ResponseWriter, r *http.Request) {
	paquetes, err := h.paquetes.Listar(r.Context())
	if err != nil {
		responderErrorOperacion(w, services.Resultado{}, err)

		return
	}

	respuesta := listaPaquetes{Paquetes: make([]paqueteRespuesta, 0, len(paquetes))}
	for _, paquete := range paquetes {
		respuesta.Paquetes = append(respuesta.Paquetes, paqueteRespuesta{
			Nombre:  paquete.Nombre,
			Version: paquete.Version,
			Estado:  paquete.Estado,
		})
	}

	escribirJSON(w, http.StatusOK, respuesta)
}

// Paquete atiende DELETE /api/packages/{name}: elimina el paquete
// indicado en la ruta.
func (h *Sistema) Paquete(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodDelete) {
		return
	}

	nombre := r.PathValue(campoName)
	if err := validarPaquete(campoName, nombre); err != nil {
		responderValidacion(w, err)

		return
	}

	resultado, err := h.paquetes.Eliminar(r.Context(), nombre)
	if err != nil {
		responderErrorOperacion(w, resultado, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// LlaveRoot atiende GET y POST /api/keys/root: descarga la llave
// privada de root o genera un par nuevo.
func (h *Sistema) LlaveRoot(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet, http.MethodPost) {
		return
	}

	if r.Method == http.MethodGet {
		h.descargarLlaveRoot(w)

		return
	}

	resultado, err := h.llaves.Generar(r.Context())
	if errors.Is(err, services.ErrLlaveExistente) {
		escribirJSON(w, http.StatusConflict, respuestaOperacion{
			Mensaje: "La llave SSH de root ya existe en el anfitrión.",
			Tecnico: err.Error(),
		})

		return
	}

	if err != nil {
		responderErrorOperacion(w, resultado, err)

		return
	}

	w.Header().Set("Location", "/api/keys/root")
	escribirResultado(w, http.StatusCreated, resultado)
}

// descargarLlaveRoot entrega la llave privada como archivo adjunto.
func (h *Sistema) descargarLlaveRoot(w http.ResponseWriter) {
	nombre, contenido, err := h.llaves.Descargar()
	if errors.Is(err, services.ErrLlaveNoEncontrada) {
		escribirJSON(w, http.StatusNotFound, respuestaOperacion{
			Mensaje: "No hay una llave SSH de root generada.",
		})

		return
	}

	if err != nil {
		slog.Error("descargar la llave de root", claveError, err)
		escribirJSON(w, http.StatusInternalServerError, respuestaOperacion{
			Mensaje: "No fue posible leer la llave privada.",
			Tecnico: err.Error(),
		})

		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+nombre+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(contenido); err != nil {
		slog.Error("escribir la llave de root", claveError, err)
	}
}

// configuracionIssue es el cuerpo de GET /api/issue.
type configuracionIssue struct {
	Mensaje    string `json:"mensaje"`
	NombreHost bool   `json:"nombre_host"`
	Fecha      bool   `json:"fecha"`
	Hora       bool   `json:"hora"`
	IP         bool   `json:"ip"`
	Contenido  string `json:"contenido,omitempty"`
}

// Issue atiende GET y PUT /api/issue: consulta o reemplaza el banner
// previo al inicio de sesión.
func (h *Sistema) Issue(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet, http.MethodPut) {
		return
	}

	if r.Method == http.MethodGet {
		h.obtenerIssue(w, r)

		return
	}

	var solicitud solicitudIssue
	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarMensajeIssue(solicitud.Mensaje); err != nil {
		responderValidacion(w, err)

		return
	}

	if !hayOpcionesIssue(solicitud) {
		responderValidacion(w, campoInvalido(campoMensaje, "indique un mensaje o al menos una opción"))

		return
	}

	resultado, err := h.issue.Escribir(r.Context(), services.ConfiguracionIssue{
		Mensaje:    solicitud.Mensaje,
		NombreHost: solicitud.NombreHost,
		Fecha:      solicitud.Fecha,
		Hora:       solicitud.Hora,
		IP:         solicitud.IP,
	})
	if err != nil {
		responderErrorOperacion(w, resultado, err)

		return
	}

	escribirResultado(w, http.StatusOK, resultado)
}

// hayOpcionesIssue informa si la solicitud incluye algún elemento.
func hayOpcionesIssue(solicitud solicitudIssue) bool {
	return strings.TrimSpace(solicitud.Mensaje) != "" ||
		solicitud.NombreHost || solicitud.Fecha || solicitud.Hora || solicitud.IP
}

// obtenerIssue responde la configuración actual de /etc/issue.
func (h *Sistema) obtenerIssue(w http.ResponseWriter, r *http.Request) {
	config, err := h.issue.Obtener(r.Context())
	if err != nil {
		responderErrorOperacion(w, services.Resultado{}, err)

		return
	}

	escribirJSON(w, http.StatusOK, configuracionIssue{
		Mensaje:    config.Mensaje,
		NombreHost: config.NombreHost,
		Fecha:      config.Fecha,
		Hora:       config.Hora,
		IP:         config.IP,
		Contenido:  config.Contenido,
	})
}

// Hostname atiende GET y PUT /api/hostname: consulta o reemplaza el
// nombre de host y su entrada en /etc/hosts.
func (h *Sistema) Hostname(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet, http.MethodPut) {
		return
	}

	if r.Method == http.MethodGet {
		noImplementado(w, "consultar el nombre de host")

		return
	}

	var solicitud solicitudHostname
	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarHostname(solicitud.Nombre); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarIPv4(campoIP, solicitud.IP); err != nil {
		responderValidacion(w, err)

		return
	}

	noImplementado(w, "asignar el nombre de host")
}
