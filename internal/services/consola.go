// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// schemaUsersConsole crea la tabla de cuentas de la consola.
const schemaUsersConsole = `
CREATE TABLE IF NOT EXISTS usuarios_consola (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	usuario TEXT NOT NULL UNIQUE,
	hash TEXT NOT NULL,
	creado_en TEXT NOT NULL
);`

// consoleSecretKey es la clave de configuración con el secreto que firma
// las cookies de sesión.
const consoleSecretKey = "consola_secreto"

// SessionLifetime es el tiempo que dura una sesión de la consola.
const SessionLifetime = 12 * time.Hour

// Límites de la contraseña de la consola.
const (
	minPasswordConsole = 8
	maxPasswordConsole = 128
)

// consoleUserPattern valida el nombre de usuario de la consola.
var consoleUserPattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]{2,31}$`)

// ErrConsoleConfigured indica que la consola ya tiene una cuenta.
var ErrConsoleConfigured = errors.New("la consola ya tiene una cuenta")

// ErrConsoleData indica que el usuario o la contraseña no cumplen las
// reglas de validación.
var ErrConsoleData = errors.New("los datos de la cuenta no son válidos")

// Console autentica a los usuarios de la consola con una cuenta guardada
// en SQLite y emite cookies de sesión firmadas.
type Console struct {
	store *Store
	now   func() time.Time
}

// NewConsole construye el servicio de autenticación sobre el almacén.
func NewConsole(store *Store) *Console {
	return &Console{store: store, now: time.Now}
}

// IsConfigured informa si ya existe una cuenta de la consola.
func (c *Console) IsConfigured() (bool, error) {
	var total int

	if err := c.store.db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM usuarios_consola`).Scan(&total); err != nil {
		return false, fmt.Errorf("contar las cuentas de la consola: %w", err)
	}

	return total > 0, nil
}

// Configure crea la primera cuenta de la consola. Falla con
// ErrConsoleConfigured si ya existe una y con ErrConsoleData si el
// usuario o la contraseña no son válidos.
func (c *Console) Configure(ctx context.Context, user, password string) error {
	if err := validateConsoleData(user, password); err != nil {
		return err
	}

	configurada, err := c.IsConfigured()
	if err != nil {
		return err
	}

	if configurada {
		return ErrConsoleConfigured
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("derivar la contraseña de la consola: %w", err)
	}

	if _, err := c.store.db.ExecContext(ctx, `
		INSERT INTO usuarios_consola (usuario, hash, creado_en) VALUES (?, ?, ?)`,
		user, string(hash), c.now().UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("guardar la cuenta de la consola: %w", err)
	}

	return nil
}

// Authenticate comprueba el usuario y la contraseña. Devuelve false sin
// error cuando no coinciden, para no revelar cuál de los dos falló.
func (c *Console) Authenticate(ctx context.Context, user, password string) (bool, error) {
	if !consoleUserPattern.MatchString(user) || password == "" {
		return false, nil
	}

	var hash string

	err := c.store.db.QueryRowContext(ctx,
		`SELECT hash FROM usuarios_consola WHERE usuario = ?`, user).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil //nolint:nilerr // una cuenta inexistente es un acceso fallido, no un error.
	}

	if err != nil {
		return false, fmt.Errorf("leer la cuenta de la consola: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return false, nil //nolint:nilerr // una contraseña incorrecta es un acceso fallido, no un error.
	}

	return true, nil
}

// OpenSession emite una cookie de sesión firmada para el usuario.
func (c *Console) OpenSession(user string) (string, error) {
	secret, err := c.store.consoleSecret()
	if err != nil {
		return "", err
	}

	expires := c.now().Add(SessionLifetime).Unix()
	content := user + "|" + strconv.FormatInt(expires, 10)

	return content + "|" + signConsole(secret, content), nil
}

// Validate comprueba la firma y la vigencia de una cookie de sesión y
// devuelve el usuario cuando es válida.
func (c *Console) Validate(token string) (string, bool) {
	parts := strings.Split(token, "|")
	if len(parts) != 3 {
		return "", false
	}

	user, expiresText, firma := parts[0], parts[1], parts[2]

	expires, err := strconv.ParseInt(expiresText, 10, 64)
	if err != nil || c.now().Unix() >= expires {
		return "", false
	}

	secret, err := c.store.consoleSecret()
	if err != nil {
		return "", false
	}

	esperada := signConsole(secret, user+"|"+expiresText)
	if !hmac.Equal([]byte(firma), []byte(esperada)) {
		return "", false
	}

	return user, true
}

// validateConsoleData aplica las reglas del usuario y de la contraseña.
func validateConsoleData(user, password string) error {
	if !consoleUserPattern.MatchString(user) {
		return fmt.Errorf("%w: el usuario debe tener entre 3 y 32 caracteres y empezar por letra o guion bajo", ErrConsoleData)
	}

	if len(password) < minPasswordConsole || len(password) > maxPasswordConsole {
		return fmt.Errorf("%w: la contraseña debe tener entre %d y %d caracteres",
			ErrConsoleData, minPasswordConsole, maxPasswordConsole)
	}

	if strings.ContainsRune(password, 0) {
		return fmt.Errorf("%w: la contraseña no puede contener NUL", ErrConsoleData)
	}

	return nil
}

// signConsole calcula el HMAC-SHA256 del contenido de la cookie.
func signConsole(secret []byte, content string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(content))

	return hex.EncodeToString(mac.Sum(nil))
}

// consoleSecret devuelve el secreto que firma las cookies y lo genera en
// el primer uso.
func (a *Store) consoleSecret() ([]byte, error) {
	var value string

	err := a.db.QueryRowContext(context.Background(),
		`SELECT valor FROM configuracion WHERE clave = ?`, consoleSecretKey).Scan(&value)
	if err == nil {
		secret, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return nil, fmt.Errorf("decodificar el secreto de la consola: %w", err)
		}

		return secret, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("leer el secreto de la consola: %w", err)
	}

	secret := make([]byte, 32)
	if _, err := randomRead(secret); err != nil {
		return nil, fmt.Errorf("generar el secreto de la consola: %w", err)
	}

	if _, err := a.db.ExecContext(context.Background(),
		`INSERT INTO configuracion (clave, valor) VALUES (?, ?)`,
		consoleSecretKey, base64.StdEncoding.EncodeToString(secret),
	); err != nil {
		return nil, fmt.Errorf("guardar el secreto de la consola: %w", err)
	}

	return secret, nil
}
