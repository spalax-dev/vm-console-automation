// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package ssh

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"consola-vm/internal/core/domain"
)

const (
	salidaInstalado  = "instalado\n"
	binAptGet        = "/usr/bin/apt-get"
	remoteService    = "/etc/systemd/system/svc.service"
	msgConnectFail   = "Connect() error = %v, want nil"
	msgRunArgsPrefix = "Run() invocó"
)

// fakeExecutor registra las órdenes ejecutadas y devuelve respuestas
// preprogramadas, para probar el adaptador sin un servidor SSH real.
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

	return f.results[program], f.errs[program]
}

func (f *fakeExecutor) lastCall() []string {
	if len(f.calls) == 0 {
		return nil
	}

	return f.calls[len(f.calls)-1]
}

func writeTempKey(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(path, []byte("llave-falsa"), 0o600); err != nil {
		t.Fatalf("no se pudo crear la llave temporal: %v", err)
	}

	return path
}

func runningTarget() domain.Target {
	return domain.Target{ID: "debian13", Address: netip.MustParseAddr("10.0.2.15")}
}

func TestNameIsSSH(t *testing.T) {
	t.Parallel()

	connector := New(newFakeExecutor(), WithKey(writeTempKey(t)))

	if got := connector.Name(); got != binarySSH {
		t.Errorf("Name() = %q, want %q", got, binarySSH)
	}
}

func TestProbeErrorWhenAddressMissing(t *testing.T) {
	t.Parallel()

	connector := New(newFakeExecutor(), WithKey(writeTempKey(t)))
	target := domain.Target{ID: "debian13"}

	if err := connector.Probe(context.Background(), target); err == nil {
		t.Fatal("Probe() error = nil, want un error cuando no hay dirección conocida")
	}
}

func TestProbeErrorWhenKeyMissing(t *testing.T) {
	t.Parallel()

	connector := New(newFakeExecutor(), WithKey(filepath.Join(t.TempDir(), "no-existe")))

	if err := connector.Probe(context.Background(), runningTarget()); err == nil {
		t.Fatal("Probe() error = nil, want un error cuando la llave no existe")
	}
}

func TestRunBuildsSSHCommandWithQuotedRemoteCommand(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.results[binarySSH] = domain.Result{Stdout: salidaInstalado}

	key := writeTempKey(t)
	connector := New(fake, WithKey(key), WithUser("root"), WithPort(2222))

	session, err := connector.Connect(context.Background(), runningTarget())
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	result, err := session.Run(context.Background(), &domain.Command{
		Path: binAptGet,
		Args: []string{"install", "-y", "nginx"},
		Dir:  "/opt/trabajo",
	})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	if result.Stdout != salidaInstalado {
		t.Errorf("Run().Stdout = %q, want %q", result.Stdout, salidaInstalado)
	}

	want := []string{
		binarySSH, "-i", key, "-p", "2222",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"root@10.0.2.15",
		`cd '/opt/trabajo' && ` + (&domain.Command{
			Path: binAptGet,
			Args: []string{"install", "-y", "nginx"},
		}).ShellLine(),
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

func TestRunReturnsExitErrorFromExecutor(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()
	fake.errs[binarySSH] = &domain.ExitError{Code: 100, Command: "apt-get"}

	connector := New(fake, WithKey(writeTempKey(t)))

	session, err := connector.Connect(context.Background(), runningTarget())
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

func TestPutUsesSCPAndSetsPermissions(t *testing.T) {
	t.Parallel()

	fake := newFakeExecutor()

	key := writeTempKey(t)
	connector := New(fake, WithKey(key), WithUser("root"))

	session, err := connector.Connect(context.Background(), runningTarget())
	if err != nil {
		t.Fatalf(msgConnectFail, err)
	}

	err = session.Put(context.Background(), "./svc.service", remoteService)
	if err != nil {
		t.Fatalf("Put() error = %v, want nil", err)
	}

	wantSCP := []string{
		binarySCP, "-i", key, "-P", "22",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"./svc.service", "root@10.0.2.15:" + remoteService,
	}

	got := fake.calls[0]
	if len(got) != len(wantSCP) {
		t.Fatalf("Put() invocó %v, want %v", got, wantSCP)
	}

	for i := range wantSCP {
		if got[i] != wantSCP[i] {
			t.Fatalf("Put() invocó %v, want %v", got, wantSCP)
		}
	}

	wantChmod := `'/bin/chmod' '755' '` + remoteService + `'`
	last := fake.lastCall()

	if got := last[len(last)-1]; got != wantChmod {
		t.Errorf("Put() cerró con %q, want %q", got, wantChmod)
	}
}
