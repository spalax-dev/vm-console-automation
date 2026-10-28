// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package config resuelve la configuración de la consola a partir de las
// variables de entorno y de las convenciones del sistema de archivos.
package config

import (
	"os"
	"path/filepath"
)

// Variables de entorno que configuran la consola.
const (
	EnvLog       = "VMCONS_LOG"        // formato del log: text o json
	EnvLogLevel  = "VMCONS_LOG_LEVEL"  // nivel mínimo del log
	EnvVM        = "VMCONS_VM"         // nombre o UUID de la VM en VirtualBox
	EnvSSHKey    = "VMCONS_SSH_KEY"    // ruta de la llave privada en el anfitrión
	EnvGuestUser = "VMCONS_GUEST_USER" // usuario de Guest Additions
	// #nosec G101 -- es el nombre de la variable de entorno, no la credencial.
	EnvGuestPass = "VMCONS_GUEST_PASS" // contraseña de Guest Additions
	EnvAddress   = "VMCONS_ADDR"       // dirección donde escucha la consola
	EnvWeb       = "VMCONS_WEB"        // directorio de la interfaz web
	EnvDB        = "VMCONS_DB"         // ruta de la base SQLite
)

// AddressProd es la dirección de escucha cuando VMCONS_ADDR no está definida.
const AddressProd = "127.0.0.1:8080"

// Valores de la convención de directorios del usuario.
const (
	appDirectory = "vmcons"
	envDataHome  = "XDG_DATA_HOME"
)

// Address devuelve la dirección de escucha configurada.
func Address() string {
	if value := os.Getenv(EnvAddress); value != "" {
		return value
	}

	return AddressProd
}

// DatabasePath devuelve la ruta de la base SQLite de la consola.
func DatabasePath() string {
	if value := os.Getenv(EnvDB); value != "" {
		return value
	}

	return filepath.Join(dataDirectory(), "vmcons.db")
}

// dataDirectory devuelve el directorio donde la consola guarda su base y
// la clave de cifrado: $XDG_DATA_HOME/vmcons o, si la variable falta o no
// es absoluta, ~/.local/share/vmcons. Nunca devuelve una ruta relativa al
// directorio de trabajo, salvo que el sistema no publique el del usuario.
func dataDirectory() string {
	if value := os.Getenv(envDataHome); filepath.IsAbs(value) {
		return filepath.Join(value, appDirectory)
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".local", "share", appDirectory)
	}

	if config, err := os.UserConfigDir(); err == nil && config != "" {
		return filepath.Join(config, appDirectory)
	}

	return "data"
}

// CipherKeyPath devuelve la clave local que cifra las contraseñas. Vive
// junto a la base para que ambas se muevan y se copien juntas.
func CipherKeyPath() string {
	return filepath.Join(filepath.Dir(DatabasePath()), "credenciales.key")
}

// SSHKeyPath devuelve la ruta de la llave privada de root: la variable de
// entorno o, por omisión, el directorio de configuración del usuario.
func SSHKeyPath() string {
	if value := os.Getenv(EnvSSHKey); value != "" {
		return value
	}

	config, err := os.UserConfigDir()
	if err != nil {
		config = "."
	}

	return filepath.Join(config, appDirectory, "id_ed25519")
}

// GuestUser devuelve el usuario de Guest Additions, «root» por omisión.
func GuestUser() string {
	if value := os.Getenv(EnvGuestUser); value != "" {
		return value
	}

	return "root"
}
