// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"consola-vm/internal/core/domain"
)

const (
	serverSrv1     = "srv1"
	serverSrv2     = "srv2"
	stateUp        = "up"
	stateDown      = "down"
	stateUP        = "UP"
	stateDownMayus = "DOWN"
)

// configHaproxy simula una configuración con un front-end y un back-end.
const configHaproxy = `global
    daemon

defaults
    mode http

frontend http
    bind *:80
    default_backend app

backend app
    balance roundrobin
    server srv1 192.168.1.61:8080 check
`

// newHAProxyForTest arma el servicio con una sesión falsa.
func newHAProxyForTest(session *stubSession, err error) *HAProxy {
	return NewHAProxy(&stubOpener{session: session, err: err})
}

// TestHaproxyList interpreta los servidores del backend.
func TestHaproxyList(t *testing.T) {
	t.Parallel()

	session := &stubSession{result: domain.Result{Stdout: configHaproxy}}

	servers, err := newHAProxyForTest(session, nil).List(context.Background())
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	if len(servers) != 1 || servers[0].Address != ipServerTests || servers[0].Port != 8080 {
		t.Errorf("servidores = %+v, want 192.168.1.61:8080", servers)
	}
}

// TestHaproxyAdd modifica la configuración y recarga HAProxy.
func TestHaproxyAdd(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{{Stdout: configHaproxy}, {}}}
	haproxy := newHAProxyForTest(session, nil)

	result, err := haproxy.Add(context.Background(), ipServerNew, 9090)
	if err != nil {
		t.Fatalf("Agregar() error = %v, want nil", err)
	}

	if len(session.commands) != 2 {
		t.Fatalf(msgCommandDos, len(session.commands))
	}

	command := session.commands[1].ShellLine()
	for _, expected := range []string{"base64 -d", binHaproxy, "-c", "reload haproxy"} {
		if !strings.Contains(command, expected) {
			t.Errorf("orden = %q, want %q", command, expected)
		}
	}

	if result.Code != 0 {
		t.Errorf("Codigo = %d, want 0", result.Code)
	}
}

// TestHaproxyWriteKeepsFinalLF comprueba que la configuración enviada a
// HAProxy termina en salto de línea aunque la recibida no lo tenga.
func TestHaproxyWriteKeepsFinalLF(t *testing.T) {
	t.Parallel()

	session := &stubSession{result: domain.Result{}}

	if _, err := newHAProxyForTest(session, nil).write(
		context.Background(), session, "global\n    daemon", "Configuración escrita.",
	); err != nil {
		t.Fatalf("write() error = %v, want nil", err)
	}

	if len(session.commands) != 1 {
		t.Fatalf(msgCommandUnica, len(session.commands))
	}

	script := session.commands[0].Args[1]

	const prefix = "printf '%s' '"

	start := strings.Index(script, prefix)
	if start < 0 {
		t.Fatalf("orden = %q, want el Base64 precedido de printf", script)
	}

	encoded := script[start+len(prefix):]

	end := strings.Index(encoded, "'")
	if end < 0 {
		t.Fatalf("orden = %q, want el Base64 entre comillas", script)
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded[:end])
	if err != nil {
		t.Fatalf("decodificar Base64: %v", err)
	}

	if !strings.HasSuffix(string(decoded), "\n") {
		t.Errorf("configuración = %q, want el salto de línea final", decoded)
	}
}

// TestHaproxyAddDuplicate comprueba el servidor ya presente.
func TestHaproxyAddDuplicate(t *testing.T) {
	t.Parallel()

	session := &stubSession{result: domain.Result{Stdout: configHaproxy}}
	haproxy := newHAProxyForTest(session, nil)

	if _, err := haproxy.Add(context.Background(), ipServerTests, 8080); !errors.Is(err, ErrServerExists) {
		t.Fatalf("Agregar() error = %v, want ErrServidorExistente", err)
	}

	if len(session.commands) != 1 {
		t.Errorf("la sesión ejecutó %d órdenes, want solo la lectura", len(session.commands))
	}
}

// TestHaproxyWithoutBackend comprueba una configuración sin backend.
func TestHaproxyWithoutBackend(t *testing.T) {
	t.Parallel()

	session := &stubSession{result: domain.Result{Stdout: "global\n    daemon\n"}}

	if _, err := newHAProxyForTest(session, nil).List(context.Background()); !errors.Is(err, ErrBackendNotFound) {
		t.Fatalf("Listar() error = %v, want ErrBackendNoEncontrado", err)
	}
}

// TestHaproxyDelete quita el servidor de la configuración.
func TestHaproxyDelete(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{{Stdout: configHaproxy}, {}}}
	haproxy := newHAProxyForTest(session, nil)

	if _, err := haproxy.Delete(context.Background(), ipServerTests, 8080); err != nil {
		t.Fatalf(msgDeleteOK, err)
	}

	command := session.commands[1].ShellLine()
	if !strings.Contains(command, "base64 -d") {
		t.Errorf("orden = %q, want base64 -d", command)
	}
}

// TestHaproxyDeleteInexistente comprueba el servidor ausente.
func TestHaproxyDeleteInexistente(t *testing.T) {
	t.Parallel()

	session := &stubSession{result: domain.Result{Stdout: configHaproxy}}

	if _, err := newHAProxyForTest(session, nil).Delete(context.Background(), "192.168.1.99", 9999); !errors.Is(err, ErrServerNotFound) {
		t.Fatalf("Eliminar() error = %v, want ErrServidorNoEncontrado", err)
	}

	if len(session.commands) != 1 {
		t.Errorf("la sesión ejecutó %d órdenes, want solo la lectura", len(session.commands))
	}
}

// TestAddServerInsertaInBackend comprueba la línea generada.
func TestAddServerInsertaInBackend(t *testing.T) {
	t.Parallel()

	updated, err := addServer(configHaproxy, ipServerNew, 9090)
	if err != nil {
		t.Fatalf("agregarServidor() error = %v, want nil", err)
	}

	if !strings.Contains(updated, "    server srv_192_168_1_62_9090 192.168.1.62:9090 check") {
		t.Errorf("configuración = %q, want la línea del servidor", updated)
	}

	if strings.Contains(updated, "frontend http\n    server") {
		t.Errorf("configuración = %q, el servidor debe ir en el backend", updated)
	}
}

// TestRemoveServerDejaBackend comprueba la eliminación de la línea.
func TestRemoveServerDejaBackend(t *testing.T) {
	t.Parallel()

	updated, err := removeServer(configHaproxy, ipServerTests, 8080)
	if err != nil {
		t.Fatalf("quitarServidor() error = %v, want nil", err)
	}

	if strings.Contains(updated, serverSrv1) {
		t.Errorf("configuración = %q, el servidor no se quitó", updated)
	}

	if !strings.Contains(updated, "backend app") || !strings.Contains(updated, "balance roundrobin") {
		t.Errorf("configuración = %q, want el backend intacto", updated)
	}
}

// TestHaproxyAddErrorOfReload comprueba el fallo de recarga.
func TestHaproxyAddErrorOfReload(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		results: []domain.Result{{Stdout: configHaproxy}},
		result:  domain.Result{Stderr: "haproxy: invalid config", ExitCode: 1},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: binSh}},
	}

	result, err := newHAProxyForTest(session, nil).Add(context.Background(), ipServerNew, 9090)
	if err == nil {
		t.Fatal("Agregar() error = nil, want el error de validación")
	}

	if result.Code != 1 || result.Stderr == "" {
		t.Errorf(msgResultTechnical, result)
	}
}

// TestHaproxyWithoutSession comprueba la ausencia de sesión.
func TestHaproxyWithoutSession(t *testing.T) {
	t.Parallel()

	haproxy := newHAProxyForTest(nil, ErrSessionUnavailable)

	if _, err := haproxy.List(context.Background()); !errors.Is(err, ErrSessionUnavailable) {
		t.Fatalf("Listar() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestHaproxyErrorsOfSession cubre la ausencia de sesión y la lectura de
// la configuración.
func TestHaproxyErrorsOfSession(t *testing.T) {
	t.Parallel()

	withoutSession := newHAProxyForTest(nil, ErrSessionUnavailable)

	if _, err := withoutSession.Add(context.Background(), ipServerTests, 8080); !errors.Is(err, ErrSessionUnavailable) {
		t.Errorf("Agregar() error = %v, want ErrSesionNoDisponible", err)
	}

	if _, err := withoutSession.Delete(context.Background(), ipServerTests, 8080); !errors.Is(err, ErrSessionUnavailable) {
		t.Errorf("Eliminar() error = %v, want ErrSesionNoDisponible", err)
	}

	withError := newHAProxyForTest(&stubSession{err: &domain.ExitError{Code: 1, Command: binCat}}, nil)

	if _, err := withError.List(context.Background()); err == nil {
		t.Error("Listar() error = nil, want el error de lectura")
	}

	if _, err := withError.Add(context.Background(), ipServerTests, 8080); err == nil {
		t.Error("Agregar() error = nil, want el error de lectura")
	}
}

// TestHaproxyDeleteErrorOfReload cubre el fallo al recargar HAProxy.
func TestHaproxyDeleteErrorOfReload(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		results: []domain.Result{{Stdout: configHaproxy}},
		result:  domain.Result{Stderr: "reload failed", ExitCode: 1},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: binSh}},
	}

	if _, err := newHAProxyForTest(session, nil).Delete(context.Background(), ipServerTests, 8080); err == nil {
		t.Fatal("Eliminar() error = nil, want el error de recarga")
	}
}

// statsRow construye una fila CSV del socket de estadísticas con el
// estado indicado.
func statsRow(name, state string) string {
	return "app," + name + strings.Repeat(",", 16) + state + ",1,1,1"
}

// estadisticasHaproxy simula la salida de «show stat» con dos servidores.
var estadisticasHaproxy = strings.Join([]string{
	"# pxname,svname,qcur,qmax,scur,smax,slim,stot,bin,bout,status",
	statsRow("FRONTEND", "OPEN"),
	statsRow(serverSrv1, stateUP),
	statsRow(serverSrv2, stateDownMayus),
	"app,BACKEND,,,,,,,,,,UP",
}, "\n")

// TestHaproxyListWithState toma el estado de cada servidor del socket de
// estadísticas.
func TestHaproxyListWithState(t *testing.T) {
	t.Parallel()

	config := configHaproxy + "    server srv2 192.168.1.62:9090 check\n"
	session := &stubSession{results: []domain.Result{
		{Stdout: config},
		{Stdout: estadisticasHaproxy},
	}}

	servers, err := newHAProxyForTest(session, nil).List(context.Background())
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	if len(servers) != 2 {
		t.Fatalf("Listar() devolvió %d servidores, want 2", len(servers))
	}

	if servers[0].State != stateUp || servers[1].State != stateDown {
		t.Errorf("estados = %q, %q, want up y down", servers[0].State, servers[1].State)
	}
}

// TestHaproxyListWithoutSocketDejaStateUnknown comprueba que el
// listado sigue funcionando cuando el socket no responde.
func TestHaproxyListWithoutSocketDejaStateUnknown(t *testing.T) {
	t.Parallel()

	session := &stubSession{
		results: []domain.Result{{Stdout: configHaproxy}},
		errs:    []error{nil, errors.New("socat no está instalado")},
	}

	servers, err := newHAProxyForTest(session, nil).List(context.Background())
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	if len(servers) != 1 || servers[0].State != "unknown" {
		t.Errorf("servidores = %+v, want un servidor con estado unknown", servers)
	}
}

// TestHaproxyListWithoutServersNotConsultaSocket evita la consulta cuando
// el back-end está vacío.
func TestHaproxyListWithoutServersNotConsultaSocket(t *testing.T) {
	t.Parallel()

	config := "backend app\n    balance roundrobin\n"
	session := &stubSession{results: []domain.Result{{Stdout: config}}}

	servers, err := newHAProxyForTest(session, nil).List(context.Background())
	if err != nil {
		t.Fatalf(msgListOK, err)
	}

	if len(servers) != 0 {
		t.Errorf("servidores = %+v, want una lista vacía", servers)
	}

	if len(session.commands) != 1 {
		t.Errorf("órdenes ejecutadas = %d, want 1 (solo leer la configuración)", len(session.commands))
	}
}

// TestParseStates traduce el campo status de la salida CSV.
func TestParseStates(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		output  string
		esperar map[string]string
	}{
		{"vacía", "", map[string]string{}},
		{"solo cabecera", "# pxname,svname,status", map[string]string{}},
		{"fila corta", "app,srv1,UP", map[string]string{}},
		{
			"estados conocidos",
			strings.Join([]string{
				statsRow(serverSrv1, stateUP),
				statsRow(serverSrv2, stateDownMayus),
				statsRow("srv3", "MAINT"),
				statsRow("srv4", "DRAIN"),
			}, "\n"),
			map[string]string{serverSrv1: stateUp, serverSrv2: stateDown, "srv3": stateDown, "srv4": stateDown},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			states := parseStates(testCase.output)
			if len(states) != len(testCase.esperar) {
				t.Fatalf("estados = %v, want %v", states, testCase.esperar)
			}

			for name, state := range testCase.esperar {
				if states[name] != state {
					t.Errorf("estado de %s = %q, want %q", name, states[name], state)
				}
			}
		})
	}
}
