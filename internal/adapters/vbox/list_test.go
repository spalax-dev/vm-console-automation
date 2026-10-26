// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package vbox

import (
	"context"
	"errors"
	"strings"
	"testing"

	"consola-vm/internal/core/domain"
)

// Salidas de ejemplo de `VBoxManage list`.
const (
	outputVMs = `"vm-1" {8f3a1c2e-1111-4a4a-9a9a-000000000001}
"ubuntu-lab" {8f3a1c2e-2222-4a4a-9a9a-000000000002}
`
	outputInRunning = `"vm-1" {8f3a1c2e-1111-4a4a-9a9a-000000000001}
`
	msgListWithoutError = "List() error = %v, want nil"
	testMachineName     = "vm-1"
)

// fakeListExecutor responde a `list vms` y `list runningvms` con salidas
// preprogramadas.
type fakeListExecutor struct {
	vms     domain.Result
	vmsErr  error
	inCurso domain.Result
	runErr  error
	calls   []string
}

// Execute devuelve la salida correspondiente a la suborden recibida.
func (f *fakeListExecutor) Execute(_ context.Context, _ string, args ...string) (domain.Result, error) {
	f.calls = append(f.calls, strings.Join(args, " "))

	if len(args) > 1 && args[1] == "runningvms" {
		return f.inCurso, f.runErr
	}

	return f.vms, f.vmsErr
}

// TestListReturnsMachinesWithSuState comprueba la lectura del
// inventario y la marca de las máquinas en ejecución.
func TestListReturnsMachinesWithSuState(t *testing.T) {
	t.Parallel()

	exe := &fakeListExecutor{
		vms:     domain.Result{Stdout: outputVMs},
		inCurso: domain.Result{Stdout: outputInRunning},
	}

	machines, err := New(exe).List(context.Background())
	if err != nil {
		t.Fatalf(msgListWithoutError, err)
	}

	if len(machines) != 2 {
		t.Fatalf("len(maquinas) = %d, want 2", len(machines))
	}

	if machines[0].Name != "vm-1" || machines[0].UUID != "8f3a1c2e-1111-4a4a-9a9a-000000000001" {
		t.Errorf("maquinas[0] = %+v, want vm-1 con su UUID", machines[0])
	}

	if machines[0].State != domain.StateRunning {
		t.Errorf("maquinas[0].State = %q, want %q", machines[0].State, domain.StateRunning)
	}

	if machines[1].State != domain.StatePoweredOff {
		t.Errorf("maquinas[1].State = %q, want %q", machines[1].State, domain.StatePoweredOff)
	}
}

// TestListWithoutMachines devuelve una lista vacía, no un error.
func TestListWithoutMachines(t *testing.T) {
	t.Parallel()

	exe := &fakeListExecutor{}

	machines, err := New(exe).List(context.Background())
	if err != nil {
		t.Fatalf(msgListWithoutError, err)
	}

	if len(machines) != 0 {
		t.Errorf("len(maquinas) = %d, want 0", len(machines))
	}
}

// TestListIgnoresLinesWithoutFormat descarta el ruido de la salida.
func TestListIgnoresLinesWithoutFormat(t *testing.T) {
	t.Parallel()

	exe := &fakeListExecutor{
		vms: domain.Result{Stdout: "aviso del hipervisor\n" + outputVMs},
	}

	machines, err := New(exe).List(context.Background())
	if err != nil {
		t.Fatalf(msgListWithoutError, err)
	}

	if len(machines) != 2 {
		t.Errorf("len(maquinas) = %d, want 2", len(machines))
	}
}

// TestListPropagatesFailureOfHypervisor comprueba que el error de
// VBoxManage no se oculta.
func TestListPropagatesFailureOfHypervisor(t *testing.T) {
	t.Parallel()

	exe := &fakeListExecutor{vmsErr: errors.New("VBoxManage no está disponible")}

	if _, err := New(exe).List(context.Background()); err == nil {
		t.Fatal("List() error = nil, want error")
	}
}

// TestListPropagatesFailureOfMachinesInRunning evita informar un estado
// que no se pudo consultar.
func TestListPropagatesFailureOfMachinesInRunning(t *testing.T) {
	t.Parallel()

	exe := &fakeListExecutor{
		vms:    domain.Result{Stdout: outputVMs},
		runErr: errors.New("fallo al listar las máquinas en ejecución"),
	}

	if _, err := New(exe).List(context.Background()); err == nil {
		t.Fatal("List() error = nil, want error")
	}
}

// TestStartInvocaStartvmWithoutInterfaz comprueba la orden de encendido.
func TestStartInvocaStartvmWithoutInterfaz(t *testing.T) {
	t.Parallel()

	exe := &fakeListExecutor{}

	if err := New(exe).Start(context.Background(), domain.Target{ID: testMachineName}); err != nil {
		t.Fatalf("Start() error = %v, want nil", err)
	}

	if len(exe.calls) != 1 {
		t.Fatalf("llamadas = %d, want 1", len(exe.calls))
	}

	if exe.calls[0] != "startvm vm-1 --type headless" {
		t.Errorf("orden = %q, want %q", exe.calls[0], "startvm vm-1 --type headless")
	}
}

// TestStartPropagatesFailureOfHypervisor conserva la causa del error.
func TestStartPropagatesFailureOfHypervisor(t *testing.T) {
	t.Parallel()

	exe := &fakeListExecutor{vmsErr: errors.New("la máquina no existe")}

	if err := New(exe).Start(context.Background(), domain.Target{ID: testMachineName}); err == nil {
		t.Fatal("Start() error = nil, want error")
	}
}
