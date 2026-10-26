// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// fileBDTests es el nombre de la base temporal.
const fileBDTests = "vmcons.db"

// newTestDB crea la base y la clave en un directorio temporal.
func newTestDB(t *testing.T) (store *Store, pathBD, pathKey string) {
	t.Helper()

	directory := t.TempDir()
	pathBD = filepath.Join(directory, fileBDTests)
	pathKey = filepath.Join(directory, fileKey)

	store, err := NewStore(pathBD, pathKey)
	if err != nil {
		t.Fatalf(msgStoreOK, err)
	}

	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("cerrar almacen de prueba: %v", err)
		}
	})

	return store, pathBD, pathKey
}

// TestStoreSavesYLoads comprueba la persistencia en SQLite.
func TestStoreSavesYLoads(t *testing.T) {
	t.Parallel()

	store, pathBD, _ := newTestDB(t)

	if err := store.Save(testMachineName, Credentials{
		User: userRoot, Password: passwordTests, Port: 22, IsRegistered: true,
	}); err != nil {
		t.Fatalf(msgSaveOK, err)
	}

	other, err := NewStore(pathBD, filepath.Join(filepath.Dir(pathBD), fileKey))
	if err != nil {
		t.Fatalf(msgStoreOK, err)
	}

	t.Cleanup(func() {
		if err := other.Close(); err != nil {
			t.Errorf("cerrar segundo almacen: %v", err)
		}
	})

	credentials, ok := other.Get(testMachineName)
	if !ok {
		t.Fatal("Obtener() no encontró las credenciales")
	}

	if credentials.User != userRoot || credentials.Password != passwordTests || !credentials.IsRegistered {
		t.Errorf("credenciales = %+v, want root/secreto/registrada", credentials)
	}

	user, password, ok := other.Guest(testMachineName)
	if !ok || user != userRoot || password != passwordTests {
		t.Errorf("Guest() = %q, %q, %v, want root, secreto, true", user, password, ok)
	}
}

// TestStoreCifraContrasena comprueba que la base no guarda el texto
// plano y que la clave es privada.
func TestStoreCifraContrasena(t *testing.T) {
	t.Parallel()

	store, pathBD, pathKey := newTestDB(t)

	if err := store.Save(testMachineName, Credentials{User: userRoot, Password: passwordTests}); err != nil {
		t.Fatalf(msgSaveOK, err)
	}

	db, err := sql.Open("sqlite", pathBD)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("cerrar la base: %v", err)
		}
	}()

	var ciphertext string
	if err := db.QueryRowContext(context.Background(),
		`SELECT password FROM credenciales WHERE vm = ?`, testMachineName,
	).Scan(&ciphertext); err != nil {
		t.Fatalf("leer la contraseña: %v", err)
	}

	if ciphertext == passwordTests {
		t.Error("la contraseña se guardó en texto plano")
	}

	info, err := os.Stat(pathKey)
	if err != nil {
		t.Fatalf("Stat() de la clave: %v", err)
	}

	if info.Mode().Perm() != 0o600 && runtime.GOOS != "windows" {
		t.Errorf("permisos de la clave = %v, want 0600", info.Mode().Perm())
	}
}

// TestStoreRegisteredYDelete comprueba la marca y el borrado.
func TestStoreRegisteredYDelete(t *testing.T) {
	t.Parallel()

	store, _, _ := newTestDB(t)

	if err := store.Save(vmTests, Credentials{User: "admin", Password: "clave", IsRegistered: true}); err != nil {
		t.Fatalf(msgSaveOK, err)
	}

	if !store.IsRegistered(vmTests) {
		t.Error("Registrada() = false, want true")
	}

	if err := store.Delete(vmTests); err != nil {
		t.Fatalf("Borrar() error = %v", err)
	}

	if store.IsRegistered(vmTests) {
		t.Error("Registrada() = true tras borrar")
	}

	if _, ok := store.Get(vmTests); ok {
		t.Error("Obtener() encontró credenciales borradas")
	}
}

// TestStoreKeyCorrupta rechaza una clave con longitud inválida.
func TestStoreKeyCorrupta(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	pathKey := filepath.Join(directory, fileKey)

	if err := os.WriteFile(pathKey, []byte("corta"), 0o600); err != nil {
		t.Fatalf("preparar la clave: %v", err)
	}

	_, err := NewStore(filepath.Join(directory, fileBDTests), pathKey)
	if err == nil {
		t.Fatal("NuevoAlmacen() error = nil, want el error de clave inválida")
	}
}

// TestStoreMigraHistory comprueba que una base anterior recibe la
// columna descripcion y conserva sus operaciones.
func TestStoreMigraHistory(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	pathBD := filepath.Join(directory, fileBDTests)
	pathKey := filepath.Join(directory, fileKey)

	db, err := sql.Open("sqlite", pathBD)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	if _, err := db.ExecContext(t.Context(), `CREATE TABLE operaciones (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		fecha_hora TEXT NOT NULL,
		vm TEXT NOT NULL,
		modulo TEXT NOT NULL,
		operacion TEXT NOT NULL,
		resultado TEXT NOT NULL,
		detalle TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatalf("crear la tabla anterior: %v", err)
	}

	if _, err := db.ExecContext(t.Context(), `INSERT INTO operaciones
		(fecha_hora, vm, modulo, operacion, resultado, detalle)
		VALUES ('2026-10-25T10:00:00Z', 'vm-1', 'paquetes',
		'POST /api/packages', 'ok', 'HTTP 201')`); err != nil {
		t.Fatalf("insertar la operación anterior: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("cerrar la base anterior: %v", err)
	}

	store, err := NewStore(pathBD, pathKey)
	if err != nil {
		t.Fatalf(msgStoreOK, err)
	}

	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("cerrar almacen migrado: %v", err)
		}
	})

	history := NewHistory(store)

	operations, err := history.List(context.Background(), &HistoryFilter{})
	if err != nil {
		t.Fatalf(msgListError, err)
	}

	if len(operations) != 1 {
		t.Fatalf("Listar() devolvió %d, want 1", len(operations))
	}

	if operations[0].Description != "" {
		t.Errorf("Descripcion de la fila anterior = %q, want vacía", operations[0].Description)
	}

	if err := history.Register(context.Background(), &Operation{
		VM: testMachineName, Module: modulePackages, Operation: "POST /api/packages",
		Description: descriptionNano, Outcome: "ok",
	}); err != nil {
		t.Fatalf("Registrar() error = %v", err)
	}

	operations, err = history.List(context.Background(), &HistoryFilter{})
	if err != nil {
		t.Fatalf(msgListError, err)
	}

	if len(operations) != 2 || operations[0].Description != descriptionNano {
		t.Errorf("operaciones = %+v, want la descripción nueva", operations)
	}
}

// TestStoreSavesMachineActive comprueba la persistencia del vínculo.
func TestStoreSavesMachineActive(t *testing.T) {
	t.Parallel()

	store, pathBD, pathKey := newTestDB(t)

	if err := store.SaveActive(testMachineName); err != nil {
		t.Fatalf("GuardarActiva() error = %v", err)
	}

	other, err := NewStore(pathBD, pathKey)
	if err != nil {
		t.Fatalf(msgStoreOK, err)
	}

	t.Cleanup(func() {
		if err := other.Close(); err != nil {
			t.Errorf("cerrar segundo almacen: %v", err)
		}
	})

	if active := other.Active(); active != testMachineName {
		t.Errorf("Activa() = %q, want %q", active, testMachineName)
	}
}

// TestStoreCloseWithoutBase comprueba que cerrar un almacén sin base de
// datos no falla, tanto con receptor nulo como con la conexión vacía.
func TestStoreCloseWithoutBase(t *testing.T) {
	t.Parallel()

	var nulo *Store

	if err := nulo.Close(); err != nil {
		t.Fatalf("Close() con receptor nulo error = %v, want nil", err)
	}

	empty := &Store{}

	if err := empty.Close(); err != nil {
		t.Fatalf("Close() sin conexión error = %v, want nil", err)
	}
}

// connectorCierre es un conector de prueba cuyo cierre falla.
type connectorCierre struct{}

// Connect devuelve una conexión de prueba.
func (connectorCierre) Connect(context.Context) (driver.Conn, error) {
	return connectionCierre{}, nil
}

// Driver devuelve el controlador de prueba.
func (connectorCierre) Driver() driver.Driver { return controladorCierre{} }

// Close devuelve el error configurado al cerrar la base.
func (connectorCierre) Close() error { return errCierreFailed }

// controladorCierre es el controlador del conector de prueba.
type controladorCierre struct{}

// Open no se usa porque el conector entrega la conexión directamente.
func (controladorCierre) Open(string) (driver.Conn, error) {
	return connectionCierre{}, nil
}

// connectionCierre es una conexión de prueba que no ejecuta sentencias.
type connectionCierre struct{}

// Prepare no está soportado por la conexión de prueba.
func (connectionCierre) Prepare(string) (driver.Stmt, error) {
	return nil, errCierreFailed
}

// Close devuelve el error configurado.
func (connectionCierre) Close() error { return nil }

// Begin no está soportado por la conexión de prueba.
func (connectionCierre) Begin() (driver.Tx, error) { return nil, errCierreFailed }

// errCierreFailed es el error que devuelve el conector de prueba.
var errCierreFailed = errors.New("cierre fallido")

// TestStoreCloseReturnsError comprueba que Close propaga el fallo del
// cierre de la base de datos.
func TestStoreCloseReturnsError(t *testing.T) {
	t.Parallel()

	store := &Store{db: sql.OpenDB(connectorCierre{})}

	if err := store.Close(); !errors.Is(err, errCierreFailed) {
		t.Fatalf("Close() error = %v, want %v", err, errCierreFailed)
	}
}
