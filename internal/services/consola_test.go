// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// usuarioConsola y passwordConsola son las credenciales de las pruebas.
const (
	userConsole     = "jorge"
	userAndrey      = "andrey"
	passwordConsole = "una-clave-larga" // #nosec G101 -- contraseña de prueba.
	msgConfigureOK  = "Configurar() error = %v, want nil"
)

// TestConsoleWithoutAccount comprueba el estado inicial.
func TestConsoleWithoutAccount(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	console := NewConsole(store)

	configurada, err := console.IsConfigured()
	if err != nil {
		t.Fatalf("Configurada() error = %v, want nil", err)
	}

	if configurada {
		t.Error("Configurada() = true, want false en una base nueva")
	}

	if _, ok := console.Validate(""); ok {
		t.Error("Validar() = true, want false sin cookie")
	}

	if _, ok := console.Validate("jorge|9999999999|firma"); ok {
		t.Error("Validar() = true, want false con una firma inventada")
	}
}

// TestConsoleConfiguraYAuthenticates crea la cuenta y comprueba el acceso.
func TestConsoleConfiguraYAuthenticates(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	console := NewConsole(store)
	ctx := context.Background()

	if err := console.Configure(ctx, userConsole, passwordConsole); err != nil {
		t.Fatalf(msgConfigureOK, err)
	}

	configurada, err := console.IsConfigured()
	if err != nil || !configurada {
		t.Fatalf("Configurada() = %v, %v, want true", configurada, err)
	}

	testCases := []struct {
		name     string
		user     string
		password string
		expected bool
	}{
		{"correctas", userConsole, passwordConsole, true},
		{"contraseña incorrecta", userConsole, "otra-clave", false},
		{"usuario inexistente", userAndrey, passwordConsole, false},
		{"usuario inválido", "Jorge", passwordConsole, false},
		{"contraseña vacía", userConsole, "", false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ok, err := console.Authenticate(ctx, testCase.user, testCase.password)
			if err != nil {
				t.Fatalf("Autenticar() error = %v, want nil", err)
			}

			if ok != testCase.expected {
				t.Errorf("Autenticar() = %v, want %v", ok, testCase.expected)
			}
		})
	}
}

// TestConsoleDataInvalid rechaza usuarios y contraseñas fuera de las
// reglas antes de tocar la base.
func TestConsoleDataInvalid(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	console := NewConsole(store)

	testCases := []struct {
		name     string
		user     string
		password string
	}{
		{"usuario con mayúscula", "Jorge", passwordConsole},
		{"usuario corto", "ab", passwordConsole},
		{"usuario con espacio", "jor ge", passwordConsole},
		{"contraseña corta", userConsole, "corta"},
		{"contraseña con NUL", userConsole, "una\x00clave"},
		{"contraseña larga", userConsole, strings.Repeat("a", maxPasswordConsole+1)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := console.Configure(context.Background(), testCase.user, testCase.password)
			if !errors.Is(err, ErrConsoleData) {
				t.Errorf("Configurar() error = %v, want ErrDatosConsola", err)
			}
		})
	}
}

// TestConsoleSoloAccount impide crear una segunda cuenta.
func TestConsoleSoloAccount(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	console := NewConsole(store)
	ctx := context.Background()

	if err := console.Configure(ctx, userConsole, passwordConsole); err != nil {
		t.Fatalf(msgConfigureOK, err)
	}

	err := console.Configure(ctx, userAndrey, passwordConsole)
	if !errors.Is(err, ErrConsoleConfigured) {
		t.Errorf("Configurar() error = %v, want ErrConsolaConfigurada", err)
	}
}

// TestConsoleSessionFirmada comprueba la emisión y la validación de la
// cookie, incluida su vigencia.
func TestConsoleSessionFirmada(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)
	console := NewConsole(store)

	if err := console.Configure(context.Background(), userConsole, passwordConsole); err != nil {
		t.Fatalf(msgConfigureOK, err)
	}

	token, err := console.OpenSession(userConsole)
	if err != nil {
		t.Fatalf("AbrirSesion() error = %v, want nil", err)
	}

	user, ok := console.Validate(token)
	if !ok || user != userConsole {
		t.Fatalf("Validar() = %q, %v, want %q, true", user, ok, userConsole)
	}

	parts := strings.Split(token, "|")
	manipulado := parts[0] + "|" + parts[1] + "|" + strings.Repeat("0", len(parts[2]))

	if _, ok := console.Validate(manipulado); ok {
		t.Error("Validar() = true, want false con la firma alterada")
	}

	if _, ok := console.Validate("solo-dos|partes"); ok {
		t.Error("Validar() = true, want false con formato incompleto")
	}

	// Doce horas después la sesión vence.
	console.now = func() time.Time { return time.Now().Add(SessionLifetime + time.Minute) }

	if _, ok := console.Validate(token); ok {
		t.Error("Validar() = true, want false con la sesión vencida")
	}
}

// TestConsoleSecretPersistente valida una cookie emitida por otra
// instancia de la consola sobre la misma base.
func TestConsoleSecretPersistente(t *testing.T) {
	t.Parallel()

	store, pathBD, pathKey := newTestDB(t)
	first := NewConsole(store)

	if err := first.Configure(context.Background(), userConsole, passwordConsole); err != nil {
		t.Fatalf(msgConfigureOK, err)
	}

	token, err := first.OpenSession(userConsole)
	if err != nil {
		t.Fatalf("AbrirSesion() error = %v, want nil", err)
	}

	other, err := NewStore(pathBD, pathKey)
	if err != nil {
		t.Fatalf("NuevoAlmacen() error = %v, want nil", err)
	}

	user, ok := NewConsole(other).Validate(token)
	if !ok || user != userConsole {
		t.Errorf("Validar() tras reabrir la base = %q, %v, want %q, true", user, ok, userConsole)
	}
}
