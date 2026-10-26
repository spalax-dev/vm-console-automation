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

// usuarioJorgePrueba es la cuenta de las pruebas.
const usuarioJorgePrueba = "jorge"

// TestUsuariosObtenerSinSesion propaga el fallo del abridor.
func TestUsuariosObtenerSinSesion(t *testing.T) {
	t.Parallel()

	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, nil, errors.New("sin sesión"))

	if _, err := usuarios.Obtener(context.Background(), usuarioJorgePrueba); err == nil {
		t.Error("Obtener() error = nil, want el fallo de la sesión")
	}

	if _, err := usuarios.Eliminar(context.Background(), usuarioJorgePrueba, false); err == nil {
		t.Error("Eliminar() error = nil, want el fallo de la sesión")
	}
}

// TestUsuariosCrearConBusquedaRota propaga el fallo de getent.
func TestUsuariosCrearConBusquedaRota(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{err: errors.New("getent caído")}
	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	if _, err := usuarios.Crear(context.Background(), usuarioJorgePrueba, false, false); err == nil {
		t.Error("Crear() error = nil, want el fallo de getent")
	}

	if _, err := usuarios.Eliminar(context.Background(), usuarioJorgePrueba, false); err == nil {
		t.Error("Eliminar() error = nil, want el fallo de getent")
	}
}

// TestUsuariosCrearConGeneradorCaido propaga el fallo de ssh-keygen.
func TestUsuariosCrearConGeneradorCaido(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{}
	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{err: errors.New("sin ssh-keygen")}, sesion, nil)

	if _, err := usuarios.Crear(context.Background(), usuarioJorgePrueba, false, true); err == nil {
		t.Error("Crear() error = nil, want el fallo del generador")
	}
}

// TestUsuariosBorrarLlavesConFalloDeBorrado comprueba que no poder
// borrar la llave privada no impide eliminar la cuenta.
func TestUsuariosBorrarLlavesConFalloDeBorrado(t *testing.T) {
	t.Parallel()

	directorio := t.TempDir()
	ruta := filepath.Join(directorio, usuarioJorgePrueba+"_ed25519")

	// Un directorio no vacío hace que os.Remove falle, sin depender de
	// los permisos ni de que las pruebas corran como root.
	if err := os.MkdirAll(filepath.Join(ruta, "contenido"), 0o700); err != nil {
		t.Fatalf("crear el directorio de la llave: %v", err)
	}

	if err := os.WriteFile(filepath.Join(ruta, "contenido", "archivo"), []byte("x"), 0o600); err != nil {
		t.Fatalf("crear el archivo interno: %v", err)
	}

	sesion := &stubSession{results: []domain.Result{
		{Stdout: usuarioJorgePrueba + ":x:1000:1000::/home/jorge:/bin/bash\n"},
		{},
	}}
	usuarios := NuevoUsuarios(ConfigUsuarios{
		Sesiones:   &stubAbridor{sesion: sesion},
		Ejecutor:   &stubExecutor{},
		Directorio: directorio,
	})

	if _, err := usuarios.Eliminar(context.Background(), usuarioJorgePrueba, false); err != nil {
		t.Errorf("Eliminar() error = %v, want nil", err)
	}

	if _, err := os.Stat(ruta); err != nil {
		t.Errorf("la llave que no se pudo borrar desapareció: %v", err)
	}
}

// TestParsearUsuarioCorto devuelve una cuenta vacía con pocos campos.
func TestParsearUsuarioCorto(t *testing.T) {
	t.Parallel()

	if usuario := parsearUsuario("jorge:x:1000"); usuario.Nombre != "" {
		t.Errorf("parsearUsuario() = %+v, want vacío", usuario)
	}

	if usuario := parsearUsuario("jorge:x:no-numero:1000::/home/jorge:/bin/bash"); usuario.Nombre != "" {
		t.Errorf("parsearUsuario() = %+v, want vacío con UID inválido", usuario)
	}
}
