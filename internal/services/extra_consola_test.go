// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"encoding/base64"
	"testing"
)

// withClosedDB devuelve un almacén cuya base ya no responde.
func withClosedDB(t *testing.T) *Store {
	t.Helper()

	store, _, _ := newTestDB(t)

	if err := store.db.Close(); err != nil {
		t.Fatalf("cerrar la base: %v", err)
	}

	return store
}

// TestConsoleWithBaseClosed propaga los errores de SQLite.
func TestConsoleWithBaseClosed(t *testing.T) {
	t.Parallel()

	store := withClosedDB(t)
	console := NewConsole(store)
	ctx := context.Background()

	if _, err := console.IsConfigured(); err == nil {
		t.Error("Configurada() error = nil, want el fallo de SQLite")
	}

	if err := console.Configure(ctx, userConsole, passwordConsole); err == nil {
		t.Error("Configurar() error = nil, want el fallo de SQLite")
	}

	if _, err := console.Authenticate(ctx, userConsole, passwordConsole); err == nil {
		t.Error("Autenticar() error = nil, want el fallo de SQLite")
	}

	if _, err := console.OpenSession(userConsole); err == nil {
		t.Error("AbrirSesion() error = nil, want el fallo de SQLite")
	}

	if _, ok := console.Validate("jorge|9999999999|firma"); ok {
		t.Error("Validar() = true, want false sin secreto disponible")
	}
}

// TestConsoleSecretCorrupto detecta un secreto que no es base64.
func TestConsoleSecretCorrupto(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)

	if _, err := store.db.ExecContext(context.Background(),
		`INSERT INTO configuracion (clave, valor) VALUES (?, ?)`,
		consoleSecretKey, "no-es-base64"); err != nil {
		t.Fatalf("guardar el secreto: %v", err)
	}

	if _, err := store.consoleSecret(); err == nil {
		t.Error("secretoConsola() error = nil, want el fallo de decodificación")
	}

	if _, err := NewConsole(store).OpenSession(userConsole); err == nil {
		t.Error("AbrirSesion() error = nil, want el fallo de decodificación")
	}
}

// TestSecretNotSeSavesWithoutBase comprueba el fallo al insertar el secreto.
func TestSecretNotSeSavesWithoutBase(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)

	if err := store.db.Close(); err != nil {
		t.Fatalf("cerrar la base: %v", err)
	}

	if _, err := store.consoleSecret(); err == nil {
		t.Error("secretoConsola() error = nil, want el fallo al guardar")
	}
}

// TestSecretSeReutiliza usa el mismo secreto en llamadas sucesivas.
func TestSecretSeReutiliza(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)

	first, err := store.consoleSecret()
	if err != nil {
		t.Fatalf("secretoConsola() error = %v, want nil", err)
	}

	second, err := store.consoleSecret()
	if err != nil || base64.StdEncoding.EncodeToString(first) != base64.StdEncoding.EncodeToString(second) {
		t.Errorf("secretoConsola() no reutiliza el secreto: %v", err)
	}
}

// TestConsoleWithInsercionBloqueada cubre los fallos al guardar la cuenta
// y el secreto.
func TestConsoleWithInsercionBloqueada(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	ctx := context.Background()

	if _, err := store.db.ExecContext(ctx,
		`CREATE TRIGGER bloquear_cuenta BEFORE INSERT ON usuarios_consola
		 BEGIN SELECT RAISE(ABORT, 'sin escritura'); END;`); err != nil {
		t.Fatalf("crear el disparador de cuentas: %v", err)
	}

	if err := NewConsole(store).Configure(ctx, userConsole, passwordConsole); err == nil {
		t.Error("Configurar() error = nil, want el fallo al guardar la cuenta")
	}

	if _, err := store.db.ExecContext(ctx,
		`CREATE TRIGGER bloquear_secreto BEFORE INSERT ON configuracion
		 BEGIN SELECT RAISE(ABORT, 'sin escritura'); END;`); err != nil {
		t.Fatalf("crear el disparador de configuración: %v", err)
	}

	if _, err := store.consoleSecret(); err == nil {
		t.Error("secretoConsola() error = nil, want el fallo al guardar el secreto")
	}
}
