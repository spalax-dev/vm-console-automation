// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestHaproxyErrorAlLeerLaConfiguracion propaga el fallo de lectura.
func TestHaproxyErrorAlLeerLaConfiguracion(t *testing.T) {
	t.Parallel()

	conError := &stubSession{errs: []error{errors.New("cat falló")}}
	haproxy := nuevoHaproxyDePrueba(conError, nil)

	if _, err := haproxy.Agregar(context.Background(), ipServidorNuevo, 9090); err == nil {
		t.Error("Agregar() error = nil, want el fallo de lectura")
	}

	otro := &stubSession{errs: []error{errors.New("cat falló")}}
	if _, err := nuevoHaproxyDePrueba(otro, nil).Eliminar(context.Background(), ipServidorPrueba, 8080); err == nil {
		t.Error("Eliminar() error = nil, want el fallo de lectura")
	}
}

// TestParsearServidoresIgnoraLineasAjenas descarta lo que no es un
// servidor con destino válido.
func TestParsearServidoresIgnoraLineasAjenas(t *testing.T) {
	t.Parallel()

	config := "backend app\n" +
		"    server incompleto\n" +
		"    server srv mala-direccion:8080\n" +
		"    balance roundrobin\n" +
		"    server srv1 192.168.1.61:8080 check\n"

	servidores, encontrado := parsearServidores(config)
	if !encontrado || len(servidores) != 1 {
		t.Fatalf("parsearServidores() = %+v, %v, want un servidor", servidores, encontrado)
	}
}

// TestServidoresSinBackend devuelve el error de configuración.
func TestServidoresSinBackend(t *testing.T) {
	t.Parallel()

	config := "global\n    daemon\n"

	if _, err := agregarServidor(config, ipServidorNuevo, 9090); !errors.Is(err, ErrBackendNoEncontrado) {
		t.Errorf("agregarServidor() error = %v, want ErrBackendNoEncontrado", err)
	}

	if _, err := quitarServidor(config, ipServidorNuevo, 9090); !errors.Is(err, ErrBackendNoEncontrado) {
		t.Errorf("quitarServidor() error = %v, want ErrBackendNoEncontrado", err)
	}
}

// TestServidorCoincideRechazaLineasAjenas compara solo líneas «server».
func TestServidorCoincideRechazaLineasAjenas(t *testing.T) {
	t.Parallel()

	casos := []string{
		"",
		"server",
		"balance roundrobin",
		"server srv sin-direccion",
	}

	for _, linea := range casos {
		if servidorCoincide(linea, ipServidorPrueba, 8080) {
			t.Errorf("servidorCoincide(%q) = true, want false", linea)
		}
	}
}

// TestLimitesBackend reconoce el final del bloque por otra sección o por
// un segundo back-end.
func TestLimitesBackend(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre string
		config string
		inicio int
		fin    int
	}{
		{
			"otra sección",
			"frontend http\n    bind *:80\nbackend app\n    server srv1 192.168.1.61:8080\nfrontend otro\n",
			2, 4,
		},
		{
			"segundo back-end",
			"backend app\n    server srv1 192.168.1.61:8080\nbackend otro\n    server srv2 192.168.1.62:9090\n",
			0, 2,
		},
		{
			"sin back-end",
			"global\n    daemon\n",
			-1, -1,
		},
		{
			"al final del archivo",
			"global\n    daemon\nbackend app\n    server srv1 192.168.1.61:8080\n",
			2, 5,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			inicio, fin := limitesBackend(strings.Split(caso.config, "\n"))
			if inicio != caso.inicio || fin != caso.fin {
				t.Errorf("limitesBackend() = %d, %d, want %d, %d", inicio, fin, caso.inicio, caso.fin)
			}
		})
	}
}
