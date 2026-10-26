// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"net/http"
	"testing"
)

const (
	descriptionNano       = "Instalar el paquete «nano»"
	descriptionReinicio   = "Reiniciar la unidad «catalogo.service»"
	pathUnitTests         = "/api/units/catalogo.service"
	pathRegistrationTests = "/api/session/register"
)

// TestDescribeOperation comprueba los resúmenes legibles del historial.
func TestDescribeOperation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		method   string
		pattern  string
		path     string
		data     map[string]any
		expected string
	}{
		{
			name:     "consultar sesión",
			method:   http.MethodGet,
			path:     pathSession,
			expected: "Consultar el estado de la sesión",
		},
		{
			name:     "diagnosticar sesión",
			method:   http.MethodGet,
			path:     "/api/session/diagnostico",
			expected: "Diagnosticar el registro de la máquina",
		},
		{
			name:     "listar máquinas",
			method:   http.MethodGet,
			path:     "/api/vms",
			expected: "Listar las máquinas virtuales",
		},
		{
			name:     "listar paquetes",
			method:   http.MethodGet,
			path:     pathPackages,
			expected: "Listar los paquetes instalados",
		},
		{
			name:     "consultar paquete",
			method:   http.MethodGet,
			path:     pathPackages + "/nano",
			expected: "Consultar el paquete «nano»",
		},
		{
			name:     "listar usuarios",
			method:   http.MethodGet,
			path:     pathUsers,
			expected: "Listar los usuarios",
		},
		{
			name:     "descargar llave de usuario",
			method:   http.MethodGet,
			path:     pathUsers + "/jorge/key",
			expected: "Descargar la llave privada de «jorge»",
		},
		{
			name:     "listar unidades",
			method:   http.MethodGet,
			path:     pathUnits,
			expected: "Listar las unidades de systemd",
		},
		{
			name:     "listar servidores",
			method:   http.MethodGet,
			path:     pathBackend,
			expected: "Listar los servidores del back-end",
		},
		{
			name:     "consultar historial",
			method:   http.MethodGet,
			path:     "/api/historial",
			expected: "Consultar el historial de operaciones",
		},
		{
			name:     "instalar paquete",
			method:   http.MethodPost,
			path:     pathPackages,
			data:     map[string]any{keyName: "nano"},
			expected: descriptionNano,
		},
		{
			name:     "instalar paquete sin nombre",
			method:   http.MethodPost,
			path:     pathPackages,
			expected: "Instalar un paquete",
		},
		{
			name:     "eliminar paquete",
			method:   http.MethodDelete,
			path:     "/api/packages/nano%2Bplus",
			expected: "Eliminar el paquete «nano+plus»",
		},
		{
			name:     "llave de root",
			method:   http.MethodPost,
			path:     "/api/keys/root",
			expected: "Generar la llave SSH de root",
		},
		{
			name:     "personalizar issue",
			method:   http.MethodPut,
			path:     pathIssue,
			expected: "Personalizar /etc/issue",
		},
		{
			name:     "cambiar hostname",
			method:   http.MethodPut,
			path:     "/api/hostname",
			data:     map[string]any{keyName: "nodo-1"},
			expected: "Cambiar el nombre de host a «nodo-1»",
		},
		{
			name:     "crear usuario",
			method:   http.MethodPost,
			path:     "/api/users",
			data:     map[string]any{keyName: "jorge"},
			expected: "Crear el usuario «jorge»",
		},
		{
			name:     "crear usuario sin nombre",
			method:   http.MethodPost,
			path:     "/api/users",
			expected: "Crear un usuario",
		},
		{
			name:     "eliminar usuario",
			method:   http.MethodDelete,
			path:     "/api/users/jorge",
			expected: "Eliminar el usuario «jorge»",
		},
		{
			name:     "eliminar usuario con archivos",
			method:   http.MethodDelete,
			path:     "/api/users/jorge",
			data:     map[string]any{"archivos": true},
			expected: "Eliminar el usuario «jorge» y sus archivos",
		},
		{
			name:     "desplegar unidad",
			method:   http.MethodPut,
			path:     pathUnitTests,
			expected: "Desplegar la unidad «catalogo.service»",
		},
		{
			name:     "activar unidad",
			method:   http.MethodPatch,
			path:     pathUnitTests,
			data:     map[string]any{"activo": true},
			expected: "Activar la unidad «catalogo.service»",
		},
		{
			name:     "detener unidad",
			method:   http.MethodPatch,
			path:     pathUnitTests,
			data:     map[string]any{"activo": false},
			expected: "Detener la unidad «catalogo.service»",
		},
		{
			name:     "habilitar unidad",
			method:   http.MethodPatch,
			path:     pathUnitTests,
			data:     map[string]any{"habilitado": true},
			expected: "Habilitar el arranque de la unidad «catalogo.service»",
		},
		{
			name:     "deshabilitar unidad",
			method:   http.MethodPatch,
			path:     pathUnitTests,
			data:     map[string]any{"habilitado": false},
			expected: "Deshabilitar el arranque de la unidad «catalogo.service»",
		},
		{
			name:     "estado de unidad sin datos",
			method:   http.MethodPatch,
			path:     pathUnitTests,
			expected: "Cambiar el estado de la unidad «catalogo.service»",
		},
		{
			name:     "reiniciar unidad",
			method:   http.MethodPost,
			path:     "/api/units/catalogo.service/restarts",
			expected: descriptionReinicio,
		},
		{
			name:     "agregar servidor",
			method:   http.MethodPost,
			path:     pathBackend,
			data:     map[string]any{"direccion": "10.0.0.5", "puerto": float64(8080)},
			expected: "Agregar el servidor «10.0.0.5:8080» al back-end",
		},
		{
			name:     "agregar servidor sin datos",
			method:   http.MethodPost,
			path:     pathBackend,
			expected: "Agregar un servidor al back-end",
		},
		{
			name:     "eliminar servidor",
			method:   http.MethodDelete,
			path:     "/api/haproxy/backend/servers/10.0.0.5%3A8080",
			expected: "Eliminar el servidor «10.0.0.5:8080» del back-end",
		},
		{
			name:     "registrar máquina",
			method:   http.MethodPost,
			path:     pathRegistrationTests,
			data:     map[string]any{keyVM: testMachineName, "usuario": "jorge"},
			expected: "Registrar la máquina «vm-1» (usuario jorge)",
		},
		{
			name:     "registrar máquina sin usuario",
			method:   http.MethodPost,
			path:     pathRegistrationTests,
			data:     map[string]any{keyVM: testMachineName},
			expected: "Registrar la máquina «vm-1»",
		},
		{
			name:     "registrar máquina sin datos",
			method:   http.MethodPost,
			path:     pathRegistrationTests,
			expected: "Registrar una máquina",
		},
		{
			name:     "olvidar registro",
			method:   http.MethodDelete,
			path:     pathRegistrationTests,
			expected: "Olvidar el registro de la máquina",
		},
		{
			name:     "encender máquina",
			method:   http.MethodPost,
			path:     "/api/vms/ubuntu-lab/poweron",
			expected: "Encender la máquina «ubuntu-lab»",
		},
		{
			name:     "vincular máquina",
			method:   http.MethodPut,
			path:     pathSession,
			data:     map[string]any{keyVM: testMachineName},
			expected: "Vincular la máquina «vm-1»",
		},
		{
			name:     "ruta desconocida con patrón",
			method:   http.MethodPost,
			pattern:  "/api/desconocida",
			path:     "/api/desconocida",
			expected: "POST /api/desconocida",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			obtained := DescribeOperation(testCase.method, testCase.pattern, testCase.path, testCase.data)
			if obtained != testCase.expected {
				t.Errorf("DescribirOperacion() = %q, want %q", obtained, testCase.expected)
			}
		})
	}
}
