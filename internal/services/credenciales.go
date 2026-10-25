// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// Credenciales son los datos de acceso a una máquina virtual. La
// contraseña se guarda en un archivo con permisos 0600 porque los
// canales de Guest Additions y SSH la necesitan de forma no interactiva.
type Credenciales struct {
	// Usuario es el usuario con el que se abre la sesión.
	Usuario string `json:"usuario"`
	// Password es la contraseña del usuario.
	Password string `json:"password,omitempty"`
	// Puerto es el puerto SSH; cero significa el puerto por omisión.
	Puerto uint16 `json:"puerto,omitempty"`
	// Direccion es una dirección alternativa (por ejemplo, un reenvío
	// de puertos); vacía significa descubrirla con Guest Additions.
	Direccion string `json:"direccion,omitempty"`
}

// Almacen guarda las credenciales por máquina en un archivo JSON.
type Almacen struct {
	ruta  string
	mu    sync.RWMutex
	datos map[string]Credenciales
}

// NuevoAlmacen carga el archivo de credenciales; si no existe o está
// corrupto, arranca vacío y deja constancia en el log.
func NuevoAlmacen(ruta string) *Almacen {
	almacen := &Almacen{ruta: ruta, datos: map[string]Credenciales{}}

	// #nosec G304 -- la ruta la fija la consola al arrancar.
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			slog.Warn("leer las credenciales", "ruta", ruta, claveErrorLog, err)
		}

		return almacen
	}

	if err := json.Unmarshal(contenido, &almacen.datos); err != nil {
		slog.Warn("interpretar las credenciales", "ruta", ruta, claveErrorLog, err)

		almacen.datos = map[string]Credenciales{}
	}

	return almacen
}

// Obtener devuelve las credenciales de una máquina.
func (a *Almacen) Obtener(vm string) (Credenciales, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	credenciales, ok := a.datos[vm]

	return credenciales, ok
}

// Guardar registra las credenciales de una máquina y persiste el
// archivo.
func (a *Almacen) Guardar(vm string, credenciales Credenciales) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.datos[vm] = credenciales

	return a.escribir()
}

// Borrar elimina las credenciales de una máquina.
func (a *Almacen) Borrar(vm string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.datos, vm)

	return a.escribir()
}

// Guest implementa la resolución de credenciales de Guest Additions.
func (a *Almacen) Guest(vm string) (usuario, password string, ok bool) {
	credenciales, ok := a.Obtener(vm)

	return credenciales.Usuario, credenciales.Password, ok
}

// escribir guarda el mapa en disco con permisos 0600 y de forma atómica.
func (a *Almacen) escribir() error {
	if err := os.MkdirAll(filepath.Dir(a.ruta), 0o700); err != nil {
		return fmt.Errorf("crear el directorio de credenciales: %w", err)
	}

	// #nosec G117 -- el archivo es privado (0600) y es la única forma de
	// recordar la contraseña de Guest Additions entre ejecuciones.
	contenido, err := json.MarshalIndent(a.datos, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar las credenciales: %w", err)
	}

	temporal := a.ruta + ".tmp"

	// #nosec G306 -- el archivo contiene contraseñas y debe ser privado.
	if err := os.WriteFile(temporal, contenido, 0o600); err != nil {
		return fmt.Errorf("escribir las credenciales: %w", err)
	}

	if err := os.Rename(temporal, a.ruta); err != nil {
		return fmt.Errorf("reemplazar las credenciales: %w", err)
	}

	return nil
}
