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

// TestUsuariosBorrarLlavesConPermisosDenegados ignora los archivos que no
// puede borrar.
func TestUsuariosBorrarLlavesConPermisosDenegados(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("ejecutando como root: los permisos no impiden borrar")
	}

	directorio := t.TempDir()

	for _, sufijo := range []string{"", ".pub"} {
		if err := os.WriteFile(filepath.Join(directorio, usuarioJorgePrueba+"_ed25519"+sufijo), []byte("x"), 0o600); err != nil {
			t.Fatalf("crear la llave: %v", err)
		}
	}

	if err := os.Chmod(directorio, 0o500); err != nil { // #nosec G302 -- permisos de la prueba.
		t.Fatalf("restringir el directorio: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chmod(directorio, 0o700); err != nil { // #nosec G302 -- permisos de la prueba.
			t.Logf("restaurar permisos: %v", err)
		}
	})

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
