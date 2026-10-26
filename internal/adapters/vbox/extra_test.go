// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package vbox

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"consola-vm/internal/core/domain"
)

// TestVersion devuelve la versión del hipervisor.
func TestVersion(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["--version"] = domain.Result{Stdout: "7.2.18r\n"}

	version, err := New(fake).Version(context.Background())
	if err != nil || version != "7.2.18r" {
		t.Errorf("Version() = %q, %v, want 7.2.18r", version, err)
	}

	withError := newFakeExecutor()
	withError.errs = map[string]error{"--version": errors.New("sin VBoxManage")}

	if _, err := New(withError).Version(context.Background()); err == nil {
		t.Error("Version() error = nil, want el fallo de VBoxManage")
	}
}

// TestGuestAdditions lee la versión, el nivel y los usuarios conectados.
func TestGuestAdditions(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["showvminfo"] = domain.Result{Stdout: vmRunning + "\n" +
		`GuestAdditionsVersion="6.0.0"` + "\n" + `GuestAdditionsRunLevel=2` + "\n"}
	fake.results["guestproperty"] = domain.Result{Stdout: "Value: jorge,andrey\n"}

	guest, err := New(fake).GuestAdditions(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf("GuestAdditions() error = %v, want nil", err)
	}

	if guest.Version != "6.0.0" || guest.RunLevel != 2 {
		t.Errorf("GuestAdditions() = %+v, want 6.0.0 nivel 2", guest)
	}

	if len(guest.LoggedUsers) != 2 || guest.LoggedUsers[0] != "jorge" {
		t.Errorf("usuarios = %v, want jorge y andrey", guest.LoggedUsers)
	}

	// Si la consulta de usuarios falla, el resto del informe se conserva.
	fake.errs = map[string]error{"guestproperty": errors.New("ocupado")}

	guest, err = New(fake).GuestAdditions(context.Background(), domain.Target{ID: vmID})
	if err != nil || len(guest.LoggedUsers) != 0 {
		t.Errorf("GuestAdditions() sin usuarios = %+v, %v, want vacío sin error", guest, err)
	}

	withoutData := newFakeExecutor()
	withoutData.errs = map[string]error{cmdShowVMInfo: errors.New("sin VBoxManage")}

	if _, err := New(withoutData).GuestAdditions(context.Background(), domain.Target{ID: vmID}); err == nil {
		t.Error("GuestAdditions() error = nil, want el fallo de VBoxManage")
	}
}

// TestValueProperty interpreta las propiedades de showvminfo.
func TestValueProperty(t *testing.T) {
	t.Parallel()

	output := "  GuestAdditionsVersion=\"6.0.0\"\nOtra=1\n"

	if value := valueProperty(output, "GuestAdditionsVersion"); value != "6.0.0" {
		t.Errorf("valorPropiedad() = %q, want 6.0.0", value)
	}

	if value := valueProperty(output, "Inexistente"); value != "" {
		t.Errorf("valorPropiedad() = %q, want vacío", value)
	}

	if number := enteroProperty(output, "Otra"); number != 1 {
		t.Errorf("enteroPropiedad() = %d, want 1", number)
	}

	if number := enteroProperty("Otra=abc", "Otra"); number != 0 {
		t.Errorf("enteroPropiedad() con valor inválido = %d, want 0", number)
	}

	if number := enteroProperty("Otra=1", "Otra"); number != 1 {
		t.Errorf("enteroPropiedad() sin espacios = %d, want 1", number)
	}

	if number := enteroProperty(output, "Inexistente"); number != 0 {
		t.Errorf("enteroPropiedad() ausente = %d, want 0", number)
	}
}

// TestParseUsers interpreta la lista de usuarios conectados.
func TestParseUsers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		output string
		want   int
	}{
		{"Value: jorge,andrey", 2},
		{"Value: jorge,,andrey,", 2},
		{"Value:", 0},
		{"sin la propiedad", 0},
	}

	for _, testCase := range testCases {
		if users := parseUsers(testCase.output); len(users) != testCase.want {
			t.Errorf("parsearUsuarios(%q) = %v, want %d", testCase.output, users, testCase.want)
		}
	}
}

// TestCloseNotMakesNada comprueba que la sesión de guestcontrol no guarda
// recursos.
func TestCloseNotMakesNada(t *testing.T) {
	t.Parallel()

	session := &Session{}

	if err := session.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

// TestSessionRunPropagatesFailure devuelve el error del hipervisor.
func TestSessionRunPropagatesFailure(t *testing.T) {
	t.Parallel()

	fake := runningFake()
	fake.errs = map[string]error{"guestcontrol": &domain.ExitError{Code: 1, Command: "VBoxManage"}}

	session, err := New(fake).Connect(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	if _, err := session.Run(context.Background(), &domain.Command{Path: "/usr/bin/id"}); err == nil {
		t.Error("Run() error = nil, want el fallo de guestcontrol")
	}
}

// TestSessionPutPropagatesFailure devuelve el error de la transferencia.
func TestSessionPutPropagatesFailure(t *testing.T) {
	t.Parallel()

	fake := runningFake()
	fake.errs = map[string]error{subGuestControl: &domain.ExitError{Code: 1, Command: "VBoxManage"}}

	session, err := New(fake).Connect(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	if err := session.Put(context.Background(), "/tmp/a", "/tmp/b"); err == nil {
		t.Error("Put() error = nil, want el fallo de copyto")
	}
}

// TestErrorSanitizedKeepsOtherErrors deja intacto un error que no
// proviene de un código de salida.
func TestErrorSanitizedKeepsOtherErrors(t *testing.T) {
	t.Parallel()

	original := errors.New("sin conexión")

	if err := sanitizeError(original, "comando"); !errors.Is(err, original) {
		t.Errorf("errorSanitizado() = %v, want el error original", err)
	}
}

// TestParseStateTranslatesStates cubre el vocabulario de VirtualBox.
func TestParseStateTranslatesStates(t *testing.T) {
	t.Parallel()

	testCases := map[string]domain.MachineState{
		`VMState="running"`:  domain.StateRunning,
		`VMState="poweroff"`: domain.StatePoweredOff,
		`VMState="aborted"`:  domain.StatePoweredOff,
		`VMState="paused"`:   domain.StatePaused,
		`VMState="saved"`:    domain.StateSaved,
		`VMState="otro"`:     domain.StateUnknown,
		"sin estado":         domain.StateUnknown,
	}

	for output, expected := range testCases {
		if state := parseState(output); state != expected {
			t.Errorf("parseState(%q) = %q, want %q", output, state, expected)
		}
	}
}

// TestParseNetworkUnknown informa desconocido ante un modo ajeno.
func TestParseNetworkUnknown(t *testing.T) {
	t.Parallel()

	if mode := parseNetwork(`nic1="otro"`); mode != domain.NetworkUnknown {
		t.Errorf("parseNetwork() = %q, want desconocido", mode)
	}

	if mode := parseNetwork("sin interfaz"); mode != domain.NetworkUnknown {
		t.Errorf("parseNetwork() = %q, want desconocido", mode)
	}
}

// TestStateError devuelve el fallo del hipervisor.
func TestStateError(t *testing.T) {
	t.Parallel()

	withError := newFakeExecutor()
	withError.errs = map[string]error{cmdShowVMInfo: errors.New("sin VBoxManage")}

	if _, err := New(withError).State(context.Background(), domain.Target{ID: vmID}); err == nil {
		t.Error("State() error = nil, want el fallo de VBoxManage")
	}
}

// TestNetworkError devuelve el fallo del hipervisor.
func TestNetworkError(t *testing.T) {
	t.Parallel()

	withError := newFakeExecutor()
	withError.errs = map[string]error{cmdShowVMInfo: errors.New("sin VBoxManage")}

	if _, err := New(withError).Network(context.Background(), domain.Target{ID: vmID}); err == nil {
		t.Error("Network() error = nil, want el fallo de VBoxManage")
	}
}

// TestProbePropagatesFailureOfState no oculta el error de showvminfo.
func TestProbePropagatesFailureOfState(t *testing.T) {
	t.Parallel()

	withError := newFakeExecutor()
	withError.errs = map[string]error{cmdShowVMInfo: errors.New("sin VBoxManage")}

	if err := New(withError).Probe(context.Background(), domain.Target{ID: vmID}); err == nil {
		t.Error("Probe() error = nil, want el fallo de VBoxManage")
	}
}

// TestConnectPropagatesFailureOfProbe no abre sesión si la máquina no
// responde.
func TestConnectPropagatesFailureOfProbe(t *testing.T) {
	t.Parallel()

	withError := newFakeExecutor()
	withError.errs = map[string]error{cmdShowVMInfo: errors.New("sin VBoxManage")}

	if _, err := New(withError).Connect(context.Background(), domain.Target{ID: vmID}); err == nil {
		t.Error("Connect() error = nil, want el fallo de VBoxManage")
	}
}

// TestReintentableWithoutArguments no reintenta una orden desconocida.
func TestReintentableWithoutArguments(t *testing.T) {
	t.Parallel()

	if reintentable(nil) {
		t.Error("reintentable(nil) = true, want false")
	}

	if reintentable([]string{"startvm", vmID}) {
		t.Error("reintentable(startvm) = true, want false")
	}

	if !reintentable([]string{"--version"}) {
		t.Error("reintentable(--version) = false, want true")
	}
}

// TestRunEsperaCancelada abandona el reintento si la petición se
// cancela mientras espera.
func TestRunEsperaCancelada(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.errs = map[string]error{cmdShowVMInfo: errors.New("ocupado")}

	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelar()

	if _, err := New(fake).run(ctx, cmdShowVMInfo, vmID); err == nil {
		t.Error("ejecutar() error = nil, want el fallo tras cancelar la espera")
	}
}

// TestRunReintentaYFails devuelve el error del segundo intento.
func TestRunReintentaYFails(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.errs = map[string]error{cmdShowVMInfo: errors.New("ocupado")}

	if _, err := New(fake).run(context.Background(), cmdShowVMInfo, vmID); err == nil {
		t.Error("ejecutar() error = nil, want el fallo del reintento")
	}

	if len(fake.calls) != 2 {
		t.Errorf("llamadas = %d, want 2 intentos", len(fake.calls))
	}
}

// TestNewValuesByOmision comprueba el binario y el usuario iniciales.
func TestNewValuesByOmision(t *testing.T) {
	t.Parallel()

	client := New(newFakeExecutor())
	if client.bin != defaultBinary || client.user != "root" {
		t.Errorf("New() = %q, %q, want %q, root", client.bin, client.user, defaultBinary)
	}
}

// fakeReintento falla la primera vez y responde bien la segunda.
type fakeReintento struct {
	intentos int
}

// Execute simula el bloqueo momentáneo de VBoxManage.
func (f *fakeReintento) Execute(_ context.Context, _ string, _ ...string) (domain.Result, error) {
	f.intentos++

	if f.intentos == 1 {
		return domain.Result{}, errors.New("ocupado")
	}

	return domain.Result{Stdout: vmRunning}, nil
}

// TestRunReintentaYTriunfa devuelve el resultado del segundo intento.
func TestRunReintentaYTriunfa(t *testing.T) {
	t.Parallel()

	fake := &fakeReintento{}

	result, err := New(fake).run(context.Background(), cmdShowVMInfo, vmID)
	if err != nil {
		t.Fatalf("ejecutar() error = %v, want nil", err)
	}

	if result.Stdout != vmRunning || fake.intentos != 2 {
		t.Errorf("ejecutar() = %q tras %d intentos, want el segundo resultado",
			result.Stdout, fake.intentos)
	}
}

// TestSessionRunPasaDirectoryOfTrabajo agrega --cwd a la orden.
func TestSessionRunPasaDirectoryOfTrabajo(t *testing.T) {
	t.Parallel()

	fake := runningFake()

	session, err := New(fake).Connect(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	if _, err := session.Run(context.Background(), &domain.Command{
		Path: "/usr/bin/date",
		Dir:  "/tmp",
	}); err != nil {
		t.Fatalf(msgRunOK, err)
	}

	llamada := strings.Join(fake.lastCall(), " ")
	if !strings.Contains(llamada, "--cwd /tmp") {
		t.Errorf("Run() invocó %q, want --cwd /tmp", llamada)
	}
}

// TestParseAddressIgnoresLinesForeign y falla si no hay propiedad.
func TestParseAddressIgnoresLinesForeign(t *testing.T) {
	t.Parallel()

	if _, err := parseAddress("Address: 10.0.2.15"); err == nil {
		t.Error("parseAddress() error = nil, want el error de propiedad ausente")
	}

	if _, err := parseAddress(""); err == nil {
		t.Error("parseAddress() error = nil, want el error de propiedad ausente")
	}

	addr, err := parseAddress("IP: 10.0.2.15\nValue: 10.0.2.16\n")
	if err != nil || addr.String() != "10.0.2.16" {
		t.Errorf("parseAddress() = %q, %v, want 10.0.2.16", addr, err)
	}
}
