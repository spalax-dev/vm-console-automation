// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // controlador SQLite puro Go, sin CGO
)

// sqlDriver es el controlador de base de datos; es una variable para
// poder simular fallos de apertura en las pruebas.
var sqlDriver = "sqlite"

// adjustPermissions cambia los permisos de la base; es una variable por la
// misma razón.
var adjustPermissions = os.Chmod

// randomRead llena los buffers de cifrado y de secretos; es una variable
// para poder simular fallos de entropía en las pruebas.
var randomRead = rand.Read

// newGCM construye el modo GCM; es una variable por la misma razón.
var newGCM = cipher.NewGCM

// Credentials son los datos de acceso a una máquina virtual. La
// contraseña se cifra con AES-GCM antes de guardarse en SQLite.
type Credentials struct {
	// User es el usuario con el que se abre la sesión.
	User string
	// Password es la contraseña del usuario.
	Password string
	// Port es el puerto SSH; cero significa el puerto por omisión.
	Port uint16
	// Address es una dirección alternativa (por ejemplo, un reenvío
	// de puertos); vacía significa descubrirla con Guest Additions.
	Address string
	// IsRegistered indica que la consola ya preparó SSH en la máquina.
	IsRegistered bool
}

// schemaCredentials crea la tabla de credenciales si no existe.
const schemaCredentials = `
CREATE TABLE IF NOT EXISTS credenciales (
	vm TEXT PRIMARY KEY,
	usuario TEXT NOT NULL,
	password TEXT NOT NULL,
	puerto INTEGER NOT NULL DEFAULT 22,
	direccion TEXT NOT NULL DEFAULT '',
	registrada INTEGER NOT NULL DEFAULT 0,
	actualizado_en TEXT NOT NULL
);`

// schemaOperations crea la tabla del historial si no existe.
const schemaOperations = `
CREATE TABLE IF NOT EXISTS operaciones (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	fecha_hora TEXT NOT NULL,
	vm TEXT NOT NULL,
	modulo TEXT NOT NULL,
	operacion TEXT NOT NULL,
	descripcion TEXT NOT NULL DEFAULT '',
	resultado TEXT NOT NULL,
	detalle TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS operaciones_fecha ON operaciones (fecha_hora);
CREATE INDEX IF NOT EXISTS operaciones_vm ON operaciones (vm);`

// schemaConfig crea la tabla de ajustes de la consola.
const schemaConfig = `
CREATE TABLE IF NOT EXISTS configuracion (
	clave TEXT PRIMARY KEY,
	valor TEXT NOT NULL
);`

// activeMachineKey es la clave de la máquina vinculada.
const activeMachineKey = "vm_activa"

// Store guarda las credenciales por máquina en SQLite y cifra la
// contraseña con una clave local.
type Store struct {
	db  *sql.DB
	key []byte
}

// NewStore abre la base SQLite, crea el esquema y carga o genera la
// clave de cifrado.
func NewStore(pathBD, pathKey string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(pathBD), 0o700); err != nil {
		return nil, fmt.Errorf("crear el directorio de datos: %w", err)
	}

	key, err := loadKey(pathKey)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(sqlDriver, pathBD)
	if err != nil {
		return nil, fmt.Errorf("abrir la base de datos: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := createSchema(db); err != nil {
		return nil, err
	}

	// #nosec G302 -- es la base de datos, no un ejecutable.
	if err := adjustPermissions(pathBD, 0o600); err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Warn("ajustar permisos de la base de datos", logErrorKey, err)
	}

	return &Store{db: db, key: key}, nil
}

// createSchema crea las tablas de la consola y aplica las migraciones
// necesarias sobre bases creadas con una versión anterior.
func createSchema(db *sql.DB) error {
	schemas := []struct {
		name string
		sql  string
	}{
		{"credenciales", schemaCredentials},
		{"historial", schemaOperations},
		{"configuración", schemaConfig},
		{"cuentas de la consola", schemaUsersConsole},
	}

	for _, schema := range schemas {
		if _, err := db.ExecContext(context.Background(), schema.sql); err != nil {
			return fmt.Errorf("crear el esquema de %s: %w", schema.name, err)
		}
	}

	return ensureOperationDescriptions(db)
}

// Close cierra la conexión con la base de datos SQLite.
func (a *Store) Close() error {
	if a == nil || a.db == nil {
		return nil
	}

	if err := a.db.Close(); err != nil {
		return fmt.Errorf("cerrar la base de datos: %w", err)
	}

	return nil
}

// ensureOperationDescriptions agrega la columna descripcion a bases
// creadas con una versión anterior.
func ensureOperationDescriptions(db *sql.DB) error {
	exists, err := columnExists(db, "operaciones", "descripcion")
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	if _, err := db.ExecContext(context.Background(),
		`ALTER TABLE operaciones ADD COLUMN descripcion TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("agregar la descripción al historial: %w", err)
	}

	return nil
}

// columnExists comprueba si una tabla SQLite tiene una columna. Cierra
// las filas antes de volver para liberar la única conexión de la base.
func columnExists(db *sql.DB, table, columna string) (bool, error) {
	// #nosec G201 -- la tabla y la columna las fija el código, no el usuario.
	rows, err := db.QueryContext(context.Background(), `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return false, fmt.Errorf("leer el esquema de %s: %w", table, err)
	}

	defer func() {
		if err := rows.Close(); err != nil {
			slog.Warn("cerrar el esquema de la tabla", "tabla", table, logErrorKey, err)
		}
	}()

	for rows.Next() {
		var (
			cid       int
			name      string
			kind      string
			notNulo   int
			byDefecto sql.NullString
			key       int
		)

		if err := rows.Scan(&cid, &name, &kind, &notNulo, &byDefecto, &key); err != nil {
			return false, fmt.Errorf("leer una columna de %s: %w", table, err)
		}

		if name == columna {
			return true, nil
		}
	}

	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("recorrer el esquema de %s: %w", table, err)
	}

	return false, nil
}

// loadKey lee la clave local de 32 bytes o la genera si no existe.
func loadKey(path string) ([]byte, error) {
	// #nosec G304 -- la ruta la fija la consola al arrancar.
	content, err := os.ReadFile(path)
	if err == nil {
		if len(content) != 32 {
			return nil, fmt.Errorf("la clave %q debe tener 32 bytes", path)
		}

		return content, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("leer la clave de cifrado: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("crear el directorio de la clave: %w", err)
	}

	key := make([]byte, 32)
	if _, err := randomRead(key); err != nil {
		return nil, fmt.Errorf("generar la clave de cifrado: %w", err)
	}

	// #nosec G306 -- la clave debe ser privada.
	if err := os.WriteFile(path, key, 0o600); err != nil {
		return nil, fmt.Errorf("guardar la clave de cifrado: %w", err)
	}

	return key, nil
}

// Get devuelve las credenciales de una máquina.
func (a *Store) Get(vm string) (Credentials, bool) {
	var (
		credentials Credentials
		ciphertext  string
		registered  int
	)

	err := a.db.QueryRowContext(context.Background(),
		`SELECT usuario, password, puerto, direccion, registrada FROM credenciales WHERE vm = ?`, vm,
	).Scan(&credentials.User, &ciphertext, &credentials.Port, &credentials.Address, &registered)
	if errors.Is(err, sql.ErrNoRows) {
		return Credentials{}, false
	}

	if err != nil {
		slog.Warn("leer las credenciales", "vm", vm, logErrorKey, err)

		return Credentials{}, false
	}

	credentials.IsRegistered = registered == 1

	password, err := a.decrypt(ciphertext)
	if err != nil {
		slog.Warn("descifrar la contraseña", "vm", vm, logErrorKey, err)
	}

	credentials.Password = password

	return credentials, true
}

// Save registra o actualiza las credenciales de una máquina.
func (a *Store) Save(vm string, credentials Credentials) error {
	password, err := a.encrypt(credentials.Password)
	if err != nil {
		return err
	}

	_, err = a.db.ExecContext(context.Background(), `
		INSERT INTO credenciales (vm, usuario, password, puerto, direccion, registrada, actualizado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(vm) DO UPDATE SET
			usuario = excluded.usuario,
			password = excluded.password,
			puerto = excluded.puerto,
			direccion = excluded.direccion,
			registrada = excluded.registrada,
			actualizado_en = excluded.actualizado_en`,
		vm, credentials.User, password, credentials.Port, credentials.Address,
		boolInt(credentials.IsRegistered), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("guardar las credenciales: %w", err)
	}

	return nil
}

// Delete elimina las credenciales de una máquina.
func (a *Store) Delete(vm string) error {
	if _, err := a.db.ExecContext(context.Background(), `DELETE FROM credenciales WHERE vm = ?`, vm); err != nil {
		return fmt.Errorf("borrar las credenciales: %w", err)
	}

	return nil
}

// Active devuelve la máquina vinculada que se guardó al cerrar.
func (a *Store) Active() string {
	var value string

	err := a.db.QueryRowContext(context.Background(),
		`SELECT valor FROM configuracion WHERE clave = ?`, activeMachineKey,
	).Scan(&value)
	if err != nil {
		return ""
	}

	return value
}

// SaveActive persiste la máquina vinculada para el próximo arranque.
func (a *Store) SaveActive(vm string) error {
	_, err := a.db.ExecContext(context.Background(), `
		INSERT INTO configuracion (clave, valor) VALUES (?, ?)
		ON CONFLICT(clave) DO UPDATE SET valor = excluded.valor`,
		activeMachineKey, vm,
	)
	if err != nil {
		return fmt.Errorf("guardar la máquina vinculada: %w", err)
	}

	return nil
}

// IsRegistered informa si la máquina ya fue registrada en la consola.
func (a *Store) IsRegistered(vm string) bool {
	var registered int

	err := a.db.QueryRowContext(context.Background(), `SELECT registrada FROM credenciales WHERE vm = ?`, vm).Scan(&registered)
	if err != nil {
		return false
	}

	return registered == 1
}

// Guest implementa la resolución de credenciales de Guest Additions.
func (a *Store) Guest(vm string) (user, password string, ok bool) {
	credentials, ok := a.Get(vm)

	return credentials.User, credentials.Password, ok
}

// encrypt protege la contraseña con AES-256-GCM y la codifica en base64.
func (a *Store) encrypt(text string) (string, error) {
	bloque, err := aes.NewCipher(a.key)
	if err != nil {
		return "", fmt.Errorf("preparar el cifrado: %w", err)
	}

	gcm, err := newGCM(bloque)
	if err != nil {
		return "", fmt.Errorf("preparar GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := randomRead(nonce); err != nil {
		return "", fmt.Errorf("generar el nonce: %w", err)
	}

	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(text), nil)), nil
}

// decrypt recupera la contraseña guardada con cifrar.
func (a *Store) decrypt(text string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return "", fmt.Errorf("decodificar la contraseña: %w", err)
	}

	bloque, err := aes.NewCipher(a.key)
	if err != nil {
		return "", fmt.Errorf("preparar el descifrado: %w", err)
	}

	gcm, err := newGCM(bloque)
	if err != nil {
		return "", fmt.Errorf("preparar GCM: %w", err)
	}

	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("la contraseña cifrada está incompleta")
	}

	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]

	plano, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("descifrar la contraseña: %w", err)
	}

	return string(plano), nil
}

// boolInt convierte un bool en el entero que usa SQLite.
func boolInt(value bool) int {
	if value {
		return 1
	}

	return 0
}
