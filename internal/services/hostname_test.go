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

// nuevoHostnameDePrueba arma el servicio con una sesión falsa.
func nuevoHostnameDePrueba(sesion *stubSession, err error) *Hostname {
	return NuevoHostname(&stubAbridor{sesion: sesion, err: err})
}

// TestHostnameObtener comprueba la lectura de nombre, IP y /etc/hosts.
func TestHostnameObtener(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{results: []domain.Result{
		{Stdout: salidaHostname},
		{Stdout: "192.168.1.50 \n"},
		{Stdout: "127.0.0.1\tlocalhost\n127.0.1.1\tdebian13\n"},
	}}

	hostname := nuevoHostnameDePrueba(sesion, nil)

	config, err := hostname.Obtener(context.Background())
	if err != nil {
		t.Fatalf(msgObtenerOK, err)
	}

	if config.Nombre != "debian13" {
		t.Errorf("Nombre = %q, want debian13", config.Nombre)
	}

	if config.IP != "192.168.1.50" {
		t.Errorf("IP = %q, want 192.168.1.50", config.IP)
	}

	if !config.ActualizarHosts {
		t.Error("ActualizarHosts = false, want true")
	}
}

// TestHostnameObtenerSinEntrada comprueba que /etc/hosts sin el nombre
// deja la casilla desactivada.
func TestHostnameObtenerSinEntrada(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{results: []domain.Result{
		{Stdout: salidaHostname},
		{Stdout: "192.168.1.50 \n"},
		{Stdout: "127.0.0.1\tlocalhost\n"},
	}}

	config, err := nuevoHostnameDePrueba(sesion, nil).Obtener(context.Background())
	if err != nil {
		t.Fatalf(msgObtenerOK, err)
	}

	if config.ActualizarHosts {
		t.Error("ActualizarHosts = true, want false")
	}
}

// TestHostnameObtenerSinIP comprueba el error cuando no hay IPv4.
func TestHostnameObtenerSinIP(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{results: []domain.Result{
		{Stdout: salidaHostname},
		{Stdout: "\n"},
	}}

	if _, err := nuevoHostnameDePrueba(sesion, nil).Obtener(context.Background()); err == nil {
		t.Fatal("Obtener() error = nil, want el error de IP desconocida")
	}
}

// TestHostnameObtenerSinSesion comprueba que se propaga la ausencia de
// sesión.
func TestHostnameObtenerSinSesion(t *testing.T) {
	t.Parallel()

	hostname := nuevoHostnameDePrueba(nil, ErrSesionNoDisponible)

	if _, err := hostname.Obtener(context.Background()); !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf(msgObtenerSinSesion, err)
	}
}

// TestHostnameEscribirActualizaHosts comprueba el cambio de nombre y la
// actualización de /etc/hosts.
func TestHostnameEscribirActualizaHosts(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{results: []domain.Result{
		{Stdout: "127.0.0.1\tlocalhost\n192.168.1.60\tnodo-01\n"},
	}}

	hostname := nuevoHostnameDePrueba(sesion, nil)

	resultado, err := hostname.Escribir(context.Background(), ConfiguracionHostname{
		Nombre:          nombreHostPrueba,
		IP:              ipHostPrueba,
		ActualizarHosts: true,
	})
	if err != nil {
		t.Fatalf(msgEscribirOK, err)
	}

	if len(sesion.commands) != 2 {
		t.Fatalf(msgOrdenDos, len(sesion.commands))
	}

	orden := sesion.commands[1].ShellLine()
	if !strings.Contains(orden, "hostnamectl set-hostname") {
		t.Errorf("orden = %q, want hostnamectl", orden)
	}

	if !strings.Contains(orden, "127.0.0.1") || !strings.Contains(orden, "192.168.1.60 nodo-01") {
		t.Errorf("orden = %q, want /etc/hosts actualizado", orden)
	}

	if !sesion.closed {
		t.Error(msgSesionAbierta)
	}

	if resultado.Codigo != 0 {
		t.Errorf("Codigo = %d, want 0", resultado.Codigo)
	}
}

// TestHostnameEscribirSinHosts comprueba que la casilla desactivada no
// toca /etc/hosts.
func TestHostnameEscribirSinHosts(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{}
	hostname := nuevoHostnameDePrueba(sesion, nil)

	if _, err := hostname.Escribir(context.Background(), ConfiguracionHostname{
		Nombre: nombreHostPrueba,
		IP:     ipHostPrueba,
	}); err != nil {
		t.Fatalf(msgEscribirOK, err)
	}

	if len(sesion.commands) != 1 {
		t.Fatalf(msgOrdenUnica, len(sesion.commands))
	}

	if orden := sesion.commands[0].ShellLine(); strings.Contains(orden, "/etc/hosts") {
		t.Errorf("orden = %q, no debe tocar /etc/hosts", orden)
	}
}

// TestHostnameEscribirErrorDeOrden comprueba el fallo al cambiar el
// nombre.
func TestHostnameEscribirErrorDeOrden(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{
		result: domain.Result{Stderr: "hostnamectl: not found", ExitCode: 127},
		err:    &domain.ExitError{Code: 127, Command: "sh"},
	}

	resultado, err := nuevoHostnameDePrueba(sesion, nil).Escribir(context.Background(), ConfiguracionHostname{
		Nombre: nombreHostPrueba,
		IP:     ipHostPrueba,
	})
	if err == nil {
		t.Fatal("Escribir() error = nil, want el error de hostnamectl")
	}

	if resultado.Codigo != 127 {
		t.Errorf("Codigo = %d, want 127", resultado.Codigo)
	}

	if resultado.Stderr != "hostnamectl: not found" {
		t.Errorf(msgStderrSesion, resultado.Stderr)
	}
}

// TestContenidoHostsReemplazaEntrada comprueba el reemplazo de la línea
// del nombre sin duplicarla.
func TestContenidoHostsReemplazaEntrada(t *testing.T) {
	t.Parallel()

	entrada := "127.0.0.1\tlocalhost\n192.168.1.60\tnodo-01\n::1 localhost\n"

	obtenido := contenidoHosts(entrada, nombreHostPrueba, ipServidorPrueba)
	esperado := "127.0.0.1\tlocalhost\n::1 localhost\n192.168.1.61 nodo-01\n"

	if obtenido != esperado {
		t.Errorf("contenidoHosts() = %q, want %q", obtenido, esperado)
	}
}

// TestHostnameObtenerErrorDeNombre comprueba el fallo al leer
// /etc/hostname.
func TestHostnameObtenerErrorDeNombre(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{errs: []error{&domain.ExitError{Code: 1, Command: comandoCat}}}

	if _, err := nuevoHostnameDePrueba(sesion, nil).Obtener(context.Background()); err == nil {
		t.Fatal("Obtener() error = nil, want el error de lectura")
	}
}

// TestHostnameObtenerErrorDeHosts comprueba el fallo al leer /etc/hosts.
func TestHostnameObtenerErrorDeHosts(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{
		results: []domain.Result{{Stdout: salidaHostname}, {Stdout: "192.168.1.50\n"}},
		errs:    []error{nil, nil, &domain.ExitError{Code: 1, Command: comandoCat}},
	}

	if _, err := nuevoHostnameDePrueba(sesion, nil).Obtener(context.Background()); err == nil {
		t.Fatal("Obtener() error = nil, want el error de /etc/hosts")
	}
}

// TestHostnameObtenerErrorDeIP comprueba el fallo de hostname -I.
func TestHostnameObtenerErrorDeIP(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{
		results: []domain.Result{{Stdout: salidaHostname}},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: "hostname"}},
	}

	if _, err := nuevoHostnameDePrueba(sesion, nil).Obtener(context.Background()); err == nil {
		t.Fatal("Obtener() error = nil, want el error de hostname -I")
	}
}

// TestHostnameEscribirErrorDeLecturaHosts comprueba el fallo al leer
// /etc/hosts.
func TestHostnameEscribirErrorDeLecturaHosts(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{errs: []error{&domain.ExitError{Code: 1, Command: comandoCat}}}

	hostname := nuevoHostnameDePrueba(sesion, nil)

	_, err := hostname.Escribir(context.Background(), ConfiguracionHostname{
		Nombre:          nombreHostPrueba,
		IP:              ipHostPrueba,
		ActualizarHosts: true,
	})
	if err == nil {
		t.Fatal("Escribir() error = nil, want el error al leer /etc/hosts")
	}
}

// TestHostnameEscribirSinSesion comprueba que se propaga la ausencia de
// sesión al escribir.
func TestHostnameEscribirSinSesion(t *testing.T) {
	t.Parallel()

	hostname := nuevoHostnameDePrueba(nil, ErrSesionNoDisponible)

	_, err := hostname.Escribir(context.Background(), ConfiguracionHostname{Nombre: nombreHostPrueba})
	if !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf("Escribir() error = %v, want ErrSesionNoDisponible", err)
	}
}
