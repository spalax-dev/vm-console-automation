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

// unitTests es la unidad usada en las pruebas.
const unitTests = "catalogo.service"

// outputUnits simula systemctl list-units.
const outputUnits = "catalogo.service loaded active running Catálogo\n" +
	"ssh.service loaded active running OpenBSD Secure Shell server\n" +
	"cron.service loaded inactive dead Regular background daemon\n"

// outputUnitFiles simula systemctl list-unit-files.
const outputUnitFiles = "catalogo.service enabled\n" +
	"ssh.service enabled\n" +
	"cron.service disabled\n"

// newSystemdOfTests arma el servicio con una sesión falsa.
func newSystemdOfTests(session *stubSession, err error) *Systemd {
	return NewSystemd(&stubOpener{session: session, err: err})
}

// TestSystemdList combina unidades y habilitación.
func TestSystemdList(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{
		{Stdout: outputUnits},
		{Stdout: outputUnitFiles},
	}}

	lists, err := newSystemdOfTests(session, nil).List(context.Background())
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	if len(lists) != 3 {
		t.Fatalf("Listar() devolvió %d unidades, want 3", len(lists))
	}

	if !lists[0].Active || !lists[0].Enabled || lists[0].State != stateActive || lists[0].Detail != "running" {
		t.Errorf("catalogo = %+v, want activa, habilitada y running", lists[0])
	}

	if lists[2].Active || lists[2].Enabled || lists[2].State != stateInactive {
		t.Errorf("cron = %+v, want inactiva y deshabilitada", lists[2])
	}
}

// TestSystemdGet interpreta systemctl show.
func TestSystemdGet(t *testing.T) {
	t.Parallel()

	session := &stubSession{result: domain.Result{Stdout: "LoadState=loaded\nActiveState=active\nUnitFileState=enabled\nSubState=running\n"}}

	state, err := newSystemdOfTests(session, nil).Get(context.Background(), unitTests)
	if err != nil {
		t.Fatalf("Obtener() error = %v, want nil", err)
	}

	if !state.Active || !state.Enabled || state.State != stateActive || state.Detail != "running" {
		t.Errorf("estado = %+v, want activa, habilitada y running", state)
	}
}

// TestSystemdGetNotFound comprueba LoadState=not-found.
func TestSystemdGetNotFound(t *testing.T) {
	t.Parallel()

	session := &stubSession{result: domain.Result{Stdout: "LoadState=not-found\n"}}

	if _, err := newSystemdOfTests(session, nil).Get(context.Background(), unitTests); !errors.Is(err, ErrUnitNotFound) {
		t.Fatalf("Obtener() error = %v, want ErrUnidadNoEncontrada", err)
	}
}

// TestSystemdGetCodeCuatro comprueba el código 4 de systemctl.
func TestSystemdGetCodeCuatro(t *testing.T) {
	t.Parallel()

	session := &stubSession{err: &domain.ExitError{Code: 4, Command: binSystemctl}}

	if _, err := newSystemdOfTests(session, nil).Get(context.Background(), unitTests); !errors.Is(err, ErrUnitNotFound) {
		t.Fatalf("Obtener() error = %v, want ErrUnidadNoEncontrada", err)
	}
}

// TestSystemdDeploy transfiere, ajusta permisos y recarga systemd.
func TestSystemdDeploy(t *testing.T) {
	t.Parallel()

	session := &stubSession{}
	systemd := newSystemdOfTests(session, nil)

	result, err := systemd.Deploy(context.Background(), unitTests, []DeploymentFile{
		{Source: "./app/catalogo", Destination: "/opt/catalogo/catalogo", Mode: "755"},
	}, duenoTests, duenoTests)
	if err != nil {
		t.Fatalf("Desplegar() error = %v, want nil", err)
	}

	if len(session.puts) != 1 || session.puts[0][1] != "/opt/catalogo/catalogo" {
		t.Errorf("Put() = %v, want el archivo transferido", session.puts)
	}

	if len(session.commands) != 3 {
		t.Fatalf("la sesión ejecutó %d órdenes, want 3", len(session.commands))
	}

	if session.commands[0].Path != binChmod || strings.Join(session.commands[0].Args, " ") != "755 /opt/catalogo/catalogo" {
		t.Errorf("chmod = %v, want 755 en el destino", session.commands[0])
	}

	if session.commands[1].Path != binChown || session.commands[1].Args[0] != "catalogo:catalogo" {
		t.Errorf("chown = %v, want catalogo:catalogo", session.commands[1])
	}

	if session.commands[2].Path != binSystemctl || session.commands[2].Args[0] != "daemon-reload" {
		t.Errorf("systemctl = %v, want daemon-reload", session.commands[2])
	}

	if result.Code != 0 {
		t.Errorf(msgCodeCero, result.Code)
	}
}

// TestSystemdChangeState cubre start, stop, enable y disable.
func TestSystemdChangeState(t *testing.T) {
	t.Parallel()

	verdadero := true
	fake := false

	testCases := []struct {
		name     string
		active   *bool
		enabled  *bool
		expected string
	}{
		{"start", &verdadero, nil, "start"},
		{"stop", &fake, nil, "stop"},
		{"enable", nil, &verdadero, "enable"},
		{"disable", nil, &fake, "disable"},
	}

	for _, testCase := range testCases {
		session := &stubSession{}
		systemd := newSystemdOfTests(session, nil)

		if _, err := systemd.ChangeState(context.Background(), unitTests, testCase.active, testCase.enabled); err != nil {
			t.Fatalf("%s: CambiarEstado() error = %v", testCase.name, err)
		}

		cmd := session.commands[0]
		if cmd.Path != binSystemctl || cmd.Args[0] != testCase.expected || cmd.Args[1] != unitTests {
			t.Errorf("%s: orden = %v, want %s %s", testCase.name, cmd.Args, testCase.expected, unitTests)
		}
	}
}

// TestSystemdRestart comprueba systemctl restart.
func TestSystemdRestart(t *testing.T) {
	t.Parallel()

	session := &stubSession{}

	if _, err := newSystemdOfTests(session, nil).Restart(context.Background(), unitTests); err != nil {
		t.Fatalf("Reiniciar() error = %v, want nil", err)
	}

	cmd := session.commands[0]
	if cmd.Args[0] != "restart" || cmd.Args[1] != unitTests {
		t.Errorf("orden = %v, want restart %s", cmd.Args, unitTests)
	}
}

// TestSystemdListWithoutSession comprueba la ausencia de sesión.
func TestSystemdListWithoutSession(t *testing.T) {
	t.Parallel()

	systemd := newSystemdOfTests(nil, ErrSessionUnavailable)

	if _, err := systemd.List(context.Background()); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf(msgListWithoutSession, err)
	}
}

// TestSystemdChangeStateError comprueba el fallo de systemctl.
func TestSystemdChangeStateError(t *testing.T) {
	t.Parallel()

	verdadero := true
	session := &stubSession{
		result: domain.Result{Stderr: "Unit not found", ExitCode: 5},
		err:    &domain.ExitError{Code: 5, Command: binSystemctl},
	}

	result, err := newSystemdOfTests(session, nil).ChangeState(context.Background(), unitTests, &verdadero, nil)
	if err == nil {
		t.Fatal("CambiarEstado() error = nil, want el error de systemctl")
	}

	if result.Code != 5 || result.Stderr == "" {
		t.Errorf(msgResultTechnical, result)
	}
}

// TestSystemdListError comprueba el fallo al listar unidades.
func TestSystemdListError(t *testing.T) {
	t.Parallel()

	session := &stubSession{err: &domain.ExitError{Code: 1, Command: binSystemctl}}

	if _, err := newSystemdOfTests(session, nil).List(context.Background()); err == nil {
		t.Fatal("Listar() error = nil, want el error de systemctl")
	}
}

// TestSystemdListErrorOfFiles comprueba el fallo al consultar la
// habilitación.
func TestSystemdListErrorOfFiles(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		results: []domain.Result{{Stdout: outputUnits}},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: binSystemctl}},
	}

	if _, err := newSystemdOfTests(session, nil).List(context.Background()); err == nil {
		t.Fatal("Listar() error = nil, want el error de list-unit-files")
	}
}

// TestSystemdGetError comprueba un fallo inesperado de systemctl.
func TestSystemdGetError(t *testing.T) {
	t.Parallel()

	session := &stubSession{err: &domain.ExitError{Code: 1, Command: binSystemctl}}

	if _, err := newSystemdOfTests(session, nil).Get(context.Background(), unitTests); err == nil {
		t.Fatal("Obtener() error = nil, want el error de systemctl")
	}
}

// TestSystemdDeployErrors cubre los fallos de transferencia y
// permisos.
func TestSystemdDeployErrors(t *testing.T) {
	t.Parallel()

	files := []DeploymentFile{{Source: "./app", Destination: "/opt/app", Mode: "755"}}

	testCases := []struct {
		name    string
		session *stubSession
	}{
		{"transferencia", &stubSession{putErr: &domain.ExitError{Code: 1, Command: "scp"}}},
		{"permisos", &stubSession{errs: []error{&domain.ExitError{Code: 1, Command: binChmod}}}},
		{"propietario", &stubSession{errs: []error{nil, &domain.ExitError{Code: 1, Command: binChown}}}},
	}

	for _, testCase := range testCases {
		systemd := newSystemdOfTests(testCase.session, nil)

		if _, err := systemd.Deploy(context.Background(), unitTests, files, duenoTests, ""); err == nil {
			t.Errorf("%s: Desplegar() error = nil, want un error", testCase.name)
		}
	}
}

// TestSystemdChangeStateWithoutIndicacion comprueba el cuerpo vacío.
func TestSystemdChangeStateWithoutIndicacion(t *testing.T) {
	t.Parallel()

	if _, err := newSystemdOfTests(&stubSession{}, nil).ChangeState(context.Background(), unitTests, nil, nil); err == nil {
		t.Fatal("CambiarEstado() error = nil, want el error de estado no indicado")
	}
}

// TestSystemdWithoutSession cubre la ausencia de sesión en las operaciones.
func TestSystemdWithoutSession(t *testing.T) {
	t.Parallel()

	systemd := newSystemdOfTests(nil, ErrSessionUnavailable)
	verdadero := true

	if _, err := systemd.Get(context.Background(), unitTests); !errors.Is(err, ErrSessionUnavailable) {
		t.Errorf(msgGetWithoutSession, err)
	}

	if _, err := systemd.Deploy(context.Background(), unitTests, nil, "", ""); !errors.Is(err, ErrSessionUnavailable) {
		t.Errorf("Desplegar() error = %v, want ErrSesionNoDisponible", err)
	}

	if _, err := systemd.ChangeState(context.Background(), unitTests, &verdadero, nil); !errors.Is(err, ErrSessionUnavailable) {
		t.Errorf("CambiarEstado() error = %v, want ErrSesionNoDisponible", err)
	}

	if _, err := systemd.Restart(context.Background(), unitTests); !errors.Is(err, ErrSessionUnavailable) {
		t.Errorf("Reiniciar() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestMapState comprueba la traducción de los estados.
func TestMapState(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		stateActive:   stateActive,
		stateInactive: stateInactive,
		"failed":      "failed",
		"otro":        "unknown",
	}

	for entry, expected := range testCases {
		if obtained := mapState(entry); obtained != expected {
			t.Errorf("mapearEstado(%q) = %q, want %q", entry, obtained, expected)
		}
	}
}

// TestSystemdDeployWithoutPropietario comprueba que sin usuario no se
// ejecuta chown y que el despliegue sigue recargando systemd.
func TestSystemdDeployWithoutPropietario(t *testing.T) {
	t.Parallel()

	session := &stubSession{}
	systemd := newSystemdOfTests(session, nil)

	if _, err := systemd.Deploy(context.Background(), unitTests, []DeploymentFile{
		{Source: "./app", Destination: "/opt/app", Mode: "755"},
	}, "", ""); err != nil {
		t.Fatalf("Desplegar() error = %v, want nil", err)
	}

	if len(session.puts) != 1 {
		t.Errorf("Put() se llamó %d veces, want 1", len(session.puts))
	}

	for _, cmd := range session.commands {
		if cmd.Path == binChown {
			t.Errorf("se ejecutó chown sin propietario: %v", cmd.Args)
		}
	}

	if len(session.commands) != 2 {
		t.Fatalf("la sesión ejecutó %d órdenes, want chmod y daemon-reload", len(session.commands))
	}

	if session.commands[1].Path != binSystemctl || session.commands[1].Args[0] != "daemon-reload" {
		t.Errorf("última orden = %v, want systemctl daemon-reload", session.commands[1])
	}
}

// TestSystemdStatus ejecuta systemctl status y conserva su salida.
func TestSystemdStatus(t *testing.T) {
	t.Parallel()

	session := &stubSession{result: domain.Result{
		Stdout: "● anacron.service - Anacron\n     Active: active (running)\n",
	}}
	systemd := newSystemdOfTests(session, nil)

	result, err := systemd.Status(context.Background(), unitTests)
	if err != nil {
		t.Fatalf("Status() error = %v, want nil", err)
	}

	if len(session.commands) != 1 {
		t.Fatalf("la sesión ejecutó %d órdenes, want 1", len(session.commands))
	}

	cmd := session.commands[0]
	if cmd.Path != binSystemctl || strings.Join(cmd.Args, " ") != "status "+unitTests+" --no-pager --full" {
		t.Errorf("orden = %v, want systemctl status %s --no-pager --full", cmd.Args, unitTests)
	}

	if result.Code != 0 {
		t.Errorf("Code = %d, want 0", result.Code)
	}

	if !strings.Contains(result.Output, "active (running)") {
		t.Errorf("Output = %q, want la salida de systemctl status", result.Output)
	}

	if result.Message == "" || result.Operation == "" {
		t.Errorf("resultado = %+v, want mensaje y operación", result)
	}

	if !session.closed {
		t.Error(msgSessionOpen)
	}
}

// TestSystemdStatusInactive acepta el código 3 de systemctl status.
func TestSystemdStatusInactive(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		result: domain.Result{Stdout: "● anacron.service - Anacron\n     Active: inactive (dead)\n", ExitCode: 3},
		err:    &domain.ExitError{Code: 3, Command: binSystemctl},
	}

	result, err := newSystemdOfTests(session, nil).Status(context.Background(), unitTests)
	if err != nil {
		t.Fatalf("Status() error = %v, want nil para una unidad inactiva", err)
	}

	if result.Code != 0 {
		t.Errorf("Code = %d, want 0 (la consulta HTTP fue correcta)", result.Code)
	}

	if !strings.Contains(result.Output, "inactive (dead)") {
		t.Errorf("Output = %q, want la salida de la unidad inactiva", result.Output)
	}
}

// TestSystemdStatusNotFound traduce el código 4 a ErrUnitNotFound.
func TestSystemdStatusNotFound(t *testing.T) {
	t.Parallel()

	session := &stubSession{err: &domain.ExitError{Code: 4, Command: binSystemctl}}

	if _, err := newSystemdOfTests(session, nil).Status(context.Background(), unitTests); !errors.Is(err, ErrUnitNotFound) {
		t.Fatalf("Status() error = %v, want ErrUnitNotFound", err)
	}
}

// TestSystemdStatusError propaga un fallo inesperado de systemctl.
func TestSystemdStatusError(t *testing.T) {
	t.Parallel()

	session := &stubSession{err: errors.New("systemctl caído")}

	if _, err := newSystemdOfTests(session, nil).Status(context.Background(), unitTests); err == nil {
		t.Fatal("Status() error = nil, want el fallo de systemctl")
	}
}

// TestSystemdStatusWithoutSession propaga la ausencia de sesión.
func TestSystemdStatusWithoutSession(t *testing.T) {
	t.Parallel()

	systemd := newSystemdOfTests(nil, ErrSessionUnavailable)

	if _, err := systemd.Status(context.Background(), unitTests); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("Status() error = %v, want ErrSessionUnavailable", err)
	}
}
