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

// Credenciales son los datos de acceso a una máquina virtual. La
// contraseña se cifra con AES-GCM antes de guardarse en SQLite.
type Credenciales struct {
	// Usuario es el usuario con el que se abre la sesión.
	Usuario string
	// Password es la contraseña del usuario.
	Password string
	// Puerto es el puerto SSH; cero significa el puerto por omisión.
	Puerto uint16
	// Direccion es una dirección alternativa (por ejemplo, un reenvío
	// de puertos); vacía significa descubrirla con Guest Additions.
	Direccion string
	// Registrada indica que la consola ya preparó SSH en la máquina.
	Registrada bool
}

// esquemaCredenciales crea la tabla de credenciales si no existe.
const esquemaCredenciales = `
CREATE TABLE IF NOT EXISTS credenciales (
	vm TEXT PRIMARY KEY,
	usuario TEXT NOT NULL,
	password TEXT NOT NULL,
	puerto INTEGER NOT NULL DEFAULT 22,
	direccion TEXT NOT NULL DEFAULT '',
	registrada INTEGER NOT NULL DEFAULT 0,
	actualizado_en TEXT NOT NULL
);`

// esquemaOperaciones crea la tabla del historial si no existe.
const esquemaOperaciones = `
CREATE TABLE IF NOT EXISTS operaciones (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	fecha_hora TEXT NOT NULL,
	vm TEXT NOT NULL,
	modulo TEXT NOT NULL,
	operacion TEXT NOT NULL,
	resultado TEXT NOT NULL,
	detalle TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS operaciones_fecha ON operaciones (fecha_hora);
CREATE INDEX IF NOT EXISTS operaciones_vm ON operaciones (vm);`

// esquemaConfiguracion crea la tabla de ajustes de la consola.
const esquemaConfiguracion = `
CREATE TABLE IF NOT EXISTS configuracion (
	clave TEXT PRIMARY KEY,
	valor TEXT NOT NULL
);`

// claveMaquinaActiva es la clave de la máquina vinculada.
const claveMaquinaActiva = "vm_activa"

// Almacen guarda las credenciales por máquina en SQLite y cifra la
// contraseña con una clave local.
type Almacen struct {
	db    *sql.DB
	clave []byte
}

// NuevoAlmacen abre la base SQLite, crea el esquema y carga o genera la
// clave de cifrado.
func NuevoAlmacen(rutaBD, rutaClave string) (*Almacen, error) {
	if err := os.MkdirAll(filepath.Dir(rutaBD), 0o700); err != nil {
		return nil, fmt.Errorf("crear el directorio de datos: %w", err)
	}

	clave, err := cargarClave(rutaClave)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", rutaBD)
	if err != nil {
		return nil, fmt.Errorf("abrir la base de datos: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.ExecContext(context.Background(), esquemaCredenciales); err != nil {
		return nil, fmt.Errorf("crear el esquema de credenciales: %w", err)
	}

	if _, err := db.ExecContext(context.Background(), esquemaOperaciones); err != nil {
		return nil, fmt.Errorf("crear el esquema del historial: %w", err)
	}

	if _, err := db.ExecContext(context.Background(), esquemaConfiguracion); err != nil {
		return nil, fmt.Errorf("crear el esquema de configuración: %w", err)
	}

	// #nosec G302 -- es la base de datos, no un ejecutable.
	if err := os.Chmod(rutaBD, 0o600); err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Warn("ajustar permisos de la base de datos", claveErrorLog, err)
	}

	return &Almacen{db: db, clave: clave}, nil
}

// cargarClave lee la clave local de 32 bytes o la genera si no existe.
func cargarClave(ruta string) ([]byte, error) {
	// #nosec G304 -- la ruta la fija la consola al arrancar.
	contenido, err := os.ReadFile(ruta)
	if err == nil {
		if len(contenido) != 32 {
			return nil, fmt.Errorf("la clave %q debe tener 32 bytes", ruta)
		}

		return contenido, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("leer la clave de cifrado: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return nil, fmt.Errorf("crear el directorio de la clave: %w", err)
	}

	clave := make([]byte, 32)
	if _, err := rand.Read(clave); err != nil {
		return nil, fmt.Errorf("generar la clave de cifrado: %w", err)
	}

	// #nosec G306 -- la clave debe ser privada.
	if err := os.WriteFile(ruta, clave, 0o600); err != nil {
		return nil, fmt.Errorf("guardar la clave de cifrado: %w", err)
	}

	return clave, nil
}

// Obtener devuelve las credenciales de una máquina.
func (a *Almacen) Obtener(vm string) (Credenciales, bool) {
	var (
		credenciales Credenciales
		cifrado      string
		registrada   int
	)

	err := a.db.QueryRowContext(context.Background(),
		`SELECT usuario, password, puerto, direccion, registrada FROM credenciales WHERE vm = ?`, vm,
	).Scan(&credenciales.Usuario, &cifrado, &credenciales.Puerto, &credenciales.Direccion, &registrada)
	if errors.Is(err, sql.ErrNoRows) {
		return Credenciales{}, false
	}

	if err != nil {
		slog.Warn("leer las credenciales", "vm", vm, claveErrorLog, err)

		return Credenciales{}, false
	}

	credenciales.Registrada = registrada == 1

	password, err := a.descifrar(cifrado)
	if err != nil {
		slog.Warn("descifrar la contraseña", "vm", vm, claveErrorLog, err)
	}

	credenciales.Password = password

	return credenciales, true
}

// Guardar registra o actualiza las credenciales de una máquina.
func (a *Almacen) Guardar(vm string, credenciales Credenciales) error {
	password, err := a.cifrar(credenciales.Password)
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
		vm, credenciales.Usuario, password, credenciales.Puerto, credenciales.Direccion,
		booleano(credenciales.Registrada), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("guardar las credenciales: %w", err)
	}

	return nil
}

// Borrar elimina las credenciales de una máquina.
func (a *Almacen) Borrar(vm string) error {
	if _, err := a.db.ExecContext(context.Background(), `DELETE FROM credenciales WHERE vm = ?`, vm); err != nil {
		return fmt.Errorf("borrar las credenciales: %w", err)
	}

	return nil
}

// Activa devuelve la máquina vinculada que se guardó al cerrar.
func (a *Almacen) Activa() string {
	var valor string

	err := a.db.QueryRowContext(context.Background(),
		`SELECT valor FROM configuracion WHERE clave = ?`, claveMaquinaActiva,
	).Scan(&valor)
	if err != nil {
		return ""
	}

	return valor
}

// GuardarActiva persiste la máquina vinculada para el próximo arranque.
func (a *Almacen) GuardarActiva(vm string) error {
	_, err := a.db.ExecContext(context.Background(), `
		INSERT INTO configuracion (clave, valor) VALUES (?, ?)
		ON CONFLICT(clave) DO UPDATE SET valor = excluded.valor`,
		claveMaquinaActiva, vm,
	)
	if err != nil {
		return fmt.Errorf("guardar la máquina vinculada: %w", err)
	}

	return nil
}

// Registrada informa si la máquina ya fue registrada en la consola.
func (a *Almacen) Registrada(vm string) bool {
	var registrada int

	err := a.db.QueryRowContext(context.Background(), `SELECT registrada FROM credenciales WHERE vm = ?`, vm).Scan(&registrada)
	if err != nil {
		return false
	}

	return registrada == 1
}

// Guest implementa la resolución de credenciales de Guest Additions.
func (a *Almacen) Guest(vm string) (usuario, password string, ok bool) {
	credenciales, ok := a.Obtener(vm)

	return credenciales.Usuario, credenciales.Password, ok
}

// cifrar protege la contraseña con AES-256-GCM y la codifica en base64.
func (a *Almacen) cifrar(texto string) (string, error) {
	bloque, err := aes.NewCipher(a.clave)
	if err != nil {
		return "", fmt.Errorf("preparar el cifrado: %w", err)
	}

	gcm, err := cipher.NewGCM(bloque)
	if err != nil {
		return "", fmt.Errorf("preparar GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generar el nonce: %w", err)
	}

	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(texto), nil)), nil
}

// descifrar recupera la contraseña guardada con cifrar.
func (a *Almacen) descifrar(texto string) (string, error) {
	datos, err := base64.StdEncoding.DecodeString(texto)
	if err != nil {
		return "", fmt.Errorf("decodificar la contraseña: %w", err)
	}

	bloque, err := aes.NewCipher(a.clave)
	if err != nil {
		return "", fmt.Errorf("preparar el descifrado: %w", err)
	}

	gcm, err := cipher.NewGCM(bloque)
	if err != nil {
		return "", fmt.Errorf("preparar GCM: %w", err)
	}

	if len(datos) < gcm.NonceSize() {
		return "", fmt.Errorf("la contraseña cifrada está incompleta")
	}

	nonce, cifrado := datos[:gcm.NonceSize()], datos[gcm.NonceSize():]

	plano, err := gcm.Open(nil, nonce, cifrado, nil)
	if err != nil {
		return "", fmt.Errorf("descifrar la contraseña: %w", err)
	}

	return string(plano), nil
}

// booleano convierte un bool en el entero que usa SQLite.
func booleano(valor bool) int {
	if valor {
		return 1
	}

	return 0
}
