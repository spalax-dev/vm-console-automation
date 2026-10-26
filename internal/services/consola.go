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

// esquemaUsuariosConsola crea la tabla de cuentas de la consola.
const esquemaUsuariosConsola = `
CREATE TABLE IF NOT EXISTS usuarios_consola (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	usuario TEXT NOT NULL UNIQUE,
	hash TEXT NOT NULL,
	creado_en TEXT NOT NULL
);`

// claveSecretoConsola es la clave de configuración con el secreto que firma
// las cookies de sesión.
const claveSecretoConsola = "consola_secreto"

// VigenciaSesionConsola es el tiempo que dura una sesión de la consola.
const VigenciaSesionConsola = 12 * time.Hour

// Límites de la contraseña de la consola.
const (
	minPasswordConsola = 8
	maxPasswordConsola = 128
)

// reUsuarioConsola valida el nombre de usuario de la consola.
var reUsuarioConsola = regexp.MustCompile(`^[a-z_][a-z0-9_-]{2,31}$`)

// ErrConsolaConfigurada indica que la consola ya tiene una cuenta.
var ErrConsolaConfigurada = errors.New("la consola ya tiene una cuenta")

// ErrDatosConsola indica que el usuario o la contraseña no cumplen las
// reglas de validación.
var ErrDatosConsola = errors.New("los datos de la cuenta no son válidos")

// Consola autentica a los usuarios de la consola con una cuenta guardada
// en SQLite y emite cookies de sesión firmadas.
type Consola struct {
	almacen *Almacen
	ahora   func() time.Time
}

// NuevoConsola construye el servicio de autenticación sobre el almacén.
func NuevoConsola(almacen *Almacen) *Consola {
	return &Consola{almacen: almacen, ahora: time.Now}
}

// Configurada informa si ya existe una cuenta de la consola.
func (c *Consola) Configurada() (bool, error) {
	var total int

	if err := c.almacen.db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM usuarios_consola`).Scan(&total); err != nil {
		return false, fmt.Errorf("contar las cuentas de la consola: %w", err)
	}

	return total > 0, nil
}

// Configurar crea la primera cuenta de la consola. Falla con
// ErrConsolaConfigurada si ya existe una y con ErrDatosConsola si el
// usuario o la contraseña no son válidos.
func (c *Consola) Configurar(ctx context.Context, usuario, password string) error {
	if err := validarDatosConsola(usuario, password); err != nil {
		return err
	}

	configurada, err := c.Configurada()
	if err != nil {
		return err
	}

	if configurada {
		return ErrConsolaConfigurada
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("derivar la contraseña de la consola: %w", err)
	}

	if _, err := c.almacen.db.ExecContext(ctx, `
		INSERT INTO usuarios_consola (usuario, hash, creado_en) VALUES (?, ?, ?)`,
		usuario, string(hash), c.ahora().UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("guardar la cuenta de la consola: %w", err)
	}

	return nil
}

// Autenticar comprueba el usuario y la contraseña. Devuelve false sin
// error cuando no coinciden, para no revelar cuál de los dos falló.
func (c *Consola) Autenticar(ctx context.Context, usuario, password string) (bool, error) {
	if !reUsuarioConsola.MatchString(usuario) || password == "" {
		return false, nil
	}

	var hash string

	err := c.almacen.db.QueryRowContext(ctx,
		`SELECT hash FROM usuarios_consola WHERE usuario = ?`, usuario).Scan(&hash)
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

// AbrirSesion emite una cookie de sesión firmada para el usuario.
func (c *Consola) AbrirSesion(usuario string) (string, error) {
	secreto, err := c.almacen.secretoConsola()
	if err != nil {
		return "", err
	}

	expira := c.ahora().Add(VigenciaSesionConsola).Unix()
	contenido := usuario + "|" + strconv.FormatInt(expira, 10)

	return contenido + "|" + firmarConsola(secreto, contenido), nil
}

// Validar comprueba la firma y la vigencia de una cookie de sesión y
// devuelve el usuario cuando es válida.
func (c *Consola) Validar(token string) (string, bool) {
	partes := strings.Split(token, "|")
	if len(partes) != 3 {
		return "", false
	}

	usuario, expiraTexto, firma := partes[0], partes[1], partes[2]

	expira, err := strconv.ParseInt(expiraTexto, 10, 64)
	if err != nil || c.ahora().Unix() >= expira {
		return "", false
	}

	secreto, err := c.almacen.secretoConsola()
	if err != nil {
		return "", false
	}

	esperada := firmarConsola(secreto, usuario+"|"+expiraTexto)
	if !hmac.Equal([]byte(firma), []byte(esperada)) {
		return "", false
	}

	return usuario, true
}

// validarDatosConsola aplica las reglas del usuario y de la contraseña.
func validarDatosConsola(usuario, password string) error {
	if !reUsuarioConsola.MatchString(usuario) {
		return fmt.Errorf("%w: el usuario debe tener entre 3 y 32 caracteres y empezar por letra o guion bajo", ErrDatosConsola)
	}

	if len(password) < minPasswordConsola || len(password) > maxPasswordConsola {
		return fmt.Errorf("%w: la contraseña debe tener entre %d y %d caracteres",
			ErrDatosConsola, minPasswordConsola, maxPasswordConsola)
	}

	if strings.ContainsRune(password, 0) {
		return fmt.Errorf("%w: la contraseña no puede contener NUL", ErrDatosConsola)
	}

	return nil
}

// firmarConsola calcula el HMAC-SHA256 del contenido de la cookie.
func firmarConsola(secreto []byte, contenido string) string {
	mac := hmac.New(sha256.New, secreto)
	mac.Write([]byte(contenido))

	return hex.EncodeToString(mac.Sum(nil))
}

// secretoConsola devuelve el secreto que firma las cookies y lo genera en
// el primer uso.
func (a *Almacen) secretoConsola() ([]byte, error) {
	var valor string

	err := a.db.QueryRowContext(context.Background(),
		`SELECT valor FROM configuracion WHERE clave = ?`, claveSecretoConsola).Scan(&valor)
	if err == nil {
		secreto, err := base64.StdEncoding.DecodeString(valor)
		if err != nil {
			return nil, fmt.Errorf("decodificar el secreto de la consola: %w", err)
		}

		return secreto, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("leer el secreto de la consola: %w", err)
	}

	secreto := make([]byte, 32)
	if _, err := aleatorio(secreto); err != nil {
		return nil, fmt.Errorf("generar el secreto de la consola: %w", err)
	}

	if _, err := a.db.ExecContext(context.Background(),
		`INSERT INTO configuracion (clave, valor) VALUES (?, ?)`,
		claveSecretoConsola, base64.StdEncoding.EncodeToString(secreto),
	); err != nil {
		return nil, fmt.Errorf("guardar el secreto de la consola: %w", err)
	}

	return secreto, nil
}
