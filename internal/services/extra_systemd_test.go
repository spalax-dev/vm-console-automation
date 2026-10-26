// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"testing"

	"consola-vm/internal/core/domain"
)

// TestSystemdListarConMarcaDeActivo interpreta la línea con «●».
func TestSystemdListarConMarcaDeActivo(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{results: []domain.Result{
		{Stdout: "● nano.service loaded active running Editor nano\n"},
		{Stdout: "nano.service enabled\n"},
	}}

	unidades, err := NuevoSystemd(&stubAbridor{sesion: sesion}).Listar(context.Background())
	if err != nil {
		t.Fatalf("Listar() error = %v, want nil", err)
	}

	if len(unidades) != 1 || !unidades[0].Activo || !unidades[0].Habilitado {
		t.Errorf("unidades = %+v, want nano.service activa y habilitada", unidades)
	}
}
