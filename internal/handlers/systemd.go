// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"errors"
	"net/http"

	"consola-vm/internal/services"
)

// ServicioSystemd ejecuta las operaciones sobre las unidades de systemd.
type ServicioSystemd interface {
	// Listar devuelve las unidades con su estado.
	Listar(ctx context.Context) ([]services.EstadoUnidad, error)
	// Obtener devuelve el estado de una unidad.
	Obtener(ctx context.Context, unidad string) (services.EstadoUnidad, error)
	// Desplegar transfiere los archivos de la unidad.
	Desplegar(ctx context.Context, unidad string, archivos []services.ArchivoDespliegue, usuario, grupo string) (services.Resultado, error)
	// CambiarEstado activa, desactiva, habilita o deshabilita la unidad.
	CambiarEstado(ctx context.Context, unidad string, activo, habilitado *bool) (services.Resultado, error)
	// Reiniciar reinicia la unidad.
	Reiniciar(ctx context.Context, unidad string) (services.Resultado, error)
}

// Systemd agrupa los recursos de gestión de unidades con systemd
// (numeral 3.3).
type Systemd struct {
	servicio ServicioSystemd
}

// NuevoSystemd construye el handler de gestión de unidades systemd.
func NuevoSystemd(servicio ServicioSystemd) *Systemd {
	return &Systemd{servicio: servicio}
}

// archivoDespliegue describe un archivo que se transfiere a la máquina.
type archivoDespliegue struct {
	Origen  string `json:"origen"`
	Destino string `json:"destino"`
	Modo    string `json:"modo"`
}

// solicitudDespliegue es el cuerpo de PUT /api/units/{unit}.
type solicitudDespliegue struct {
	Archivos []archivoDespliegue `json:"archivos"`
	Usuario  string              `json:"usuario"`
	Grupo    string              `json:"grupo"`
}

// solicitudEstadoUnidad es el cuerpo de PATCH /api/units/{unit}: indica
// exactamente una de las dos propiedades.
type solicitudEstadoUnidad struct {
	Activo     *bool `json:"activo"`
	Habilitado *bool `json:"habilitado"`
}

// unidadRespuesta es una unidad en la respuesta JSON.
type unidadRespuesta struct {
	Unidad     string `json:"unidad"`
	Activo     bool   `json:"activo"`
	Habilitado bool   `json:"habilitado"`
	Estado     string `json:"estado"`
	Detalle    string `json:"detalle,omitempty"`
}

// listaUnidades es el cuerpo de GET /api/units.
type listaUnidades struct {
	Unidades []unidadRespuesta `json:"unidades"`
}

// Unidades atiende GET /api/units: lista las unidades administradas.
func (h *Systemd) Unidades(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet) {
		return
	}

	unidades, err := h.servicio.Listar(r.Context())
	if err != nil {
		responderErrorOperacion(w, services.Resultado{}, err)

		return
	}

	respuesta := listaUnidades{Unidades: make([]unidadRespuesta, 0, len(unidades))}
	for _, unidad := range unidades {
		respuesta.Unidades = append(respuesta.Unidades, respuestaDeUnidad(unidad))
	}

	escribirJSON(w, http.StatusOK, respuesta)
}

// Unidad atiende GET, PUT y PATCH /api/units/{unit}: consulta la
// unidad, despliega sus archivos o cambia su estado.
func (h *Systemd) Unidad(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet, http.MethodPut, http.MethodPatch) {
		return
	}

	unidad := r.PathValue(campoUnit)
	if err := validarUnidad(unidad); err != nil {
		responderValidacion(w, err)

		return
	}

	switch r.Method {
	case http.MethodGet:
		h.obtenerUnidad(w, r, unidad)
	case http.MethodPut:
		h.desplegarUnidad(w, r, unidad)
	case http.MethodPatch:
		h.cambiarEstadoUnidad(w, r, unidad)
	}
}

// obtenerUnidad responde el estado de la unidad.
func (h *Systemd) obtenerUnidad(w http.ResponseWriter, r *http.Request, unidad string) {
	estado, err := h.servicio.Obtener(r.Context(), unidad)
	if h.responderNoEncontrada(w, unidad, err) {
		return
	}

	if err != nil {
		responderErrorOperacion(w, services.Resultado{}, err)

		return
	}

	escribirJSON(w, http.StatusOK, respuestaDeUnidad(estado))
}

// desplegarUnidad valida y transfiere la unidad y sus archivos.
func (h *Systemd) desplegarUnidad(w http.ResponseWriter, r *http.Request, unidad string) {
	var solicitud solicitudDespliegue
	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarArchivos(solicitud.Archivos); err != nil {
		responderValidacion(w, err)

		return
	}

	if solicitud.Usuario != "" {
		if err := validarUsuario("usuario", solicitud.Usuario); err != nil {
			responderValidacion(w, err)

			return
		}
	}

	if solicitud.Grupo != "" {
		if err := validarUsuario("grupo", solicitud.Grupo); err != nil {
			responderValidacion(w, err)

			return
		}
	}

	archivos := make([]services.ArchivoDespliegue, 0, len(solicitud.Archivos))
	for _, archivo := range solicitud.Archivos {
		archivos = append(archivos, services.ArchivoDespliegue{
			Origen:  archivo.Origen,
			Destino: archivo.Destino,
			Modo:    archivo.Modo,
		})
	}

	resultado, err := h.servicio.Desplegar(r.Context(), unidad, archivos, solicitud.Usuario, solicitud.Grupo)
	if err != nil {
		responderErrorOperacion(w, resultado, err)

		return
	}

	escribirResultado(w, http.StatusOK, resultado)
}

// cambiarEstadoUnidad aplica el cambio y responde el estado resultante.
func (h *Systemd) cambiarEstadoUnidad(w http.ResponseWriter, r *http.Request, unidad string) {
	var solicitud solicitudEstadoUnidad
	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarEstadoUnidad(solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	resultado, err := h.servicio.CambiarEstado(r.Context(), unidad, solicitud.Activo, solicitud.Habilitado)
	if h.responderNoEncontrada(w, unidad, err) {
		return
	}

	if err != nil {
		responderErrorOperacion(w, resultado, err)

		return
	}

	estado, err := h.servicio.Obtener(r.Context(), unidad)
	if err != nil {
		responderErrorOperacion(w, services.Resultado{}, err)

		return
	}

	escribirJSON(w, http.StatusOK, respuestaDeUnidad(estado))
}

// validarEstadoUnidad exige exactamente una de las dos propiedades.
func validarEstadoUnidad(solicitud solicitudEstadoUnidad) error {
	if solicitud.Activo == nil && solicitud.Habilitado == nil {
		return campoInvalido(campoActivo, "debe indicar «activo» o «habilitado»")
	}

	if solicitud.Activo != nil && solicitud.Habilitado != nil {
		return campoInvalido(campoHabilitado, "indique solo «activo» o «habilitado»")
	}

	return nil
}

// validarArchivos comprueba que haya al menos un archivo y que cada uno
// tenga origen, destino absoluto y modo octal.
func validarArchivos(archivos []archivoDespliegue) error {
	if len(archivos) == 0 {
		return campoInvalido(campoArchivos, "debe indicar al menos un archivo")
	}

	for _, archivo := range archivos {
		if archivo.Origen == "" {
			return campoInvalido(campoOrigen, "no puede estar vacío")
		}

		if err := validarRuta(campoDestino, archivo.Destino); err != nil {
			return err
		}

		if err := validarModo(campoModo, archivo.Modo); err != nil {
			return err
		}
	}

	return nil
}

// Reinicios atiende POST /api/units/{unit}/restarts: reinicia la
// unidad indicada.
func (h *Systemd) Reinicios(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodPost) {
		return
	}

	unidad := r.PathValue(campoUnit)
	if err := validarUnidad(unidad); err != nil {
		responderValidacion(w, err)

		return
	}

	resultado, err := h.servicio.Reiniciar(r.Context(), unidad)
	if h.responderNoEncontrada(w, unidad, err) {
		return
	}

	if err != nil {
		responderErrorOperacion(w, resultado, err)

		return
	}

	escribirResultado(w, http.StatusAccepted, resultado)
}

// responderNoEncontrada responde 404 si el error indica que la unidad no
// existe y devuelve true en ese caso.
func (h *Systemd) responderNoEncontrada(w http.ResponseWriter, unidad string, err error) bool {
	if !errors.Is(err, services.ErrUnidadNoEncontrada) {
		return false
	}

	escribirJSON(w, http.StatusNotFound, respuestaError{
		Mensaje: "La unidad «" + unidad + "» no existe.",
		Tecnico: err.Error(),
	})

	return true
}

// respuestaDeUnidad convierte el estado del dominio en la respuesta.
func respuestaDeUnidad(estado services.EstadoUnidad) unidadRespuesta {
	return unidadRespuesta{
		Unidad:     estado.Unidad,
		Activo:     estado.Activo,
		Habilitado: estado.Habilitado,
		Estado:     estado.Estado,
		Detalle:    estado.Detalle,
	}
}
