// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"consola-vm/internal/core/domain"
)

// newHostnameOfTests arma el servicio con una sesión falsa.
func newHostnameOfTests(session *stubSession, err error) *Hostname {
	return NewHostname(&stubOpener{session: session, err: err})
}

// TestHostnameGet comprueba la lectura de nombre, IP y /etc/hosts.
func TestHostnameGet(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{
		{Stdout: outputHostname},
		{Stdout: "192.168.1.50 \n"},
		{Stdout: "127.0.0.1\tlocalhost\n127.0.1.1\tvm1\n"},
	}}

	hostname := newHostnameOfTests(session, nil)

	config, err := hostname.Get(context.Background())
	if err != nil {
		t.Fatalf(msgGetOK, err)
	}

	if config.Name != "vm1" {
		t.Errorf("Nombre = %q, want vm1", config.Name)
	}

	if config.IP != ipTests {
		t.Errorf("IP = %q, want 192.168.1.50", config.IP)
	}

	if !config.UpdateHosts {
		t.Error("ActualizarHosts = false, want true")
	}
}

// TestHostnameGetWithoutEntry comprueba que /etc/hosts sin el nombre
// deja la casilla desactivada.
func TestHostnameGetWithoutEntry(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{
		{Stdout: outputHostname},
		{Stdout: "192.168.1.50 \n"},
		{Stdout: "127.0.0.1\tlocalhost\n"},
	}}

	config, err := newHostnameOfTests(session, nil).Get(context.Background())
	if err != nil {
		t.Fatalf(msgGetOK, err)
	}

	if config.UpdateHosts {
		t.Error("ActualizarHosts = true, want false")
	}
}

// TestHostnameGetWithoutIP comprueba el error cuando no hay IPv4.
func TestHostnameGetWithoutIP(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{
		{Stdout: outputHostname},
		{Stdout: "\n"},
	}}

	if _, err := newHostnameOfTests(session, nil).Get(context.Background()); err == nil {
		t.Fatal("Obtener() error = nil, want el error de IP desconocida")
	}
}

// TestHostnameGetWithoutSession comprueba que se propaga la ausencia de
// sesión.
func TestHostnameGetWithoutSession(t *testing.T) {
	t.Parallel()

	hostname := newHostnameOfTests(nil, ErrSessionUnavailable)

	if _, err := hostname.Get(context.Background()); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf(msgGetWithoutSession, err)
	}
}

// TestHostnameWriteUpdatesHosts comprueba el cambio de nombre y la
// actualización de /etc/hosts.
func TestHostnameWriteUpdatesHosts(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{
		{Stdout: "127.0.0.1\tlocalhost\n192.168.1.60\tnodo-01\n"},
	}}

	hostname := newHostnameOfTests(session, nil)

	result, err := hostname.Write(context.Background(), HostnameConfig{
		Name:        nameHostTests,
		IP:          ipHostTests,
		UpdateHosts: true,
	})
	if err != nil {
		t.Fatalf(msgWriteOK, err)
	}

	if len(session.commands) != 2 {
		t.Fatalf(msgCommandDos, len(session.commands))
	}

	command := session.commands[1].ShellLine()
	if !strings.Contains(command, "hostnamectl set-hostname") {
		t.Errorf("orden = %q, want hostnamectl", command)
	}

	if !strings.Contains(command, "127.0.0.1") || !strings.Contains(command, "192.168.1.60 nodo-01") {
		t.Errorf("orden = %q, want /etc/hosts actualizado", command)
	}

	if !session.closed {
		t.Error(msgSessionOpen)
	}

	if result.Code != 0 {
		t.Errorf("Codigo = %d, want 0", result.Code)
	}
}

// TestHostnameWriteWithoutHosts comprueba que la casilla desactivada no
// toca /etc/hosts.
func TestHostnameWriteWithoutHosts(t *testing.T) {
	t.Parallel()

	session := &stubSession{}
	hostname := newHostnameOfTests(session, nil)

	if _, err := hostname.Write(context.Background(), HostnameConfig{
		Name: nameHostTests,
		IP:   ipHostTests,
	}); err != nil {
		t.Fatalf(msgWriteOK, err)
	}

	if len(session.commands) != 1 {
		t.Fatalf(msgCommandUnica, len(session.commands))
	}

	if command := session.commands[0].ShellLine(); strings.Contains(command, "/etc/hosts") {
		t.Errorf("orden = %q, no debe tocar /etc/hosts", command)
	}
}

// TestHostnameWriteErrorOfCommand comprueba el fallo al cambiar el
// nombre.
func TestHostnameWriteErrorOfCommand(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		result: domain.Result{Stderr: "hostnamectl: not found", ExitCode: 127},
		err:    &domain.ExitError{Code: 127, Command: "sh"},
	}

	result, err := newHostnameOfTests(session, nil).Write(context.Background(), HostnameConfig{
		Name: nameHostTests,
		IP:   ipHostTests,
	})
	if err == nil {
		t.Fatal("Escribir() error = nil, want el error de hostnamectl")
	}

	if result.Code != 127 {
		t.Errorf("Codigo = %d, want 127", result.Code)
	}

	if result.Stderr != "hostnamectl: not found" {
		t.Errorf(msgStderrSession, result.Stderr)
	}
}

// TestContentHostsReemplazaEntry comprueba el reemplazo de la línea
// del nombre sin duplicarla.
func TestContentHostsReemplazaEntry(t *testing.T) {
	t.Parallel()

	entry := "127.0.0.1\tlocalhost\n192.168.1.60\tnodo-01\n::1 localhost\n"

	obtained := hostsContent(entry, nameHostTests, ipServerTests)
	expected := "127.0.0.1\tlocalhost\n::1 localhost\n192.168.1.61 nodo-01\n"

	if obtained != expected {
		t.Errorf("contenidoHosts() = %q, want %q", obtained, expected)
	}
}

// TestHostnameGetErrorOfName comprueba el fallo al leer
// /etc/hostname.
func TestHostnameGetErrorOfName(t *testing.T) {
	t.Parallel()

	session := &stubSession{errs: []error{&domain.ExitError{Code: 1, Command: commandCat}}}

	if _, err := newHostnameOfTests(session, nil).Get(context.Background()); err == nil {
		t.Fatal("Obtener() error = nil, want el error de lectura")
	}
}

// TestHostnameGetErrorOfHosts comprueba el fallo al leer /etc/hosts.
func TestHostnameGetErrorOfHosts(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		results: []domain.Result{{Stdout: outputHostname}, {Stdout: "192.168.1.50\n"}},
		errs:    []error{nil, nil, &domain.ExitError{Code: 1, Command: commandCat}},
	}

	if _, err := newHostnameOfTests(session, nil).Get(context.Background()); err == nil {
		t.Fatal("Obtener() error = nil, want el error de /etc/hosts")
	}
}

// TestHostnameGetErrorOfIP comprueba el fallo de hostname -I.
func TestHostnameGetErrorOfIP(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		results: []domain.Result{{Stdout: outputHostname}},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: "hostname"}},
	}

	if _, err := newHostnameOfTests(session, nil).Get(context.Background()); err == nil {
		t.Fatal("Obtener() error = nil, want el error de hostname -I")
	}
}

// TestHostnameWriteErrorOfReadHosts comprueba el fallo al leer
// /etc/hosts.
func TestHostnameWriteErrorOfReadHosts(t *testing.T) {
	t.Parallel()

	session := &stubSession{errs: []error{&domain.ExitError{Code: 1, Command: commandCat}}}

	hostname := newHostnameOfTests(session, nil)

	_, err := hostname.Write(context.Background(), HostnameConfig{
		Name:        nameHostTests,
		IP:          ipHostTests,
		UpdateHosts: true,
	})
	if err == nil {
		t.Fatal("Escribir() error = nil, want el error al leer /etc/hosts")
	}
}

// TestHostnameWriteWithoutSession comprueba que se propaga la ausencia de
// sesión al escribir.
func TestHostnameWriteWithoutSession(t *testing.T) {
	t.Parallel()

	hostname := newHostnameOfTests(nil, ErrSessionUnavailable)

	_, err := hostname.Write(context.Background(), HostnameConfig{Name: nameHostTests})
	if !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("Escribir() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestContentHostsBordes comprueba que /etc/hosts se reescribe sin
// duplicar el nombre, sin confundirlo con un prefijo y conservando el
// resto de las líneas.
func TestContentHostsBordes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		entry    string
		esperada string
	}{
		{
			"archivo vacío",
			"",
			"10.0.0.5 nodo\n",
		},
		{
			"entrada nueva",
			"127.0.0.1 localhost\n",
			"127.0.0.1 localhost\n10.0.0.5 nodo\n",
		},
		{
			"reemplaza la entrada y conserva los alias",
			"10.0.0.4 nodo alias\n10.0.0.7 otro\n",
			"10.0.0.7 otro\n10.0.0.5 nodo\n",
		},
		{
			"elimina todas las apariciones",
			"10.0.0.4 nodo\n10.0.0.6 nodo alias\n",
			"10.0.0.5 nodo\n",
		},
		{
			"no confunde un prefijo con el nombre",
			"10.0.0.4 nodo-viejo\n",
			"10.0.0.4 nodo-viejo\n10.0.0.5 nodo\n",
		},
		{
			"descarta líneas vacías",
			"\n\n127.0.0.1 localhost\n\n",
			"127.0.0.1 localhost\n10.0.0.5 nodo\n",
		},
		{
			"conserva el comentario de la línea reemplazada",
			"10.0.0.5 nodo # comentario\n",
			"10.0.0.5 nodo\n",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			obtenida := hostsContent(testCase.entry, "nodo", "10.0.0.5")
			if obtenida != testCase.esperada {
				t.Errorf("contenidoHosts() = %q, want %q", obtenida, testCase.esperada)
			}
		})
	}
}

// TestNameInHostsBordes exige que el nombre aparezca como destino y
// no como prefijo de otro nombre.
func TestNameInHostsBordes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		content  string
		expected bool
	}{
		{"vacío", "", false},
		{"solo localhost", "127.0.0.1 localhost\n", false},
		{"como destino", "10.0.0.5 nodo\n", true},
		{"como alias", "10.0.0.5 nodo.lab nodo\n", true},
		{"como prefijo", "10.0.0.4 nodo-viejo\n", false},
		{"sin dirección", "nodo\n", false},
		{"con tabuladores", "10.0.0.5\tnodo\n", true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if obtained := nameInHosts(testCase.content, "nodo"); obtained != testCase.expected {
				t.Errorf("nombreEnHosts() = %v, want %v", obtained, testCase.expected)
			}
		})
	}
}
