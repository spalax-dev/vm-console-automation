// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"net/http"
	"testing"
)

// TestUnidades cubre el listado de unidades administradas.
func TestUnidades(t *testing.T) {
	t.Parallel()

	handler := NuevoSystemd().Unidades
	casos := []casoPeticion{
		{nombre: casoListar, metodo: http.MethodGet, codigo: http.StatusNotImplemented},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodPost, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, "/api/units", caso)
		comprobarCaso(t, rec, caso)
	}
}

// TestUnidad cubre la consulta, el despliegue y el cambio de estado de
// una unidad.
func TestUnidad(t *testing.T) {
	t.Parallel()

	handler := NuevoSystemd().Unidad
	casos := []struct {
		casoPeticion
		unidad string
	}{
		{casoPeticion{nombre: casoConsultar, metodo: http.MethodGet, codigo: http.StatusNotImplemented}, unidadCatalogo},
		{
			casoPeticion{
				nombre: "desplegar",
				metodo: http.MethodPut,
				cuerpo: `{"archivos":[{"origen":"./app/catalogo","destino":"/opt/catalogo/catalogo","modo":"755"}]}`,
				codigo: http.StatusNotImplemented,
			},
			unidadCatalogo,
		},
		{
			casoPeticion{
				nombre: "desplegar sin archivos",
				metodo: http.MethodPut,
				cuerpo: `{"archivos":[]}`,
				codigo: http.StatusBadRequest,
				campo:  campoArchivos,
			},
			unidadCatalogo,
		},
		{
			casoPeticion{
				nombre: "desplegar con destino relativo",
				metodo: http.MethodPut,
				cuerpo: `{"archivos":[{"origen":"./app","destino":"opt/app","modo":"755"}]}`,
				codigo: http.StatusBadRequest,
				campo:  campoDestino,
			},
			unidadCatalogo,
		},
		{
			casoPeticion{
				nombre: "activar",
				metodo: http.MethodPatch,
				cuerpo: `{"activo":true}`,
				codigo: http.StatusNotImplemented,
			},
			unidadCatalogo,
		},
		{
			casoPeticion{
				nombre: "deshabilitar",
				metodo: http.MethodPatch,
				cuerpo: `{"habilitado":false}`,
				codigo: http.StatusNotImplemented,
			},
			unidadCatalogo,
		},
		{
			casoPeticion{
				nombre: "sin estado",
				metodo: http.MethodPatch,
				cuerpo: `{}`,
				codigo: http.StatusBadRequest,
				campo:  campoActivo,
			},
			unidadCatalogo,
		},
		{
			casoPeticion{
				nombre: "dos estados",
				metodo: http.MethodPatch,
				cuerpo: `{"activo":true,"habilitado":true}`,
				codigo: http.StatusBadRequest,
				campo:  campoHabilitado,
			},
			unidadCatalogo,
		},
		{
			casoPeticion{
				nombre: "desplegar con cuerpo inválido",
				metodo: http.MethodPut,
				cuerpo: `{`,
				codigo: http.StatusBadRequest,
			},
			unidadCatalogo,
		},
		{
			casoPeticion{
				nombre: "desplegar con modo inválido",
				metodo: http.MethodPut,
				cuerpo: `{"archivos":[{"origen":"./app","destino":"/opt/app","modo":"rwx"}]}`,
				codigo: http.StatusBadRequest,
				campo:  campoModo,
			},
			unidadCatalogo,
		},
		{
			casoPeticion{
				nombre: "estado con cuerpo inválido",
				metodo: http.MethodPatch,
				cuerpo: `{`,
				codigo: http.StatusBadRequest,
			},
			unidadCatalogo,
		},
		{casoPeticion{nombre: "unidad inválida", metodo: http.MethodGet, codigo: http.StatusBadRequest, campo: campoUnit}, "../etc/passwd"},
		{casoPeticion{nombre: casoMetodoNoPermitido, metodo: http.MethodDelete, codigo: http.StatusMethodNotAllowed}, unidadCatalogo},
	}

	for _, caso := range casos {
		rec := ejecutarCasoConPath(t, handler, "/api/units/x", "unit", caso.unidad, caso.casoPeticion)
		comprobarCaso(t, rec, caso.casoPeticion)
	}
}

// TestReinicios cubre el reinicio de una unidad.
func TestReinicios(t *testing.T) {
	t.Parallel()

	handler := NuevoSystemd().Reinicios
	casos := []struct {
		casoPeticion
		unidad string
	}{
		{casoPeticion{nombre: "reiniciar", metodo: http.MethodPost, codigo: http.StatusNotImplemented}, unidadCatalogo},
		{casoPeticion{nombre: "unidad inválida", metodo: http.MethodPost, codigo: http.StatusBadRequest, campo: campoUnit}, "catalogo"},
		{casoPeticion{nombre: casoMetodoNoPermitido, metodo: http.MethodGet, codigo: http.StatusMethodNotAllowed}, unidadCatalogo},
	}

	for _, caso := range casos {
		rec := ejecutarCasoConPath(t, handler, "/api/units/x/restarts", "unit", caso.unidad, caso.casoPeticion)
		comprobarCaso(t, rec, caso.casoPeticion)
	}
}
