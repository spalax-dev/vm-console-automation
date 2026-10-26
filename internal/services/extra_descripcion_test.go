// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"net/http"
	"testing"
)

// TestDescribeOperationCoversValuesByOmision recorre las ramas que
// no tienen un resumen específico.
func TestDescribeOperationCoversValuesByOmision(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		method  string
		pattern string
		path    string
		data    map[string]any
		esperar string
	}{
		{"paquete sin verbo", http.MethodPut, "", "/api/packages", nil, "Gestionar paquetes"},
		{"paquete por nombre", http.MethodGet, "", "/api/packages/nano", nil, "Consultar el paquete «nano»"},
		{"paquete sin nombre", http.MethodPost, "", "/api/packages", nil, "Instalar un paquete"},
		{"llave descargada", http.MethodGet, "", "/api/keys/root", nil, "Descargar la llave privada de root"},
		{"issue consultada", http.MethodGet, "", "/api/issue", nil, "Consultar /etc/issue"},
		{"hostname consultado", http.MethodGet, "", "/api/hostname", nil, "Consultar el nombre de host"},
		{"hostname sin nombre", http.MethodPut, "", "/api/hostname", nil, "Cambiar el nombre de host"},
		{"usuarios listados", http.MethodGet, "", "/api/users", nil, "Listar los usuarios"},
		{"usuario consultado", http.MethodGet, "", "/api/users/jorge", nil, "Consultar el usuario «jorge»"},
		{"usuario con otro verbo", http.MethodPut, "", "/api/users/jorge", nil, "Gestionar el usuario «jorge»"},
		{"usuario sin archivos", http.MethodDelete, "", "/api/users/jorge", nil, "Eliminar el usuario «jorge»"},
		{"unidades listadas", http.MethodGet, "", "/api/units", nil, "Listar las unidades de systemd"},
		{"unidad consultada", http.MethodGet, "", "/api/units/nano.service", nil, "Consultar la unidad «nano.service»"},
		{"unidad con otro verbo", http.MethodDelete, "", "/api/units/nano.service", nil, "Consultar la unidad «nano.service»"},
		{"unidad sin estado", http.MethodPatch, "", "/api/units/nano.service", nil, "Cambiar el estado de la unidad «nano.service»"},
		{"backend consultado", http.MethodPut, "", "/api/haproxy/backend/servers", nil, "Consultar el back-end"},
		{"backend sin destino", http.MethodPost, "", "/api/haproxy/backend/servers", nil, "Agregar un servidor al back-end"},
		{
			"backend con puerto", http.MethodPost, "", "/api/haproxy/backend/servers",
			map[string]any{"direccion": "192.168.1.61", "puerto": float64(8080)},
			"Agregar el servidor «192.168.1.61:8080» al back-end",
		},
		{
			"backend sin puerto", http.MethodPost, "", "/api/haproxy/backend/servers",
			map[string]any{"direccion": "192.168.1.61"},
			"Agregar el servidor «192.168.1.61» al back-end",
		},
		{"sesión sin diagnóstico", http.MethodGet, "", "/api/session", nil, "Consultar el estado de la sesión"},
		{"sesión con otro verbo", http.MethodPatch, "", "/api/session", nil, "Consultar la sesión"},
		{"vinculación sin máquina", http.MethodPut, "", "/api/session", nil, "Vincular una máquina"},
		{"registro sin máquina", http.MethodPost, "", "/api/session/register", nil, "Registrar una máquina"},
		{"ruta con patrón", http.MethodPatch, "/api/otro/{id}", "/api/otro/7", nil, "PATCH /api/otro/{id}"},
		{"ruta desconocida", http.MethodGet, "", "/api/raro", nil, "GET /api/raro"},
		{"historial con otro verbo", http.MethodDelete, "", "/api/historial", nil, "DELETE /api/historial"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := DescribeOperation(testCase.method, testCase.pattern, testCase.path, testCase.data)
			if result != testCase.esperar {
				t.Errorf("DescribirOperacion() = %q, want %q", result, testCase.esperar)
			}
		})
	}
}

// TestLastWithoutParts devuelve vacío para una ruta vacía.
func TestLastWithoutParts(t *testing.T) {
	t.Parallel()

	if last := last(nil); last != "" {
		t.Errorf("ultimo(nil) = %q, want vacío", last)
	}
}
