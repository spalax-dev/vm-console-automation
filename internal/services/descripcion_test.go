// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"net/http"
	"testing"
)

const (
	descripcionNano     = "Instalar el paquete «nano»"
	descripcionReinicio = "Reiniciar la unidad «catalogo.service»"
	rutaUnidadPrueba    = "/api/units/catalogo.service"
	rutaRegistroPrueba  = "/api/session/register"
)

// TestDescribirOperacion comprueba los resúmenes legibles del historial.
func TestDescribirOperacion(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre   string
		metodo   string
		patron   string
		ruta     string
		datos    map[string]any
		esperado string
	}{
		{
			nombre:   "consultar sesión",
			metodo:   http.MethodGet,
			ruta:     rutaSesion,
			esperado: "Consultar el estado de la sesión",
		},
		{
			nombre:   "diagnosticar sesión",
			metodo:   http.MethodGet,
			ruta:     "/api/session/diagnostico",
			esperado: "Diagnosticar el registro de la máquina",
		},
		{
			nombre:   "listar máquinas",
			metodo:   http.MethodGet,
			ruta:     "/api/vms",
			esperado: "Listar las máquinas virtuales",
		},
		{
			nombre:   "listar paquetes",
			metodo:   http.MethodGet,
			ruta:     rutaPaquetes,
			esperado: "Listar los paquetes instalados",
		},
		{
			nombre:   "consultar paquete",
			metodo:   http.MethodGet,
			ruta:     rutaPaquetes + "/nano",
			esperado: "Consultar el paquete «nano»",
		},
		{
			nombre:   "listar usuarios",
			metodo:   http.MethodGet,
			ruta:     rutaUsuarios,
			esperado: "Listar los usuarios",
		},
		{
			nombre:   "descargar llave de usuario",
			metodo:   http.MethodGet,
			ruta:     rutaUsuarios + "/jorge/key",
			esperado: "Descargar la llave privada de «jorge»",
		},
		{
			nombre:   "listar unidades",
			metodo:   http.MethodGet,
			ruta:     rutaUnidades,
			esperado: "Listar las unidades de systemd",
		},
		{
			nombre:   "listar servidores",
			metodo:   http.MethodGet,
			ruta:     rutaBackend,
			esperado: "Listar los servidores del back-end",
		},
		{
			nombre:   "consultar historial",
			metodo:   http.MethodGet,
			ruta:     "/api/historial",
			esperado: "Consultar el historial de operaciones",
		},
		{
			nombre:   "instalar paquete",
			metodo:   http.MethodPost,
			ruta:     rutaPaquetes,
			datos:    map[string]any{claveNombre: "nano"},
			esperado: descripcionNano,
		},
		{
			nombre:   "instalar paquete sin nombre",
			metodo:   http.MethodPost,
			ruta:     rutaPaquetes,
			esperado: "Instalar un paquete",
		},
		{
			nombre:   "eliminar paquete",
			metodo:   http.MethodDelete,
			ruta:     "/api/packages/nano%2Bplus",
			esperado: "Eliminar el paquete «nano+plus»",
		},
		{
			nombre:   "llave de root",
			metodo:   http.MethodPost,
			ruta:     "/api/keys/root",
			esperado: "Generar la llave SSH de root",
		},
		{
			nombre:   "personalizar issue",
			metodo:   http.MethodPut,
			ruta:     rutaIssue,
			esperado: "Personalizar /etc/issue",
		},
		{
			nombre:   "cambiar hostname",
			metodo:   http.MethodPut,
			ruta:     "/api/hostname",
			datos:    map[string]any{claveNombre: "nodo-1"},
			esperado: "Cambiar el nombre de host a «nodo-1»",
		},
		{
			nombre:   "crear usuario",
			metodo:   http.MethodPost,
			ruta:     "/api/users",
			datos:    map[string]any{claveNombre: "jorge"},
			esperado: "Crear el usuario «jorge»",
		},
		{
			nombre:   "crear usuario sin nombre",
			metodo:   http.MethodPost,
			ruta:     "/api/users",
			esperado: "Crear un usuario",
		},
		{
			nombre:   "eliminar usuario",
			metodo:   http.MethodDelete,
			ruta:     "/api/users/jorge",
			esperado: "Eliminar el usuario «jorge»",
		},
		{
			nombre:   "eliminar usuario con archivos",
			metodo:   http.MethodDelete,
			ruta:     "/api/users/jorge",
			datos:    map[string]any{"archivos": true},
			esperado: "Eliminar el usuario «jorge» y sus archivos",
		},
		{
			nombre:   "desplegar unidad",
			metodo:   http.MethodPut,
			ruta:     rutaUnidadPrueba,
			esperado: "Desplegar la unidad «catalogo.service»",
		},
		{
			nombre:   "activar unidad",
			metodo:   http.MethodPatch,
			ruta:     rutaUnidadPrueba,
			datos:    map[string]any{"activo": true},
			esperado: "Activar la unidad «catalogo.service»",
		},
		{
			nombre:   "detener unidad",
			metodo:   http.MethodPatch,
			ruta:     rutaUnidadPrueba,
			datos:    map[string]any{"activo": false},
			esperado: "Detener la unidad «catalogo.service»",
		},
		{
			nombre:   "habilitar unidad",
			metodo:   http.MethodPatch,
			ruta:     rutaUnidadPrueba,
			datos:    map[string]any{"habilitado": true},
			esperado: "Habilitar el arranque de la unidad «catalogo.service»",
		},
		{
			nombre:   "deshabilitar unidad",
			metodo:   http.MethodPatch,
			ruta:     rutaUnidadPrueba,
			datos:    map[string]any{"habilitado": false},
			esperado: "Deshabilitar el arranque de la unidad «catalogo.service»",
		},
		{
			nombre:   "estado de unidad sin datos",
			metodo:   http.MethodPatch,
			ruta:     rutaUnidadPrueba,
			esperado: "Cambiar el estado de la unidad «catalogo.service»",
		},
		{
			nombre:   "reiniciar unidad",
			metodo:   http.MethodPost,
			ruta:     "/api/units/catalogo.service/restarts",
			esperado: descripcionReinicio,
		},
		{
			nombre:   "agregar servidor",
			metodo:   http.MethodPost,
			ruta:     rutaBackend,
			datos:    map[string]any{"direccion": "10.0.0.5", "puerto": float64(8080)},
			esperado: "Agregar el servidor «10.0.0.5:8080» al back-end",
		},
		{
			nombre:   "agregar servidor sin datos",
			metodo:   http.MethodPost,
			ruta:     rutaBackend,
			esperado: "Agregar un servidor al back-end",
		},
		{
			nombre:   "eliminar servidor",
			metodo:   http.MethodDelete,
			ruta:     "/api/haproxy/backend/servers/10.0.0.5%3A8080",
			esperado: "Eliminar el servidor «10.0.0.5:8080» del back-end",
		},
		{
			nombre:   "registrar máquina",
			metodo:   http.MethodPost,
			ruta:     rutaRegistroPrueba,
			datos:    map[string]any{claveVM: nombreDebian, "usuario": "jorge"},
			esperado: "Registrar la máquina «debian-1» (usuario jorge)",
		},
		{
			nombre:   "registrar máquina sin usuario",
			metodo:   http.MethodPost,
			ruta:     rutaRegistroPrueba,
			datos:    map[string]any{claveVM: nombreDebian},
			esperado: "Registrar la máquina «debian-1»",
		},
		{
			nombre:   "registrar máquina sin datos",
			metodo:   http.MethodPost,
			ruta:     rutaRegistroPrueba,
			esperado: "Registrar una máquina",
		},
		{
			nombre:   "olvidar registro",
			metodo:   http.MethodDelete,
			ruta:     rutaRegistroPrueba,
			esperado: "Olvidar el registro de la máquina",
		},
		{
			nombre:   "encender máquina",
			metodo:   http.MethodPost,
			ruta:     "/api/vms/ubuntu-lab/poweron",
			esperado: "Encender la máquina «ubuntu-lab»",
		},
		{
			nombre:   "vincular máquina",
			metodo:   http.MethodPut,
			ruta:     rutaSesion,
			datos:    map[string]any{claveVM: nombreDebian},
			esperado: "Vincular la máquina «debian-1»",
		},
		{
			nombre:   "ruta desconocida con patrón",
			metodo:   http.MethodPost,
			patron:   "/api/desconocida",
			ruta:     "/api/desconocida",
			esperado: "POST /api/desconocida",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			obtenido := DescribirOperacion(caso.metodo, caso.patron, caso.ruta, caso.datos)
			if obtenido != caso.esperado {
				t.Errorf("DescribirOperacion() = %q, want %q", obtenido, caso.esperado)
			}
		})
	}
}
