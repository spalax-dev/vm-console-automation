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

// unidadPrueba es la unidad usada en las pruebas.
const unidadPrueba = "catalogo.service"

// salidaUnidades simula systemctl list-units.
const salidaUnidades = "catalogo.service loaded active running Catálogo\n" +
	"ssh.service loaded active running OpenBSD Secure Shell server\n" +
	"cron.service loaded inactive dead Regular background daemon\n"

// salidaUnitFiles simula systemctl list-unit-files.
const salidaUnitFiles = "catalogo.service enabled\n" +
	"ssh.service enabled\n" +
	"cron.service disabled\n"

// nuevoSystemdDePrueba arma el servicio con una sesión falsa.
func nuevoSystemdDePrueba(sesion *stubSession, err error) *Systemd {
	return NuevoSystemd(&stubAbridor{sesion: sesion, err: err})
}

// TestSystemdListar combina unidades y habilitación.
func TestSystemdListar(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{results: []domain.Result{
		{Stdout: salidaUnidades},
		{Stdout: salidaUnitFiles},
	}}

	lista, err := nuevoSystemdDePrueba(sesion, nil).Listar(context.Background())
	if err != nil {
		t.Fatalf(msgListarOK, err)
	}

	if len(lista) != 3 {
		t.Fatalf("Listar() devolvió %d unidades, want 3", len(lista))
	}

	if !lista[0].Activo || !lista[0].Habilitado || lista[0].Estado != estadoActivo || lista[0].Detalle != "running" {
		t.Errorf("catalogo = %+v, want activa, habilitada y running", lista[0])
	}

	if lista[2].Activo || lista[2].Habilitado || lista[2].Estado != estadoInactivo {
		t.Errorf("cron = %+v, want inactiva y deshabilitada", lista[2])
	}
}

// TestSystemdObtener interpreta systemctl show.
func TestSystemdObtener(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{result: domain.Result{Stdout: "LoadState=loaded\nActiveState=active\nUnitFileState=enabled\nSubState=running\n"}}

	estado, err := nuevoSystemdDePrueba(sesion, nil).Obtener(context.Background(), unidadPrueba)
	if err != nil {
		t.Fatalf("Obtener() error = %v, want nil", err)
	}

	if !estado.Activo || !estado.Habilitado || estado.Estado != estadoActivo || estado.Detalle != "running" {
		t.Errorf("estado = %+v, want activa, habilitada y running", estado)
	}
}

// TestSystemdObtenerNoEncontrada comprueba LoadState=not-found.
func TestSystemdObtenerNoEncontrada(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{result: domain.Result{Stdout: "LoadState=not-found\n"}}

	if _, err := nuevoSystemdDePrueba(sesion, nil).Obtener(context.Background(), unidadPrueba); !errors.Is(err, ErrUnidadNoEncontrada) {
		t.Fatalf("Obtener() error = %v, want ErrUnidadNoEncontrada", err)
	}
}

// TestSystemdObtenerCodigoCuatro comprueba el código 4 de systemctl.
func TestSystemdObtenerCodigoCuatro(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{err: &domain.ExitError{Code: 4, Command: binSystemctl}}

	if _, err := nuevoSystemdDePrueba(sesion, nil).Obtener(context.Background(), unidadPrueba); !errors.Is(err, ErrUnidadNoEncontrada) {
		t.Fatalf("Obtener() error = %v, want ErrUnidadNoEncontrada", err)
	}
}

// TestSystemdDesplegar transfiere, ajusta permisos y recarga systemd.
func TestSystemdDesplegar(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{}
	systemd := nuevoSystemdDePrueba(sesion, nil)

	resultado, err := systemd.Desplegar(context.Background(), unidadPrueba, []ArchivoDespliegue{
		{Origen: "./app/catalogo", Destino: "/opt/catalogo/catalogo", Modo: "755"},
	}, duenoPrueba, duenoPrueba)
	if err != nil {
		t.Fatalf("Desplegar() error = %v, want nil", err)
	}

	if len(sesion.puts) != 1 || sesion.puts[0][1] != "/opt/catalogo/catalogo" {
		t.Errorf("Put() = %v, want el archivo transferido", sesion.puts)
	}

	if len(sesion.commands) != 3 {
		t.Fatalf("la sesión ejecutó %d órdenes, want 3", len(sesion.commands))
	}

	if sesion.commands[0].Path != binChmod || strings.Join(sesion.commands[0].Args, " ") != "755 /opt/catalogo/catalogo" {
		t.Errorf("chmod = %v, want 755 en el destino", sesion.commands[0])
	}

	if sesion.commands[1].Path != binChown || sesion.commands[1].Args[0] != "catalogo:catalogo" {
		t.Errorf("chown = %v, want catalogo:catalogo", sesion.commands[1])
	}

	if sesion.commands[2].Path != binSystemctl || sesion.commands[2].Args[0] != "daemon-reload" {
		t.Errorf("systemctl = %v, want daemon-reload", sesion.commands[2])
	}

	if resultado.Codigo != 0 {
		t.Errorf(msgCodigoCero, resultado.Codigo)
	}
}

// TestSystemdCambiarEstado cubre start, stop, enable y disable.
func TestSystemdCambiarEstado(t *testing.T) {
	t.Parallel()

	verdadero := true
	falso := false

	casos := []struct {
		nombre     string
		activo     *bool
		habilitado *bool
		esperado   string
	}{
		{"start", &verdadero, nil, "start"},
		{"stop", &falso, nil, "stop"},
		{"enable", nil, &verdadero, "enable"},
		{"disable", nil, &falso, "disable"},
	}

	for _, caso := range casos {
		sesion := &stubSession{}
		systemd := nuevoSystemdDePrueba(sesion, nil)

		if _, err := systemd.CambiarEstado(context.Background(), unidadPrueba, caso.activo, caso.habilitado); err != nil {
			t.Fatalf("%s: CambiarEstado() error = %v", caso.nombre, err)
		}

		cmd := sesion.commands[0]
		if cmd.Path != binSystemctl || cmd.Args[0] != caso.esperado || cmd.Args[1] != unidadPrueba {
			t.Errorf("%s: orden = %v, want %s %s", caso.nombre, cmd.Args, caso.esperado, unidadPrueba)
		}
	}
}

// TestSystemdReiniciar comprueba systemctl restart.
func TestSystemdReiniciar(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{}

	if _, err := nuevoSystemdDePrueba(sesion, nil).Reiniciar(context.Background(), unidadPrueba); err != nil {
		t.Fatalf("Reiniciar() error = %v, want nil", err)
	}

	cmd := sesion.commands[0]
	if cmd.Args[0] != "restart" || cmd.Args[1] != unidadPrueba {
		t.Errorf("orden = %v, want restart %s", cmd.Args, unidadPrueba)
	}
}

// TestSystemdListarSinSesion comprueba la ausencia de sesión.
func TestSystemdListarSinSesion(t *testing.T) {
	t.Parallel()

	systemd := nuevoSystemdDePrueba(nil, ErrSesionNoDisponible)

	if _, err := systemd.Listar(context.Background()); !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf(msgListarSinSesion, err)
	}
}

// TestSystemdCambiarEstadoError comprueba el fallo de systemctl.
func TestSystemdCambiarEstadoError(t *testing.T) {
	t.Parallel()

	verdadero := true
	sesion := &stubSession{
		result: domain.Result{Stderr: "Unit not found", ExitCode: 5},
		err:    &domain.ExitError{Code: 5, Command: binSystemctl},
	}

	resultado, err := nuevoSystemdDePrueba(sesion, nil).CambiarEstado(context.Background(), unidadPrueba, &verdadero, nil)
	if err == nil {
		t.Fatal("CambiarEstado() error = nil, want el error de systemctl")
	}

	if resultado.Codigo != 5 || resultado.Stderr == "" {
		t.Errorf(msgResultadoTecnico, resultado)
	}
}

// TestSystemdListarError comprueba el fallo al listar unidades.
func TestSystemdListarError(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{err: &domain.ExitError{Code: 1, Command: binSystemctl}}

	if _, err := nuevoSystemdDePrueba(sesion, nil).Listar(context.Background()); err == nil {
		t.Fatal("Listar() error = nil, want el error de systemctl")
	}
}

// TestSystemdListarErrorDeArchivos comprueba el fallo al consultar la
// habilitación.
func TestSystemdListarErrorDeArchivos(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{
		results: []domain.Result{{Stdout: salidaUnidades}},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: binSystemctl}},
	}

	if _, err := nuevoSystemdDePrueba(sesion, nil).Listar(context.Background()); err == nil {
		t.Fatal("Listar() error = nil, want el error de list-unit-files")
	}
}

// TestSystemdObtenerError comprueba un fallo inesperado de systemctl.
func TestSystemdObtenerError(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{err: &domain.ExitError{Code: 1, Command: binSystemctl}}

	if _, err := nuevoSystemdDePrueba(sesion, nil).Obtener(context.Background(), unidadPrueba); err == nil {
		t.Fatal("Obtener() error = nil, want el error de systemctl")
	}
}

// TestSystemdDesplegarErrores cubre los fallos de transferencia y
// permisos.
func TestSystemdDesplegarErrores(t *testing.T) {
	t.Parallel()

	archivos := []ArchivoDespliegue{{Origen: "./app", Destino: "/opt/app", Modo: "755"}}

	casos := []struct {
		nombre string
		sesion *stubSession
	}{
		{"transferencia", &stubSession{putErr: &domain.ExitError{Code: 1, Command: "scp"}}},
		{"permisos", &stubSession{errs: []error{&domain.ExitError{Code: 1, Command: binChmod}}}},
		{"propietario", &stubSession{errs: []error{nil, &domain.ExitError{Code: 1, Command: binChown}}}},
	}

	for _, caso := range casos {
		systemd := nuevoSystemdDePrueba(caso.sesion, nil)

		if _, err := systemd.Desplegar(context.Background(), unidadPrueba, archivos, duenoPrueba, ""); err == nil {
			t.Errorf("%s: Desplegar() error = nil, want un error", caso.nombre)
		}
	}
}

// TestSystemdCambiarEstadoSinIndicacion comprueba el cuerpo vacío.
func TestSystemdCambiarEstadoSinIndicacion(t *testing.T) {
	t.Parallel()

	if _, err := nuevoSystemdDePrueba(&stubSession{}, nil).CambiarEstado(context.Background(), unidadPrueba, nil, nil); err == nil {
		t.Fatal("CambiarEstado() error = nil, want el error de estado no indicado")
	}
}

// TestSystemdSinSesion cubre la ausencia de sesión en las operaciones.
func TestSystemdSinSesion(t *testing.T) {
	t.Parallel()

	systemd := nuevoSystemdDePrueba(nil, ErrSesionNoDisponible)
	verdadero := true

	if _, err := systemd.Obtener(context.Background(), unidadPrueba); !errors.Is(err, ErrSesionNoDisponible) {
		t.Errorf(msgObtenerSinSesion, err)
	}

	if _, err := systemd.Desplegar(context.Background(), unidadPrueba, nil, "", ""); !errors.Is(err, ErrSesionNoDisponible) {
		t.Errorf("Desplegar() error = %v, want ErrSesionNoDisponible", err)
	}

	if _, err := systemd.CambiarEstado(context.Background(), unidadPrueba, &verdadero, nil); !errors.Is(err, ErrSesionNoDisponible) {
		t.Errorf("CambiarEstado() error = %v, want ErrSesionNoDisponible", err)
	}

	if _, err := systemd.Reiniciar(context.Background(), unidadPrueba); !errors.Is(err, ErrSesionNoDisponible) {
		t.Errorf("Reiniciar() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestMapearEstado comprueba la traducción de los estados.
func TestMapearEstado(t *testing.T) {
	t.Parallel()

	casos := map[string]string{
		estadoActivo:   estadoActivo,
		estadoInactivo: estadoInactivo,
		"failed":       "failed",
		"otro":         "unknown",
	}

	for entrada, esperado := range casos {
		if obtenido := mapearEstado(entrada); obtenido != esperado {
			t.Errorf("mapearEstado(%q) = %q, want %q", entrada, obtenido, esperado)
		}
	}
}
