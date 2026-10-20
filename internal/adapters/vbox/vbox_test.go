// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package vbox

import (
	"context"
	"errors"
	"testing"

	"consola-vm/internal/core/domain"
)

const (
	vmID             = "debian13"
	vmRunning        = `VMState="running"`
	salidaInstalado  = "instalado\n"
	credRoot         = "root"
	credPass         = "secreto"
	binAptGet        = "/usr/bin/apt-get"
	msgConnectFail   = "Connect() error = %v, want nil"
	msgRunArgsPrefix = "Run() invocó"
)

// fakeExecutor registra las órdenes ejecutadas y devuelve respuestas
// preprogramadas, para probar el adaptador sin VirtualBox instalado.
type fakeExecutor struct {
	calls   [][]string
	results map[string]domain.Result
	errs    map[string]error
}

func newFakeExecutor() *fakeExecutor {
	return &fakeExecutor{
		results: make(map[string]domain.Result),
		errs:    make(map[string]error),
	}
}

func (f *fakeExecutor) Execute(_ context.Context, program string, args ...string) (domain.Result, error) {
	call := append([]string{program}, args...)
	f.calls = append(f.calls, call)

	key := call[1]

	return f.results[key], f.errs[key]
}

func (f *fakeExecutor) lastCall() []string {
	if len(f.calls) == 0 {
		return nil
	}

	return f.calls[len(f.calls)-1]
}

func runningFake() *fakeExecutor {
	fake := newFakeExecutor()
	fake.results["showvminfo"] = domain.Result{Stdout: vmRunning}

	return fake
}

func TestAddressParsesGuestPropertyValue(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["guestproperty"] = domain.Result{Stdout: "Value: 10.0.2.15\n"}

	client := New(fake)

	addr, err := client.Address(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf("Address() error = %v, want nil", err)
	}

	if addr.String() != "10.0.2.15" {
		t.Errorf("Address() = %q, want %q", addr.String(), "10.0.2.15")
	}
}

func TestAddressErrorWhenPropertyMissing(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["guestproperty"] = domain.Result{Stdout: "Value: \n"}
	fake.errs["guestproperty"] = errors.New("VERR_NOT_FOUND")

	client := New(fake)

	if _, err := client.Address(context.Background(), domain.Target{ID: vmID}); err == nil {
		t.Fatal("Address() error = nil, want un error cuando la propiedad no existe")
	}
}

func TestAddressErrorWhenOutputIsNotAnAddress(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["guestproperty"] = domain.Result{Stdout: "Value: no-es-una-ip\n"}

	client := New(fake)

	if _, err := client.Address(context.Background(), domain.Target{ID: vmID}); err == nil {
		t.Fatal("Address() error = nil, want un error cuando el valor no es una dirección")
	}
}

func TestNameIsGuestAdditions(t *testing.T) {
	t.Parallel()

	client := New(newFakeExecutor())

	if got := client.Name(); got != "guest-additions" {
		t.Errorf("Name() = %q, want %q", got, "guest-additions")
	}
}

func TestProbeErrorWhenMachineNotRunning(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["showvminfo"] = domain.Result{Stdout: `VMState="poweroff"`}

	client := New(fake)

	if err := client.Probe(context.Background(), domain.Target{ID: vmID}); err == nil {
		t.Fatal("Probe() error = nil, want un error cuando la máquina no está en ejecución")
	}
}

func TestProbeNilWhenMachineRunning(t *testing.T) {
	t.Parallel()

	client := New(runningFake())

	if err := client.Probe(context.Background(), domain.Target{ID: vmID}); err != nil {
		t.Fatalf("Probe() error = %v, want nil", err)
	}
}

func TestStateParsesMachineReadableOutput(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["showvminfo"] = domain.Result{Stdout: `name="debian13"
` + vmRunning + `
memory=2048
`}

	client := New(fake)

	state, err := client.State(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf("State() error = %v, want nil", err)
	}

	if state != domain.StateRunning {
		t.Errorf("State() = %q, want %q", state, domain.StateRunning)
	}
}

func TestStateUnknownWhenValueNotRecognized(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["showvminfo"] = domain.Result{Stdout: `VMState="teletransportado"`}

	client := New(fake)

	state, err := client.State(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf("State() error = %v, want nil", err)
	}

	if state != domain.StateUnknown {
		t.Errorf("State() = %q, want %q", state, domain.StateUnknown)
	}
}

func TestGuestSessionRunBuildsGuestControlCommand(t *testing.T) {
	t.Parallel()

	fake := runningFake()
	fake.results["guestcontrol"] = domain.Result{Stdout: salidaInstalado}

	client := New(fake, WithGuestCredentials(credRoot, credPass))

	session, err := client.Connect(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	result, err := session.Run(context.Background(), &domain.Command{
		Path: binAptGet,
		Args: []string{"install", "-y", "nginx"},
	})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	if result.Stdout != salidaInstalado {
		t.Errorf("Run().Stdout = %q, want %q", result.Stdout, salidaInstalado)
	}

	want := []string{
		"VBoxManage", "guestcontrol", vmID, "run",
		"--username", credRoot, "--password", credPass,
		"--exe", binAptGet,
		"--", binAptGet, "install", "-y", "nginx",
	}

	got := fake.lastCall()
	if len(got) != len(want) {
		t.Fatalf("%s %v, want %v", msgRunArgsPrefix, got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s %v, want %v", msgRunArgsPrefix, got, want)
		}
	}
}

func TestGuestSessionRunWrapsExitError(t *testing.T) {
	t.Parallel()

	fake := runningFake()
	fake.errs["guestcontrol"] = &domain.ExitError{Code: 100, Command: "apt-get"}

	client := New(fake)

	session, err := client.Connect(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	_, err = session.Run(context.Background(), &domain.Command{Path: binAptGet})

	var exitErr *domain.ExitError

	if !errors.As(err, &exitErr) {
		t.Fatalf("Run() error = %v, quiere ser *domain.ExitError", err)
	}

	if exitErr.Code != 100 {
		t.Errorf("ExitError.Code = %d, want 100", exitErr.Code)
	}
}

func TestGuestSessionPutUsesCopyTo(t *testing.T) {
	t.Parallel()

	fake := runningFake()
	fake.results["guestcontrol"] = domain.Result{}

	client := New(fake, WithGuestCredentials(credRoot, credPass))

	session, err := client.Connect(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	err = session.Put(context.Background(), "./svc.service", "/etc/systemd/system/svc.service")
	if err != nil {
		t.Fatalf("Put() error = %v, want nil", err)
	}

	want := []string{
		"VBoxManage", "guestcontrol", vmID, "copyto",
		"--username", credRoot, "--password", credPass,
		"./svc.service", "/etc/systemd/system/svc.service",
	}

	got := fake.lastCall()
	if len(got) != len(want) {
		t.Fatalf("Put() invocó %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Put() invocó %v, want %v", got, want)
		}
	}
}

func TestNewAppliesBinaryOption(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["guestproperty"] = domain.Result{Stdout: "Value: 192.168.1.50\n"}

	customBinary := "C:\\Program Files\\Oracle\\VBoxManage.exe"
	client := New(fake, WithBinary(customBinary))

	if _, err := client.Address(context.Background(), domain.Target{ID: vmID}); err != nil {
		t.Fatalf("Address() error = %v, want nil", err)
	}

	if got := fake.lastCall()[0]; got != customBinary {
		t.Errorf("programa invocado = %q, want la ruta personalizada", got)
	}
}
