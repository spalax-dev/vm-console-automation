// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestHaproxyErrorToReadConfig propaga el fallo de lectura.
func TestHaproxyErrorToReadConfig(t *testing.T) {
	t.Parallel()

	withError := &stubSession{errs: []error{errors.New("cat falló")}}
	haproxy := newHAProxyForTest(withError, nil)

	if _, err := haproxy.Add(context.Background(), ipServerNew, 9090); err == nil {
		t.Error("Agregar() error = nil, want el fallo de lectura")
	}

	other := &stubSession{errs: []error{errors.New("cat falló")}}
	if _, err := newHAProxyForTest(other, nil).Delete(context.Background(), ipServerTests, 8080); err == nil {
		t.Error("Eliminar() error = nil, want el fallo de lectura")
	}
}

// TestParseServersIgnoresLinesForeign descarta lo que no es un
// servidor con destino válido.
func TestParseServersIgnoresLinesForeign(t *testing.T) {
	t.Parallel()

	config := "backend app\n" +
		"    server incompleto\n" +
		"    server srv mala-direccion:8080\n" +
		"    balance roundrobin\n" +
		"    server srv1 192.168.1.61:8080 check\n"

	servers, found := parseServers(config)
	if !found || len(servers) != 1 {
		t.Fatalf("parsearServidores() = %+v, %v, want un servidor", servers, found)
	}
}

// TestServersWithoutBackend devuelve el error de configuración.
func TestServersWithoutBackend(t *testing.T) {
	t.Parallel()

	config := "global\n    daemon\n"

	if _, err := addServer(config, ipServerNew, 9090); !errors.Is(err, ErrBackendNotFound) {
		t.Errorf("agregarServidor() error = %v, want ErrBackendNoEncontrado", err)
	}

	if _, err := removeServer(config, ipServerNew, 9090); !errors.Is(err, ErrBackendNotFound) {
		t.Errorf("quitarServidor() error = %v, want ErrBackendNoEncontrado", err)
	}
}

// TestServerCoincideRejectsLinesForeign compara solo líneas «server».
func TestServerCoincideRejectsLinesForeign(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"",
		"server",
		"balance roundrobin",
		"server srv sin-direccion",
	}

	for _, line := range testCases {
		if serverMatches(line, ipServerTests, 8080) {
			t.Errorf("servidorCoincide(%q) = true, want false", line)
		}
	}
}

// TestLimitesBackend reconoce el final del bloque por otra sección o por
// un segundo back-end.
func TestLimitesBackend(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		config string
		start  int
		end    int
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			start, end := backendLimits(strings.Split(testCase.config, "\n"))
			if start != testCase.start || end != testCase.end {
				t.Errorf("limitesBackend() = %d, %d, want %d, %d", start, end, testCase.start, testCase.end)
			}
		})
	}
}
