// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package vbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"consola-vm/internal/core/domain"
)

const (
	vmID             = "vm1"
	vmRunning        = `VMState="running"`
	outputInstalado  = "instalado\n"
	credRoot         = "root"
	credPass         = "secreto"
	binAptGet        = "/usr/bin/apt-get"
	msgConnectFail   = "Connect() error = %v, want nil"
	msgRunArgsPrefix = "Run() invocó"
	msgArgs          = "%s %v, want %v"
	subInstall       = "install"
	flagPassword     = "--password"
	flagUsername     = "--username"
	subGuestControl  = "guestcontrol"
	msgRunOK         = "Run() error = %v, want nil"
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

// TestAddressParsesGuestPropertyValue lee la dirección publicada por
// Guest Additions en la primera interfaz.
// guestProperty arma una línea de «guestproperty enumerate» con el mismo
// formato que imprime VBoxManage.
func guestProperty(key, value string) string {
	return fmt.Sprintf("%s = '%s' @ 2026-10-28T19:48:27.000Z\n", key, value)
}

func TestAddressParsesGuestPropertyValue(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["guestproperty"] = domain.Result{Stdout: guestProperty(`/VirtualBox/GuestInfo/Net/0/V4/IP`, "10.0.2.15")}

	client := New(fake)

	addr, err := client.Address(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf("Address() error = %v, want nil", err)
	}

	if addr.String() != "10.0.2.15" {
		t.Errorf("Address() = %q, want %q", addr.String(), "10.0.2.15")
	}
}

// TestAddressUsesLaterInterfaceWhenFirstHasNoIP cubre el caso real de una
// máquina cuyo adaptador 0 no publica dirección pero el 1 sí.
func TestAddressUsesLaterInterfaceWhenFirstHasNoIP(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["guestproperty"] = domain.Result{Stdout: guestProperty(`/VirtualBox/GuestInfo/Net/0/V4/IP`, "") +
		guestProperty(`/VirtualBox/GuestInfo/Net/1/V4/IP`, "192.168.1.29")}

	client := New(fake)

	addr, err := client.Address(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf("Address() error = %v, want nil", err)
	}

	if addr.String() != "192.168.1.29" {
		t.Errorf("Address() = %q, want la dirección de la segunda interfaz", addr.String())
	}
}

// TestAddressSkipsUnusableAddresses ignora loopback, enlace local y
// valores que no son direcciones.
func TestAddressSkipsUnusableAddresses(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["guestproperty"] = domain.Result{Stdout: guestProperty(`/VirtualBox/GuestInfo/Net/0/V4/IP`, "127.0.0.1") +
		guestProperty(`/VirtualBox/GuestInfo/Net/1/V4/IP`, "169.254.10.20") +
		guestProperty(`/VirtualBox/GuestInfo/Net/2/V4/IP`, "no-es-una-ip") +
		guestProperty(`/VirtualBox/GuestInfo/Net/3/V4/IP`, "192.168.1.29")}

	client := New(fake)

	addr, err := client.Address(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf("Address() error = %v, want nil", err)
	}

	if addr.String() != "192.168.1.29" {
		t.Errorf("Address() = %q, want la única dirección utilizable", addr.String())
	}
}

// TestAddressErrorWhenPropertyMissing comprueba el fallo de enumeración.
func TestAddressErrorWhenPropertyMissing(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["guestproperty"] = domain.Result{Stdout: ""}
	fake.errs["guestproperty"] = errors.New("VERR_NOT_FOUND")

	client := New(fake)

	if _, err := client.Address(context.Background(), domain.Target{ID: vmID}); err == nil {
		t.Fatal("Address() error = nil, want un error cuando la propiedad no existe")
	}
}

// TestAddressErrorWhenCableDisconnected explica que la máquina no tiene
// red conectada, en lugar de culpar a Guest Additions.
func TestAddressErrorWhenCableDisconnected(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["guestproperty"] = domain.Result{Stdout: guestProperty(`/VirtualBox/GuestInfo/Net/Count`, "0")}
	fake.results["showvminfo"] = domain.Result{Stdout: "nic1=\"bridged\"\ncableconnected1=\"off\"\n"}

	client := New(fake)

	_, err := client.Address(context.Background(), domain.Target{ID: vmID})
	if err == nil {
		t.Fatal("Address() error = nil, want el aviso de interfaz desconectada")
	}

	if !strings.Contains(err.Error(), "desconectada") {
		t.Errorf("Address() error = %q, want el aviso de cable desconectado", err)
	}
}

// TestAddressErrorWhenNoIPv4Published cubre una máquina conectada pero
// sin dirección publicada.
func TestAddressErrorWhenNoIPv4Published(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["guestproperty"] = domain.Result{Stdout: guestProperty(`/VirtualBox/GuestInfo/Net/Count`, "0")}
	fake.results["showvminfo"] = domain.Result{Stdout: "nic1=\"bridged\"\ncableconnected1=\"on\"\n"}

	client := New(fake)

	_, err := client.Address(context.Background(), domain.Target{ID: vmID})
	if err == nil {
		t.Fatal("Address() error = nil, want un error cuando no hay dirección")
	}

	if strings.Contains(err.Error(), "desconectada") {
		t.Errorf("Address() error = %q, no debe culpar al cable si está conectado", err)
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
	fake.results["showvminfo"] = domain.Result{Stdout: `name="vm1"
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
	fake.results[subGuestControl] = domain.Result{Stdout: outputInstalado}

	client := New(fake, WithGuestCredentials(credRoot, credPass))

	session, err := client.Connect(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	result, err := session.Run(context.Background(), &domain.Command{
		Path: binAptGet,
		Args: []string{subInstall, "-y", "nginx"},
	})
	if err != nil {
		t.Fatalf(msgRunOK, err)
	}

	if result.Stdout != outputInstalado {
		t.Errorf("Run().Stdout = %q, want %q", result.Stdout, outputInstalado)
	}

	want := []string{
		defaultBinary, subGuestControl, vmID, "run",
		flagUsername, credRoot, flagPassword, credPass,
		"--exe", binAptGet,
		"--", subInstall, "-y", "nginx",
	}

	got := fake.lastCall()
	if len(got) != len(want) {
		t.Fatalf(msgArgs, msgRunArgsPrefix, got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf(msgArgs, msgRunArgsPrefix, got, want)
		}
	}
}

func TestGuestSessionRunWrapsExitError(t *testing.T) {
	t.Parallel()

	fake := runningFake()
	fake.errs[subGuestControl] = &domain.ExitError{Code: 100, Command: "apt-get"}

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
	fake.results[subGuestControl] = domain.Result{}

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
		defaultBinary, subGuestControl, vmID, "copyto",
		flagUsername, credRoot, flagPassword, credPass,
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
	fake.results["guestproperty"] = domain.Result{Stdout: guestProperty(`/VirtualBox/GuestInfo/Net/0/V4/IP`, "192.168.1.50")}

	customBinary := "C:\\Program Files\\Oracle\\VBoxManage.exe"
	client := New(fake, WithBinary(customBinary))

	if _, err := client.Address(context.Background(), domain.Target{ID: vmID}); err != nil {
		t.Fatalf("Address() error = %v, want nil", err)
	}

	if got := fake.lastCall()[0]; got != customBinary {
		t.Errorf("programa invocado = %q, want la ruta personalizada", got)
	}
}

// TestGuestSessionRunPassesEnvironment comprueba que las variables de
// entorno de la orden se envían con --putenv.
func TestGuestSessionRunPassesEnvironment(t *testing.T) {
	t.Parallel()

	fake := runningFake()
	client := New(fake, WithGuestCredentials(credRoot, credPass))

	session, err := client.Connect(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	_, err = session.Run(context.Background(), &domain.Command{
		Path: binAptGet,
		Args: []string{subInstall, "-y", "nano"},
		Env:  []string{"DEBIAN_FRONTEND=noninteractive"},
	})
	if err != nil {
		t.Fatalf(msgRunOK, err)
	}

	want := []string{
		defaultBinary, subGuestControl, vmID, "run",
		flagUsername, credRoot, flagPassword, credPass,
		"--putenv", "DEBIAN_FRONTEND=noninteractive",
		"--exe", binAptGet,
		"--", subInstall, "-y", "nano",
	}

	got := fake.lastCall()
	if len(got) != len(want) {
		t.Fatalf(msgArgs, msgRunArgsPrefix, got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf(msgArgs, msgRunArgsPrefix, got, want)
		}
	}
}

// TestNetworkParsesNICMode comprueba la lectura del modo de red.
func TestNetworkParsesNICMode(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results["showvminfo"] = domain.Result{Stdout: `nic1="bridged"
` + vmRunning + `
`}

	mode, err := New(fake).Network(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf("Network() error = %v, want nil", err)
	}

	if mode != domain.NetworkBridged {
		t.Errorf("Network() = %q, want %q", mode, domain.NetworkBridged)
	}
}

// TestMapNetwork traduce los modos de VirtualBox.
func TestMapNetwork(t *testing.T) {
	t.Parallel()

	testCases := map[string]domain.NetworkMode{
		"bridged":    domain.NetworkBridged,
		"nat":        domain.NetworkNAT,
		"natnetwork": domain.NetworkNAT,
		"hostonly":   domain.NetworkHostOnly,
		"intnet":     domain.NetworkInternal,
		"none":       domain.NetworkNone,
		"otro":       domain.NetworkUnknown,
	}

	for entry, expected := range testCases {
		if obtained := mapNetwork(entry); obtained != expected {
			t.Errorf("mapNetwork(%q) = %q, want %q", entry, obtained, expected)
		}
	}
}

// fakeCredentials devuelve un usuario y una contraseña fijos.
type fakeCredentials struct {
	user     string
	password string
	ok       bool
}

// Guest implementa la resolución de credenciales.
func (f fakeCredentials) Guest(string) (user, password string, ok bool) {
	return f.user, f.password, f.ok
}

// TestGuestSessionUsesStoredCredentials comprueba que la sesión usa las
// credenciales por máquina cuando el almacén las conoce.
func TestGuestSessionUsesStoredCredentials(t *testing.T) {
	t.Parallel()

	fake := runningFake()
	fake.results["guestcontrol"] = domain.Result{}

	client := New(fake,
		WithGuestCredentials("root", "global"),
		WithCredentials(fakeCredentials{user: "admin", password: "clave", ok: true}),
	)

	session, err := client.Connect(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	if _, err := session.Run(context.Background(), &domain.Command{Path: binAptGet}); err != nil {
		t.Fatalf(msgRunOK, err)
	}

	got := fake.lastCall()
	if !contieneValue(got, "admin") || !contieneValue(got, "clave") {
		t.Errorf("argumentos = %v, want las credenciales del almacén", got)
	}
}

// contieneValue informa si la lista incluye el valor.
func contieneValue(values []string, buscado string) bool {
	for _, value := range values {
		if value == buscado {
			return true
		}
	}

	return false
}

// TestGuestSessionSanitizesError comprueba que la contraseña no aparece
// en el error devuelto.
func TestGuestSessionSanitizesError(t *testing.T) {
	t.Parallel()

	fake := runningFake()
	fake.errs["guestcontrol"] = &domain.ExitError{
		Code:    1,
		Command: "VBoxManage guestcontrol vm1 run --password secreto",
	}

	client := New(fake, WithGuestCredentials(credRoot, "secreto"))

	session, err := client.Connect(context.Background(), domain.Target{ID: vmID})
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	_, err = session.Run(context.Background(), &domain.Command{Path: binAptGet})
	if err == nil {
		t.Fatal("Run() error = nil, want el error de Guest Additions")
	}

	if strings.Contains(err.Error(), "secreto") {
		t.Errorf("Run() error = %q, expone la contraseña", err.Error())
	}
}
