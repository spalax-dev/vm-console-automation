// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestValidarPaquete comprueba la regla de nombres de paquete APT.
func TestValidarPaquete(t *testing.T) {
	t.Parallel()

	validos := []string{nombreNano, "nginx", "python3-pip", "g++", "libc6-dev"}
	for _, valor := range validos {
		if err := validarPaquete(campoNombre, valor); err != nil {
			t.Errorf("validarPaquete(%q) error = %v, want nil", valor, err)
		}
	}

	invalidos := []string{"", "Nano", "nano; rm -rf /", "-nano", "nano/../x"}
	for _, valor := range invalidos {
		if err := validarPaquete(campoNombre, valor); err == nil {
			t.Errorf("validarPaquete(%q) error = nil, want un error", valor)
		}
	}
}

// TestValidarUsuario comprueba la regla de nombres de usuario POSIX.
func TestValidarUsuario(t *testing.T) {
	t.Parallel()

	validos := []string{usuarioJorge, "_servicio", "andrey_t", "soporte-1"}
	for _, valor := range validos {
		if err := validarUsuario(campoNombre, valor); err != nil {
			t.Errorf("validarUsuario(%q) error = %v, want nil", valor, err)
		}
	}

	invalidos := []string{"", usuarioJorgeMayus, "1jorge", "josé", strings.Repeat("a", 33)}
	for _, valor := range invalidos {
		if err := validarUsuario(campoNombre, valor); err == nil {
			t.Errorf("validarUsuario(%q) error = nil, want un error", valor)
		}
	}
}

// TestValidarHostname comprueba la regla RFC 1123.
func TestValidarHostname(t *testing.T) {
	t.Parallel()

	validos := []string{hostDebian, "nodo-web-01", "a"}
	for _, valor := range validos {
		if err := validarHostname(valor); err != nil {
			t.Errorf("validarHostname(%q) error = %v, want nil", valor, err)
		}
	}

	invalidos := []string{"", "-inicio", "mi equipo", "NODO", strings.Repeat("a", 64)}
	for _, valor := range invalidos {
		if err := validarHostname(valor); err == nil {
			t.Errorf("validarHostname(%q) error = nil, want un error", valor)
		}
	}
}

// TestValidarUnidad comprueba la regla de nombres de unidad systemd.
func TestValidarUnidad(t *testing.T) {
	t.Parallel()

	validos := []string{"catalogo.service", "vmcons-tarea.service", "getty@tty1.service"}
	for _, valor := range validos {
		if err := validarUnidad(valor); err != nil {
			t.Errorf("validarUnidad(%q) error = %v, want nil", valor, err)
		}
	}

	invalidos := []string{"", propietarioCatalogo, "../etc/passwd.service", "catalogo.timer"}
	for _, valor := range invalidos {
		if err := validarUnidad(valor); err == nil {
			t.Errorf("validarUnidad(%q) error = nil, want un error", valor)
		}
	}
}

// TestValidarServidor comprueba el identificador direccion:puerto.
func TestValidarServidor(t *testing.T) {
	t.Parallel()

	validos := []string{servidorPrueba, "10.0.2.15:1", "10.0.2.15:65535"}
	for _, valor := range validos {
		if err := validarServidor(campoServer, valor); err != nil {
			t.Errorf("validarServidor(%q) error = %v, want nil", valor, err)
		}
	}

	invalidos := []string{"", "192.168.1.61", "192.168.1.61:0", "localhost:8080", "[::1]:8080", "192.168.1.61:70000"}
	for _, valor := range invalidos {
		if err := validarServidor(campoServer, valor); err == nil {
			t.Errorf("validarServidor(%q) error = nil, want un error", valor)
		}
	}
}

// TestValidarRuta comprueba que solo se aceptan rutas absolutas sin
// traversal ni comodines.
func TestValidarRuta(t *testing.T) {
	t.Parallel()

	validos := []string{"/opt/catalogo/app", "/etc/systemd/system/catalogo.service"}
	for _, valor := range validos {
		if err := validarRuta(campoDestino, valor); err != nil {
			t.Errorf("validarRuta(%q) error = %v, want nil", valor, err)
		}
	}

	invalidos := []string{"", "opt/app", "/opt/../etc/shadow", "/opt/app/*", "/opt/app/?", "/opt/app\x00oculto"}
	for _, valor := range invalidos {
		if err := validarRuta(campoDestino, valor); err == nil {
			t.Errorf("validarRuta(%q) error = nil, want un error", valor)
		}
	}
}

// TestValidarModo comprueba la regla de permisos en octal.
func TestValidarModo(t *testing.T) {
	t.Parallel()

	for _, valor := range []string{"644", "755", "0755"} {
		if err := validarModo(campoModo, valor); err != nil {
			t.Errorf("validarModo(%q) error = %v, want nil", valor, err)
		}
	}

	for _, valor := range []string{"", "rwx", "999", "75"} {
		if err := validarModo(campoModo, valor); err == nil {
			t.Errorf("validarModo(%q) error = nil, want un error", valor)
		}
	}
}

// TestValidarMensajeIssue comprueba el límite y la ausencia de NUL.
func TestValidarMensajeIssue(t *testing.T) {
	t.Parallel()

	if err := validarMensajeIssue("Bienvenido a la consola"); err != nil {
		t.Errorf("mensaje válido: error = %v, want nil", err)
	}

	if err := validarMensajeIssue(strings.Repeat("a", 512)); err != nil {
		t.Errorf("mensaje de 512: error = %v, want nil", err)
	}

	if err := validarMensajeIssue(strings.Repeat("a", 513)); err == nil {
		t.Error("mensaje de 513 caracteres: error = nil, want un error")
	}

	if err := validarMensajeIssue("mensaje\x00oculto"); err == nil {
		t.Error("mensaje con NUL: error = nil, want un error")
	}
}

// TestValidarIPv4 comprueba la regla de direcciones IPv4.
func TestValidarIPv4(t *testing.T) {
	t.Parallel()

	validos := []string{"192.168.1.60", "10.0.2.15"}
	for _, valor := range validos {
		if err := validarIPv4(campoIP, valor); err != nil {
			t.Errorf("validarIPv4(%q) error = %v, want nil", valor, err)
		}
	}

	invalidos := []string{"", "999.1.1.1", "localhost", "::1"}
	for _, valor := range invalidos {
		if err := validarIPv4(campoIP, valor); err == nil {
			t.Errorf("validarIPv4(%q) error = nil, want un error", valor)
		}
	}
}

// TestValidarPuerto comprueba el rango 1–65535.
func TestValidarPuerto(t *testing.T) {
	t.Parallel()

	for _, valor := range []int{1, 80, 65535} {
		if err := validarPuerto(valor); err != nil {
			t.Errorf("validarPuerto(%d) error = %v, want nil", valor, err)
		}
	}

	for _, valor := range []int{0, -1, 65536} {
		if err := validarPuerto(valor); err == nil {
			t.Errorf("validarPuerto(%d) error = nil, want un error", valor)
		}
	}
}

// TestBooleanoQuery comprueba la lectura de booleanos de la query.
func TestBooleanoQuery(t *testing.T) {
	t.Parallel()

	casos := []struct {
		valor string
		want  bool
		error bool
	}{
		{"", false, false},
		{"true", true, false},
		{"false", false, false},
		{"quizá", false, true},
	}

	for _, caso := range casos {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/users/jorge?files="+caso.valor, http.NoBody)

		obtenido, err := booleanoQuery(req, campoFiles)
		if (err != nil) != caso.error {
			t.Errorf("booleanoQuery(%q) error = %v, want error=%v", caso.valor, err, caso.error)

			continue
		}

		if obtenido != caso.want {
			t.Errorf("booleanoQuery(%q) = %v, want %v", caso.valor, obtenido, caso.want)
		}
	}
}
