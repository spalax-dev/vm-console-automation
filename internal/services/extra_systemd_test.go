// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"testing"

	"consola-vm/internal/core/domain"
)

// TestSystemdListWithMarksOfActive interpreta la línea con «●».
func TestSystemdListWithMarksOfActive(t *testing.T) {
	t.Parallel()

	session := &stubSession{results: []domain.Result{
		{Stdout: "● nano.service loaded active running Editor nano\n"},
		{Stdout: "nano.service enabled\n"},
	}}

	units, err := NewSystemd(&stubOpener{session: session}).List(context.Background())
	if err != nil {
		t.Fatalf("Listar() error = %v, want nil", err)
	}

	if len(units) != 1 || !units[0].Active || !units[0].Enabled {
		t.Errorf("unidades = %+v, want nano.service activa y habilitada", units)
	}
}
