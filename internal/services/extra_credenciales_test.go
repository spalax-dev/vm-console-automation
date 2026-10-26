// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

// fileWithText crea un archivo cualquiera y devuelve su ruta.
func fileWithText(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "archivo")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("crear el archivo: %v", err)
	}

	return path
}

// TestNewStoreWithDirectoryOfDataImpossible falla cuando el
// directorio de la base es en realidad un archivo.
func TestNewStoreWithDirectoryOfDataImpossible(t *testing.T) {
	t.Parallel()

	pathBD := filepath.Join(fileWithText(t), "vmcons.db")

	if _, err := NewStore(pathBD, filepath.Join(t.TempDir(), "clave")); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo del directorio")
	}
}

// TestNewStoreWithKeyUnreadable falla si la clave es un directorio.
func TestNewStoreWithKeyUnreadable(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	if _, err := NewStore(filepath.Join(directory, fileBDTests), directory); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo de lectura de la clave")
	}
}

// TestNewStoreNotCanCreateDirectoryOfKey falla cuando el
// directorio padre de la clave es un archivo. No depende de permisos ni
// del usuario que ejecuta las pruebas.
func TestNewStoreNotCanCreateDirectoryOfKey(t *testing.T) {
	t.Parallel()

	file := fileWithText(t)
	pathKey := filepath.Join(file, "sub", "clave")

	if _, err := NewStore(filepath.Join(t.TempDir(), fileBDTests), pathKey); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo al crear el directorio de la clave")
	}
}

// TestNewStoreWithBaseThatIsDirectory falla al crear el esquema si la
// ruta de la base apunta a un directorio.
func TestNewStoreWithBaseThatIsDirectory(t *testing.T) {
	t.Parallel()

	if _, err := NewStore(t.TempDir(), filepath.Join(t.TempDir(), "clave")); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo al crear el esquema")
	}
}

// TestStoreSaveUpdatesCredentials comprueba que guardar dos veces
// la misma máquina actualiza la fila en lugar de duplicarla.
func TestStoreSaveUpdatesCredentials(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)

	first := Credentials{User: "jorge", Password: "uno", Port: 22, Address: "10.0.0.1", IsRegistered: true}
	if err := store.Save(testMachineName, first); err != nil {
		t.Fatalf("Guardar() error = %v, want nil", err)
	}

	second := Credentials{User: "andrey", Password: "dos", Port: 2222, IsRegistered: false}
	if err := store.Save(testMachineName, second); err != nil {
		t.Fatalf("Guardar() error = %v, want nil", err)
	}

	guardadas, ok := store.Get(testMachineName)
	if !ok {
		t.Fatal("Obtener() no encontró las credenciales actualizadas")
	}

	if guardadas != second {
		t.Errorf("Obtener() = %+v, want %+v", guardadas, second)
	}

	var total int
	if err := store.db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM credenciales WHERE vm = ?`, testMachineName).Scan(&total); err != nil {
		t.Fatalf("contar las credenciales: %v", err)
	}

	if total != 1 {
		t.Errorf("filas = %d, want 1", total)
	}
}

// TestGetWithDecryptionFailed devuelve la cuenta sin contraseña.
func TestGetWithDecryptionFailed(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)

	if err := store.Save(testMachineName, Credentials{User: "jorge", Password: "secreto"}); err != nil {
		t.Fatalf("Guardar() error = %v, want nil", err)
	}

	if _, err := store.db.ExecContext(context.Background(),
		`UPDATE credenciales SET password = 'invalido' WHERE vm = ?`, testMachineName); err != nil {
		t.Fatalf("corromper la contraseña: %v", err)
	}

	credentials, ok := store.Get(testMachineName)
	if !ok || credentials.Password != "" {
		t.Errorf("Obtener() = %+v, %v, want la cuenta sin contraseña", credentials, ok)
	}
}

// TestCredentialsWithBaseClosed propaga los errores de SQLite.
func TestCredentialsWithBaseClosed(t *testing.T) {
	t.Parallel()

	store := withClosedDB(t)

	if _, ok := store.Get(testMachineName); ok {
		t.Error("Obtener() = true con la base cerrada")
	}

	if _, _, ok := store.Guest(testMachineName); ok {
		t.Error("Guest() = true con la base cerrada")
	}

	if store.IsRegistered(testMachineName) {
		t.Error("Registrada() = true con la base cerrada")
	}

	if err := store.Save(testMachineName, Credentials{}); err == nil {
		t.Error("Guardar() error = nil, want el fallo de SQLite")
	}

	if err := store.Delete(testMachineName); err == nil {
		t.Error("Borrar() error = nil, want el fallo de SQLite")
	}

	if active := store.Active(); active != "" {
		t.Errorf("Activa() = %q, want vacío con la base cerrada", active)
	}

	if err := store.SaveActive(testMachineName); err == nil {
		t.Error("GuardarActiva() error = nil, want el fallo de SQLite")
	}

	if err := ensureOperationDescriptions(store.db); err == nil {
		t.Error("asegurarDescripcionOperaciones() error = nil, want el fallo de SQLite")
	}
}

// TestDecryptTextosInvalid cubre los datos corruptos.
func TestDecryptTextosInvalid(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)

	testCases := []struct {
		name string
		text string
	}{
		{"no es base64", "no-es-base64"},
		{"cifrado incompleto", base64.StdEncoding.EncodeToString([]byte("corto"))},
		{"sin autenticación", base64.StdEncoding.EncodeToString(make([]byte, 20))},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if _, err := store.decrypt(testCase.text); err == nil {
				t.Errorf("descifrar(%q) error = nil, want un error", testCase.text)
			}
		})
	}
}

// TestExistsColumnaWithBaseClosed propaga el fallo de consulta.
func TestExistsColumnaWithBaseClosed(t *testing.T) {
	t.Parallel()

	store := withClosedDB(t)

	if _, err := columnExists(store.db, "operaciones", "descripcion"); err == nil {
		t.Error("existeColumna() error = nil, want el fallo de SQLite")
	}
}
