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

// mensajes repetidos en las pruebas de /etc/issue.
const (
	textBienvenido = "Bienvenido"
	msgGetOK       = "Obtener() error = %v, want nil"
)

// issueCompleto simula un /etc/issue escrito por la consola.
const issueCompleto = "Bienvenido\nHostname: \\n\nFecha: \\d\nIP: \\4\n"

// newIssueOfTests arma el servicio con una sesión falsa.
func newIssueOfTests(session *stubSession, err error) *Issue {
	return NewIssue(&stubOpener{session: session, err: err})
}

// TestIssueWriteGeneratesContent comprueba el formato de una línea por
// elemento y los permisos del archivo.
func TestIssueWriteGeneratesContent(t *testing.T) {
	t.Parallel()

	session := &stubSession{}
	issue := newIssueOfTests(session, nil)

	result, err := issue.Write(context.Background(), IssueConfig{
		Message:  textBienvenido,
		NameHost: true,
		Date:     true,
		IP:       true,
	})
	if err != nil {
		t.Fatalf(msgWriteOK, err)
	}

	if len(session.commands) != 1 {
		t.Fatalf(msgCommandUnica, len(session.commands))
	}

	command := session.commands[0].ShellLine()
	for _, expected := range []string{"Bienvenido", "Hostname: \\n", "Fecha: \\d", "IP: \\4", "/etc/issue", "chmod 644"} {
		if !strings.Contains(command, expected) {
			t.Errorf("orden = %q, want que contenga %q", command, expected)
		}
	}

	if strings.Contains(command, "Hora: \\t") {
		t.Errorf("orden = %q, no debe incluir la hora desactivada", command)
	}

	if !session.closed {
		t.Error(msgSessionOpen)
	}

	if result.Code != 0 {
		t.Errorf(msgCodeCero, result.Code)
	}

	if result.Message == "" {
		t.Error("Mensaje vacío, want el mensaje de éxito")
	}
}

// TestBuildIssueKeepsLineBreaks comprueba que el mensaje multilínea se
// conserva tal cual en el archivo.
func TestBuildIssueKeepsLineBreaks(t *testing.T) {
	t.Parallel()

	if got := buildIssue(IssueConfig{Message: "linea1\nlinea2"}); got != "linea1\nlinea2\n" {
		t.Errorf("buildIssue() = %q, want el mensaje multilínea", got)
	}
}

// TestIssueGetReconoceOptions comprueba la lectura del formato de la
// consola.
func TestIssueGetReconoceOptions(t *testing.T) {
	t.Parallel()

	session := &stubSession{result: domain.Result{Stdout: issueCompleto}}
	issue := newIssueOfTests(session, nil)

	config, err := issue.Get(context.Background())
	if err != nil {
		t.Fatalf(msgGetOK, err)
	}

	if config.Message != textBienvenido {
		t.Errorf("Mensaje = %q, want Bienvenido", config.Message)
	}

	if !config.NameHost || !config.Date || !config.IP {
		t.Errorf("opciones = %+v, want nombre_host, fecha e IP activos", config)
	}

	if config.Hour {
		t.Error("Hora = true, want false")
	}

	if config.Content != issueCompleto {
		t.Errorf("Contenido = %q, want el archivo completo", config.Content)
	}
}

// TestIssueGetFormatAjeno comprueba que un /etc/issue ajeno se
// devuelve como mensaje sin opciones.
func TestIssueGetFormatAjeno(t *testing.T) {
	t.Parallel()

	session := &stubSession{result: domain.Result{Stdout: "Debian GNU/Linux 13 \\n \\l\n"}}
	issue := newIssueOfTests(session, nil)

	config, err := issue.Get(context.Background())
	if err != nil {
		t.Fatalf(msgGetOK, err)
	}

	if config.NameHost || config.Date || config.Hour || config.IP {
		t.Errorf("opciones = %+v, want todas desactivadas", config)
	}

	if !strings.Contains(config.Message, "Debian GNU/Linux") {
		t.Errorf("Mensaje = %q, want el contenido original", config.Message)
	}
}

// TestIssueWriteWithoutSession comprueba que se propaga la ausencia de
// sesión.
func TestIssueWriteWithoutSession(t *testing.T) {
	t.Parallel()

	issue := newIssueOfTests(nil, ErrSessionUnavailable)

	_, err := issue.Write(context.Background(), IssueConfig{Message: "hola"})
	if !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("Escribir() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestIssueGetErrorOfRead comprueba que el fallo de cat se
// propaga.
func TestIssueGetErrorOfRead(t *testing.T) {
	t.Parallel()

	session := &stubSession{err: &domain.ExitError{Code: 1, Command: commandCat}}
	issue := newIssueOfTests(session, nil)

	if _, err := issue.Get(context.Background()); err == nil {
		t.Fatal("Obtener() error = nil, want el error de lectura")
	}
}

// TestIssueWriteErrorOfCommand comprueba que el fallo de la escritura
// conserva el detalle técnico.
func TestIssueWriteErrorOfCommand(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		result: domain.Result{Stderr: "Read-only file system", ExitCode: 1},
		err:    &domain.ExitError{Code: 1, Command: "sh"},
	}
	issue := newIssueOfTests(session, nil)

	result, err := issue.Write(context.Background(), IssueConfig{Message: "hola"})
	if err == nil {
		t.Fatal("Escribir() error = nil, want el error de escritura")
	}

	if result.Code != 1 {
		t.Errorf("Codigo = %d, want 1", result.Code)
	}

	if result.Stderr != "Read-only file system" {
		t.Errorf(msgStderrSession, result.Stderr)
	}
}

// TestIssueHour comprueba la opción de hora al escribir y al leer.
func TestIssueHour(t *testing.T) {
	t.Parallel()

	session := &stubSession{}
	issue := newIssueOfTests(session, nil)

	if _, err := issue.Write(context.Background(), IssueConfig{Hour: true}); err != nil {
		t.Fatalf(msgWriteOK, err)
	}

	if command := session.commands[0].ShellLine(); !strings.Contains(command, "Hora: \\t") {
		t.Errorf("orden = %q, want la secuencia de hora", command)
	}

	session.result = domain.Result{Stdout: "Hola\nHora: \\t\n"}

	config, err := issue.Get(context.Background())
	if err != nil {
		t.Fatalf(msgGetOK, err)
	}

	if !config.Hour {
		t.Error("Hora = false, want true")
	}

	if config.Message != "Hola" {
		t.Errorf("Mensaje = %q, want Hola", config.Message)
	}
}

// TestIssueGetWithoutSession comprueba que se propaga la ausencia de
// sesión al leer.
func TestIssueGetWithoutSession(t *testing.T) {
	t.Parallel()

	issue := newIssueOfTests(nil, ErrSessionUnavailable)

	if _, err := issue.Get(context.Background()); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf(msgGetWithoutSession, err)
	}
}
