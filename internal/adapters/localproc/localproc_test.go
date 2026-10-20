// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package localproc

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

	"consola-vm/internal/core/domain"
)

func TestExecuteCapturesStdout(t *testing.T) {
	t.Parallel()

	executor := New()

	result, err := executor.Execute(context.Background(), "go", "env", "GOOS")
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	if got := strings.TrimSpace(result.Stdout); got != runtime.GOOS {
		t.Errorf("stdout = %q, want %q", got, runtime.GOOS)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestExecuteReturnsExitErrorOnFailure(t *testing.T) {
	t.Parallel()

	executor := New()

	result, err := executor.Execute(context.Background(), "go", "subcomando-inexistente-xyz")
	if err == nil {
		t.Fatal("Execute() error = nil, want un error por código de salida distinto de cero")
	}

	var exitErr *domain.ExitError

	if !errors.As(err, &exitErr) {
		t.Fatalf("Execute() error = %v, quiere ser *domain.ExitError", err)
	}

	if exitErr.Code == 0 {
		t.Error("ExitError.Code = 0, want un código distinto de cero")
	}

	if result.ExitCode == 0 {
		t.Error("result.ExitCode = 0, want un código distinto de cero")
	}

	if result.Stderr == "" {
		t.Error("result.Stderr vacío, want el mensaje de error del proceso")
	}
}

func TestExecuteUnknownProgram(t *testing.T) {
	t.Parallel()

	executor := New()

	_, err := executor.Execute(context.Background(), "programa-que-no-existe-xyz-123")
	if err == nil {
		t.Fatal("Execute() error = nil, want un error de programa inexistente")
	}

	var exitErr *domain.ExitError

	if errors.As(err, &exitErr) {
		t.Error("Execute() devolvió ExitError, want un error de ejecución (el proceso nunca inició)")
	}
}
