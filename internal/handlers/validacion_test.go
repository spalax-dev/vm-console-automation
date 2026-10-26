// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestValidatePkg comprueba la regla de nombres de paquete APT.
func TestValidatePkg(t *testing.T) {
	t.Parallel()

	valid := []string{nameNano, "nginx", "python3-pip", "g++", "libc6-dev"}
	for _, value := range valid {
		if err := validatePkg(nameField, value); err != nil {
			t.Errorf("validarPaquete(%q) error = %v, want nil", value, err)
		}
	}

	invalid := []string{"", "Nano", "nano; rm -rf /", "-nano", "nano/../x"}
	for _, value := range invalid {
		if err := validatePkg(nameField, value); err == nil {
			t.Errorf("validarPaquete(%q) error = nil, want un error", value)
		}
	}
}

// TestValidateUser comprueba la regla de nombres de usuario POSIX.
func TestValidateUser(t *testing.T) {
	t.Parallel()

	valid := []string{userJorge, "_servicio", "andrey_t", "soporte-1"}
	for _, value := range valid {
		if err := validateUser(nameField, value); err != nil {
			t.Errorf("validarUsuario(%q) error = %v, want nil", value, err)
		}
	}

	invalid := []string{"", userJorgeMayus, "1jorge", "josé", strings.Repeat("a", 33)}
	for _, value := range invalid {
		if err := validateUser(nameField, value); err == nil {
			t.Errorf("validarUsuario(%q) error = nil, want un error", value)
		}
	}
}

// TestValidateHostname comprueba la regla RFC 1123.
func TestValidateHostname(t *testing.T) {
	t.Parallel()

	valid := []string{hostTest, "nodo-web-01", "a"}
	for _, value := range valid {
		if err := validateHostname(value); err != nil {
			t.Errorf("validarHostname(%q) error = %v, want nil", value, err)
		}
	}

	invalid := []string{"", "-inicio", "mi equipo", "NODO", strings.Repeat("a", 64)}
	for _, value := range invalid {
		if err := validateHostname(value); err == nil {
			t.Errorf("validarHostname(%q) error = nil, want un error", value)
		}
	}
}

// TestValidateUnit comprueba la regla de nombres de unidad systemd.
func TestValidateUnit(t *testing.T) {
	t.Parallel()

	valid := []string{"catalogo.service", "vmcons-tarea.service", "getty@tty1.service"}
	for _, value := range valid {
		if err := validateUnit(value); err != nil {
			t.Errorf("validarUnidad(%q) error = %v, want nil", value, err)
		}
	}

	invalid := []string{"", propietarioCatalogo, "../etc/passwd.service", "catalogo.timer"}
	for _, value := range invalid {
		if err := validateUnit(value); err == nil {
			t.Errorf("validarUnidad(%q) error = nil, want un error", value)
		}
	}
}

// TestValidateServer comprueba el identificador direccion:puerto.
func TestValidateServer(t *testing.T) {
	t.Parallel()

	valid := []string{serverTests, "10.0.2.15:1", "10.0.2.15:65535"}
	for _, value := range valid {
		if _, err := validateServer(serverParam, value); err != nil {
			t.Errorf("validarServidor(%q) error = %v, want nil", value, err)
		}
	}

	invalid := []string{"", ipServer, "192.168.1.61:0", "localhost:8080", "[::1]:8080", "192.168.1.61:70000"}
	for _, value := range invalid {
		if _, err := validateServer(serverParam, value); err == nil {
			t.Errorf("validarServidor(%q) error = nil, want un error", value)
		}
	}
}

// TestValidatePath comprueba que solo se aceptan rutas absolutas sin
// traversal ni comodines.
func TestValidatePath(t *testing.T) {
	t.Parallel()

	valid := []string{"/opt/catalogo/app", "/etc/systemd/system/catalogo.service"}
	for _, value := range valid {
		if err := validatePath(destinationField, value); err != nil {
			t.Errorf("validarRuta(%q) error = %v, want nil", value, err)
		}
	}

	invalid := []string{"", "opt/app", "/opt/../etc/shadow", "/opt/app/*", "/opt/app/?", "/opt/app\x00oculto"}
	for _, value := range invalid {
		if err := validatePath(destinationField, value); err == nil {
			t.Errorf("validarRuta(%q) error = nil, want un error", value)
		}
	}
}

// TestValidateMode comprueba la regla de permisos en octal.
func TestValidateMode(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"644", "755", "0755"} {
		if err := validateMode(modeField, value); err != nil {
			t.Errorf("validarModo(%q) error = %v, want nil", value, err)
		}
	}

	for _, value := range []string{"", "rwx", "999", "75"} {
		if err := validateMode(modeField, value); err == nil {
			t.Errorf("validarModo(%q) error = nil, want un error", value)
		}
	}
}

// TestValidateMessageIssue comprueba el límite y la ausencia de NUL.
func TestValidateMessageIssue(t *testing.T) {
	t.Parallel()

	if err := validateMessageIssue("Bienvenido a la consola"); err != nil {
		t.Errorf("mensaje válido: error = %v, want nil", err)
	}

	if err := validateMessageIssue(strings.Repeat("a", 512)); err != nil {
		t.Errorf("mensaje de 512: error = %v, want nil", err)
	}

	if err := validateMessageIssue(strings.Repeat("a", 513)); err == nil {
		t.Error("mensaje de 513 caracteres: error = nil, want un error")
	}

	if err := validateMessageIssue("mensaje\x00oculto"); err == nil {
		t.Error("mensaje con NUL: error = nil, want un error")
	}
}

// TestValidateIPv4 comprueba la regla de direcciones IPv4.
func TestValidateIPv4(t *testing.T) {
	t.Parallel()

	valid := []string{"192.168.1.60", "10.0.2.15"}
	for _, value := range valid {
		if err := validateIPv4(ipField, value); err != nil {
			t.Errorf("validarIPv4(%q) error = %v, want nil", value, err)
		}
	}

	invalid := []string{"", "999.1.1.1", "localhost", "::1"}
	for _, value := range invalid {
		if err := validateIPv4(ipField, value); err == nil {
			t.Errorf("validarIPv4(%q) error = nil, want un error", value)
		}
	}
}

// TestValidatePort comprueba el rango 1–65535.
func TestValidatePort(t *testing.T) {
	t.Parallel()

	for _, value := range []int{1, 80, 65535} {
		if err := validatePort(value); err != nil {
			t.Errorf("validarPuerto(%d) error = %v, want nil", value, err)
		}
	}

	for _, value := range []int{0, -1, 65536} {
		if err := validatePort(value); err == nil {
			t.Errorf("validarPuerto(%d) error = nil, want un error", value)
		}
	}
}

// TestBooleanoQuery comprueba la lectura de booleanos de la query.
func TestBooleanoQuery(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		value string
		want  bool
		error bool
	}{
		{"", false, false},
		{"true", true, false},
		{"false", false, false},
		{"quizá", false, true},
	}

	for _, testCase := range testCases {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/users/jorge?files="+testCase.value, http.NoBody)

		obtained, err := booleanoQuery(req, filesParam)
		if (err != nil) != testCase.error {
			t.Errorf("booleanoQuery(%q) error = %v, want error=%v", testCase.value, err, testCase.error)

			continue
		}

		if obtained != testCase.want {
			t.Errorf("booleanoQuery(%q) = %v, want %v", testCase.value, obtained, testCase.want)
		}
	}
}
