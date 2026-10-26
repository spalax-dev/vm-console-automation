// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"encoding/base64"
	"testing"
)

// conBaseCerrada devuelve un almacén cuya base ya no responde.
func conBaseCerrada(t *testing.T) *Almacen {
	t.Helper()

	almacen, _, _ := nuevaBaseDePrueba(t)

	if err := almacen.db.Close(); err != nil {
		t.Fatalf("cerrar la base: %v", err)
	}

	return almacen
}

// TestConsolaConBaseCerrada propaga los errores de SQLite.
func TestConsolaConBaseCerrada(t *testing.T) {
	t.Parallel()

	almacen := conBaseCerrada(t)
	consola := NuevoConsola(almacen)
	ctx := context.Background()

	if _, err := consola.Configurada(); err == nil {
		t.Error("Configurada() error = nil, want el fallo de SQLite")
	}

	if err := consola.Configurar(ctx, usuarioConsola, passwordConsola); err == nil {
		t.Error("Configurar() error = nil, want el fallo de SQLite")
	}

	if _, err := consola.Autenticar(ctx, usuarioConsola, passwordConsola); err == nil {
		t.Error("Autenticar() error = nil, want el fallo de SQLite")
	}

	if _, err := consola.AbrirSesion(usuarioConsola); err == nil {
		t.Error("AbrirSesion() error = nil, want el fallo de SQLite")
	}

	if _, ok := consola.Validar("jorge|9999999999|firma"); ok {
		t.Error("Validar() = true, want false sin secreto disponible")
	}
}

// TestConsolaSecretoCorrupto detecta un secreto que no es base64.
func TestConsolaSecretoCorrupto(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)

	if _, err := almacen.db.ExecContext(context.Background(),
		`INSERT INTO configuracion (clave, valor) VALUES (?, ?)`,
		claveSecretoConsola, "no-es-base64"); err != nil {
		t.Fatalf("guardar el secreto: %v", err)
	}

	if _, err := almacen.secretoConsola(); err == nil {
		t.Error("secretoConsola() error = nil, want el fallo de decodificación")
	}

	if _, err := NuevoConsola(almacen).AbrirSesion(usuarioConsola); err == nil {
		t.Error("AbrirSesion() error = nil, want el fallo de decodificación")
	}
}

// TestSecretoNoSeGuardaSinBase comprueba el fallo al insertar el secreto.
func TestSecretoNoSeGuardaSinBase(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)

	if err := almacen.db.Close(); err != nil {
		t.Fatalf("cerrar la base: %v", err)
	}

	if _, err := almacen.secretoConsola(); err == nil {
		t.Error("secretoConsola() error = nil, want el fallo al guardar")
	}
}

// TestSecretoSeReutiliza usa el mismo secreto en llamadas sucesivas.
func TestSecretoSeReutiliza(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)

	primero, err := almacen.secretoConsola()
	if err != nil {
		t.Fatalf("secretoConsola() error = %v, want nil", err)
	}

	segundo, err := almacen.secretoConsola()
	if err != nil || base64.StdEncoding.EncodeToString(primero) != base64.StdEncoding.EncodeToString(segundo) {
		t.Errorf("secretoConsola() no reutiliza el secreto: %v", err)
	}
}

// TestConsolaConInsercionBloqueada cubre los fallos al guardar la cuenta
// y el secreto.
func TestConsolaConInsercionBloqueada(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	ctx := context.Background()

	if _, err := almacen.db.ExecContext(ctx,
		`CREATE TRIGGER bloquear_cuenta BEFORE INSERT ON usuarios_consola
		 BEGIN SELECT RAISE(ABORT, 'sin escritura'); END;`); err != nil {
		t.Fatalf("crear el disparador de cuentas: %v", err)
	}

	if err := NuevoConsola(almacen).Configurar(ctx, usuarioConsola, passwordConsola); err == nil {
		t.Error("Configurar() error = nil, want el fallo al guardar la cuenta")
	}

	if _, err := almacen.db.ExecContext(ctx,
		`CREATE TRIGGER bloquear_secreto BEFORE INSERT ON configuracion
		 BEGIN SELECT RAISE(ABORT, 'sin escritura'); END;`); err != nil {
		t.Fatalf("crear el disparador de configuración: %v", err)
	}

	if _, err := almacen.secretoConsola(); err == nil {
		t.Error("secretoConsola() error = nil, want el fallo al guardar el secreto")
	}
}
