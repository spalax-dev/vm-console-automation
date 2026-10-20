// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package localproc implementa el puerto ports.Executor lanzando
// procesos en el sistema operativo donde corre la consola. Es el único
// punto del código que ejecuta programas externos.
package localproc

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"consola-vm/internal/core/domain"
)

// Executor ejecuta programas del sistema operativo anfitrión.
type Executor struct{}

// New crea un ejecutor de programas locales.
func New() *Executor {
	return &Executor{}
}

// Execute implements ports.Executor: lanza program con args, captura
// stdout y stderr y devuelve el código de salida. Si el proceso termina
// con un código distinto de cero, devuelve un *domain.ExitError; si no
// puede iniciarse, devuelve un error envolviendo el fallo original.
func (e *Executor) Execute(ctx context.Context, program string, args ...string) (domain.Result, error) {
	// #nosec G204 -- el propósito del Executor es lanzar programas
	// externos; los argumentos provienen de los puertos de la consola.
	cmd := exec.CommandContext(ctx, program, args...)

	var stdout, stderr strings.Builder

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := domain.Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err == nil {
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()

		return result, &domain.ExitError{
			Code:    result.ExitCode,
			Command: strings.Join(append([]string{program}, args...), " "),
		}
	}

	result.ExitCode = -1

	return result, fmt.Errorf("no se pudo ejecutar %q: %w", program, err)
}
