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
	textoBienvenido = "Bienvenido"
	msgObtenerOK    = "Obtener() error = %v, want nil"
)

// issueCompleto simula un /etc/issue escrito por la consola.
const issueCompleto = "Bienvenido\nHostname: \\n\nFecha: \\d\nIP: \\4\n"

// nuevoIssueDePrueba arma el servicio con una sesión falsa.
func nuevoIssueDePrueba(sesion *stubSession, err error) *Issue {
	return NuevoIssue(&stubAbridor{sesion: sesion, err: err})
}

// TestIssueEscribirGeneraContenido comprueba el formato de una línea por
// elemento y los permisos del archivo.
func TestIssueEscribirGeneraContenido(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{}
	issue := nuevoIssueDePrueba(sesion, nil)

	resultado, err := issue.Escribir(context.Background(), ConfiguracionIssue{
		Mensaje:    textoBienvenido,
		NombreHost: true,
		Fecha:      true,
		IP:         true,
	})
	if err != nil {
		t.Fatalf(msgEscribirOK, err)
	}

	if len(sesion.commands) != 1 {
		t.Fatalf(msgOrdenUnica, len(sesion.commands))
	}

	orden := sesion.commands[0].ShellLine()
	for _, esperado := range []string{"Bienvenido", "Hostname: \\n", "Fecha: \\d", "IP: \\4", "/etc/issue", "chmod 644"} {
		if !strings.Contains(orden, esperado) {
			t.Errorf("orden = %q, want que contenga %q", orden, esperado)
		}
	}

	if strings.Contains(orden, "Hora: \\t") {
		t.Errorf("orden = %q, no debe incluir la hora desactivada", orden)
	}

	if !sesion.closed {
		t.Error(msgSesionAbierta)
	}

	if resultado.Codigo != 0 {
		t.Errorf(msgCodigoCero, resultado.Codigo)
	}

	if resultado.Mensaje == "" {
		t.Error("Mensaje vacío, want el mensaje de éxito")
	}
}

// TestIssueObtenerReconoceOpciones comprueba la lectura del formato de la
// consola.
func TestIssueObtenerReconoceOpciones(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{result: domain.Result{Stdout: issueCompleto}}
	issue := nuevoIssueDePrueba(sesion, nil)

	config, err := issue.Obtener(context.Background())
	if err != nil {
		t.Fatalf(msgObtenerOK, err)
	}

	if config.Mensaje != textoBienvenido {
		t.Errorf("Mensaje = %q, want Bienvenido", config.Mensaje)
	}

	if !config.NombreHost || !config.Fecha || !config.IP {
		t.Errorf("opciones = %+v, want nombre_host, fecha e IP activos", config)
	}

	if config.Hora {
		t.Error("Hora = true, want false")
	}

	if config.Contenido != issueCompleto {
		t.Errorf("Contenido = %q, want el archivo completo", config.Contenido)
	}
}

// TestIssueObtenerFormatoAjeno comprueba que un /etc/issue ajeno se
// devuelve como mensaje sin opciones.
func TestIssueObtenerFormatoAjeno(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{result: domain.Result{Stdout: "Debian GNU/Linux 13 \\n \\l\n"}}
	issue := nuevoIssueDePrueba(sesion, nil)

	config, err := issue.Obtener(context.Background())
	if err != nil {
		t.Fatalf(msgObtenerOK, err)
	}

	if config.NombreHost || config.Fecha || config.Hora || config.IP {
		t.Errorf("opciones = %+v, want todas desactivadas", config)
	}

	if !strings.Contains(config.Mensaje, "Debian GNU/Linux") {
		t.Errorf("Mensaje = %q, want el contenido original", config.Mensaje)
	}
}

// TestIssueEscribirSinSesion comprueba que se propaga la ausencia de
// sesión.
func TestIssueEscribirSinSesion(t *testing.T) {
	t.Parallel()

	issue := nuevoIssueDePrueba(nil, ErrSesionNoDisponible)

	_, err := issue.Escribir(context.Background(), ConfiguracionIssue{Mensaje: "hola"})
	if !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf("Escribir() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestIssueObtenerErrorDeLectura comprueba que el fallo de cat se
// propaga.
func TestIssueObtenerErrorDeLectura(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{err: &domain.ExitError{Code: 1, Command: comandoCat}}
	issue := nuevoIssueDePrueba(sesion, nil)

	if _, err := issue.Obtener(context.Background()); err == nil {
		t.Fatal("Obtener() error = nil, want el error de lectura")
	}
}

// TestIssueEscribirErrorDeOrden comprueba que el fallo de la escritura
// conserva el detalle técnico.
func TestIssueEscribirErrorDeOrden(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{
		result: domain.Result{Stderr: "Read-only file system", ExitCode: 1},
		err:    &domain.ExitError{Code: 1, Command: "sh"},
	}
	issue := nuevoIssueDePrueba(sesion, nil)

	resultado, err := issue.Escribir(context.Background(), ConfiguracionIssue{Mensaje: "hola"})
	if err == nil {
		t.Fatal("Escribir() error = nil, want el error de escritura")
	}

	if resultado.Codigo != 1 {
		t.Errorf("Codigo = %d, want 1", resultado.Codigo)
	}

	if resultado.Stderr != "Read-only file system" {
		t.Errorf(msgStderrSesion, resultado.Stderr)
	}
}

// TestIssueHora comprueba la opción de hora al escribir y al leer.
func TestIssueHora(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{}
	issue := nuevoIssueDePrueba(sesion, nil)

	if _, err := issue.Escribir(context.Background(), ConfiguracionIssue{Hora: true}); err != nil {
		t.Fatalf(msgEscribirOK, err)
	}

	if orden := sesion.commands[0].ShellLine(); !strings.Contains(orden, "Hora: \\t") {
		t.Errorf("orden = %q, want la secuencia de hora", orden)
	}

	sesion.result = domain.Result{Stdout: "Hola\nHora: \\t\n"}

	config, err := issue.Obtener(context.Background())
	if err != nil {
		t.Fatalf(msgObtenerOK, err)
	}

	if !config.Hora {
		t.Error("Hora = false, want true")
	}

	if config.Mensaje != "Hola" {
		t.Errorf("Mensaje = %q, want Hola", config.Mensaje)
	}
}

// TestIssueObtenerSinSesion comprueba que se propaga la ausencia de
// sesión al leer.
func TestIssueObtenerSinSesion(t *testing.T) {
	t.Parallel()

	issue := nuevoIssueDePrueba(nil, ErrSesionNoDisponible)

	if _, err := issue.Obtener(context.Background()); !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf("Obtener() error = %v, want ErrSesionNoDisponible", err)
	}
}
