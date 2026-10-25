// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import "net/http"

// Systemd agrupa los recursos de gestión de unidades con systemd
// (numeral 3.3).
type Systemd struct{}

// NuevoSystemd construye el handler de gestión de unidades systemd.
func NuevoSystemd() *Systemd {
	return &Systemd{}
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

// Unidades atiende GET /api/units: lista las unidades administradas.
func (h *Systemd) Unidades(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet) {
		return
	}

	noImplementado(w, "listar las unidades administradas")
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
		noImplementado(w, "consultar el estado de "+unidad)
	case http.MethodPut:
		h.desplegar(w, r, unidad)
	case http.MethodPatch:
		h.cambiarEstado(w, r, unidad)
	}
}

// desplegar valida y transfiere la unidad y sus archivos.
func (h *Systemd) desplegar(w http.ResponseWriter, r *http.Request, unidad string) {
	var solicitud solicitudDespliegue
	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarArchivos(solicitud.Archivos); err != nil {
		responderValidacion(w, err)

		return
	}

	noImplementado(w, "desplegar la unidad "+unidad)
}

// cambiarEstado valida y aplica un cambio de estado sobre la unidad.
func (h *Systemd) cambiarEstado(w http.ResponseWriter, r *http.Request, unidad string) {
	var solicitud solicitudEstadoUnidad
	if err := decodificarJSON(w, r, &solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	if err := validarEstadoUnidad(solicitud); err != nil {
		responderValidacion(w, err)

		return
	}

	noImplementado(w, "cambiar el estado de la unidad "+unidad)
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

	noImplementado(w, "reiniciar la unidad "+unidad)
}
