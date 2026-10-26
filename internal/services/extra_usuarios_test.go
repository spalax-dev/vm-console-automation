// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"consola-vm/internal/core/domain"
)

// userJorgeTests es la cuenta de las pruebas.
const userJorgeTests = "jorge"

// TestUsersGetWithoutSession propaga el fallo del abridor.
func TestUsersGetWithoutSession(t *testing.T) {
	t.Parallel()

	users, _, _ := newUsersForTest(t, &stubExecutor{}, nil, errors.New("sin sesión"))

	if _, err := users.Get(context.Background(), userJorgeTests); err == nil {
		t.Error("Obtener() error = nil, want el fallo de la sesión")
	}

	if _, err := users.Delete(context.Background(), userJorgeTests, false); err == nil {
		t.Error("Eliminar() error = nil, want el fallo de la sesión")
	}
}

// TestUsersCreateWithBusquedaRota propaga el fallo de getent.
func TestUsersCreateWithBusquedaRota(t *testing.T) {
	t.Parallel()

	session := &stubSession{err: errors.New("getent caído")}
	users, _, _ := newUsersForTest(t, &stubExecutor{}, session, nil)

	if _, err := users.Create(context.Background(), userJorgeTests, false, false); err == nil {
		t.Error("Crear() error = nil, want el fallo de getent")
	}

	if _, err := users.Delete(context.Background(), userJorgeTests, false); err == nil {
		t.Error("Eliminar() error = nil, want el fallo de getent")
	}
}

// TestUsersCreateWithGeneradorCaido propaga el fallo de ssh-keygen.
func TestUsersCreateWithGeneradorCaido(t *testing.T) {
	t.Parallel()

	session := &stubSession{}
	users, _, _ := newUsersForTest(t, &stubExecutor{err: errors.New("sin ssh-keygen")}, session, nil)

	if _, err := users.Create(context.Background(), userJorgeTests, false, true); err == nil {
		t.Error("Crear() error = nil, want el fallo del generador")
	}
}

// TestUsersDeleteKeysWithFailureOfBorrado comprueba que no poder
// borrar la llave privada no impide eliminar la cuenta.
func TestUsersDeleteKeysWithFailureOfBorrado(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, userJorgeTests+"_ed25519")

	// Un directorio no vacío hace que os.Remove falle, sin depender de
	// los permisos ni de que las pruebas corran como root.
	if err := os.MkdirAll(filepath.Join(path, "contenido"), 0o700); err != nil {
		t.Fatalf("crear el directorio de la llave: %v", err)
	}

	if err := os.WriteFile(filepath.Join(path, "contenido", "archivo"), []byte("x"), 0o600); err != nil {
		t.Fatalf("crear el archivo interno: %v", err)
	}

	session := &stubSession{results: []domain.Result{
		{Stdout: userJorgeTests + ":x:1000:1000::/home/jorge:/bin/bash\n"},
		{},
	}}
	users := NewUsers(UsersConfig{
		Sessions:  &stubOpener{session: session},
		Ejecutor:  &stubExecutor{},
		Directory: directory,
	})

	if _, err := users.Delete(context.Background(), userJorgeTests, false); err != nil {
		t.Errorf("Eliminar() error = %v, want nil", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("la llave que no se pudo borrar desapareció: %v", err)
	}
}

// TestParseUserShort devuelve una cuenta vacía con pocos campos.
func TestParseUserShort(t *testing.T) {
	t.Parallel()

	if user := parseUser("jorge:x:1000"); user.Name != "" {
		t.Errorf("parsearUsuario() = %+v, want vacío", user)
	}

	if user := parseUser("jorge:x:no-numero:1000::/home/jorge:/bin/bash"); user.Name != "" {
		t.Errorf("parsearUsuario() = %+v, want vacío con UID inválido", user)
	}
}
