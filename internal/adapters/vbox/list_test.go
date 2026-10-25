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
	salidaVMs = `"debian-1" {8f3a1c2e-1111-4a4a-9a9a-000000000001}
"ubuntu-lab" {8f3a1c2e-2222-4a4a-9a9a-000000000002}
`
	salidaEnMarcha = `"debian-1" {8f3a1c2e-1111-4a4a-9a9a-000000000001}
`
	msgListSinError = "List() error = %v, want nil"
	nombreDebian    = "debian-1"
)

// fakeListExecutor responde a `list vms` y `list runningvms` con salidas
// preprogramadas.
type fakeListExecutor struct {
	vms     domain.Result
	vmsErr  error
	enCurso domain.Result
	runErr  error
	calls   []string
}

// Execute devuelve la salida correspondiente a la suborden recibida.
func (f *fakeListExecutor) Execute(_ context.Context, _ string, args ...string) (domain.Result, error) {
	f.calls = append(f.calls, strings.Join(args, " "))

	if len(args) > 1 && args[1] == "runningvms" {
		return f.enCurso, f.runErr
	}

	return f.vms, f.vmsErr
}

// TestListDevuelveLasMaquinasConSuEstado comprueba la lectura del
// inventario y la marca de las máquinas en ejecución.
func TestListDevuelveLasMaquinasConSuEstado(t *testing.T) {
	t.Parallel()

	exe := &fakeListExecutor{
		vms:     domain.Result{Stdout: salidaVMs},
		enCurso: domain.Result{Stdout: salidaEnMarcha},
	}

	maquinas, err := New(exe).List(context.Background())
	if err != nil {
		t.Fatalf(msgListSinError, err)
	}

	if len(maquinas) != 2 {
		t.Fatalf("len(maquinas) = %d, want 2", len(maquinas))
	}

	if maquinas[0].Name != "debian-1" || maquinas[0].UUID != "8f3a1c2e-1111-4a4a-9a9a-000000000001" {
		t.Errorf("maquinas[0] = %+v, want debian-1 con su UUID", maquinas[0])
	}

	if maquinas[0].State != domain.StateRunning {
		t.Errorf("maquinas[0].State = %q, want %q", maquinas[0].State, domain.StateRunning)
	}

	if maquinas[1].State != domain.StatePoweredOff {
		t.Errorf("maquinas[1].State = %q, want %q", maquinas[1].State, domain.StatePoweredOff)
	}
}

// TestListSinMaquinas devuelve una lista vacía, no un error.
func TestListSinMaquinas(t *testing.T) {
	t.Parallel()

	exe := &fakeListExecutor{}

	maquinas, err := New(exe).List(context.Background())
	if err != nil {
		t.Fatalf(msgListSinError, err)
	}

	if len(maquinas) != 0 {
		t.Errorf("len(maquinas) = %d, want 0", len(maquinas))
	}
}

// TestListIgnoraLineasSinFormato descarta el ruido de la salida.
func TestListIgnoraLineasSinFormato(t *testing.T) {
	t.Parallel()

	exe := &fakeListExecutor{
		vms: domain.Result{Stdout: "aviso del hipervisor\n" + salidaVMs},
	}

	maquinas, err := New(exe).List(context.Background())
	if err != nil {
		t.Fatalf(msgListSinError, err)
	}

	if len(maquinas) != 2 {
		t.Errorf("len(maquinas) = %d, want 2", len(maquinas))
	}
}

// TestListPropagaElFalloDelHipervisor comprueba que el error de
// VBoxManage no se oculta.
func TestListPropagaElFalloDelHipervisor(t *testing.T) {
	t.Parallel()

	exe := &fakeListExecutor{vmsErr: errors.New("VBoxManage no está disponible")}

	if _, err := New(exe).List(context.Background()); err == nil {
		t.Fatal("List() error = nil, want error")
	}
}

// TestListPropagaElFalloDeLasMaquinasEnMarcha evita informar un estado
// que no se pudo consultar.
func TestListPropagaElFalloDeLasMaquinasEnMarcha(t *testing.T) {
	t.Parallel()

	exe := &fakeListExecutor{
		vms:    domain.Result{Stdout: salidaVMs},
		runErr: errors.New("fallo al listar las máquinas en ejecución"),
	}

	if _, err := New(exe).List(context.Background()); err == nil {
		t.Fatal("List() error = nil, want error")
	}
}

// TestStartInvocaStartvmSinInterfaz comprueba la orden de encendido.
func TestStartInvocaStartvmSinInterfaz(t *testing.T) {
	t.Parallel()

	exe := &fakeListExecutor{}

	if err := New(exe).Start(context.Background(), domain.Target{ID: nombreDebian}); err != nil {
		t.Fatalf("Start() error = %v, want nil", err)
	}

	if len(exe.calls) != 1 {
		t.Fatalf("llamadas = %d, want 1", len(exe.calls))
	}

	if exe.calls[0] != "startvm debian-1 --type headless" {
		t.Errorf("orden = %q, want %q", exe.calls[0], "startvm debian-1 --type headless")
	}
}

// TestStartPropagaElFalloDelHipervisor conserva la causa del error.
func TestStartPropagaElFalloDelHipervisor(t *testing.T) {
	t.Parallel()

	exe := &fakeListExecutor{vmsErr: errors.New("la máquina no existe")}

	if err := New(exe).Start(context.Background(), domain.Target{ID: nombreDebian}); err == nil {
		t.Fatal("Start() error = nil, want error")
	}
}
