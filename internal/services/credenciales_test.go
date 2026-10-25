// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// nuevaBaseDePrueba crea la base y la clave en un directorio temporal.
func nuevaBaseDePrueba(t *testing.T) (almacen *Almacen, rutaBD, rutaClave string) {
	t.Helper()

	directorio := t.TempDir()
	rutaBD = filepath.Join(directorio, "vmcons.db")
	rutaClave = filepath.Join(directorio, archivoClave)

	almacen, err := NuevoAlmacen(rutaBD, rutaClave)
	if err != nil {
		t.Fatalf(msgAlmacenOK, err)
	}

	return almacen, rutaBD, rutaClave
}

// TestAlmacenGuardaYCarga comprueba la persistencia en SQLite.
func TestAlmacenGuardaYCarga(t *testing.T) {
	t.Parallel()

	almacen, rutaBD, _ := nuevaBaseDePrueba(t)

	if err := almacen.Guardar(nombreDebian, Credenciales{
		Usuario: usuarioRoot, Password: passwordPrueba, Puerto: 22, Registrada: true,
	}); err != nil {
		t.Fatalf(msgGuardarOK, err)
	}

	otro, err := NuevoAlmacen(rutaBD, filepath.Join(filepath.Dir(rutaBD), archivoClave))
	if err != nil {
		t.Fatalf(msgAlmacenOK, err)
	}

	credenciales, ok := otro.Obtener(nombreDebian)
	if !ok {
		t.Fatal("Obtener() no encontró las credenciales")
	}

	if credenciales.Usuario != usuarioRoot || credenciales.Password != passwordPrueba || !credenciales.Registrada {
		t.Errorf("credenciales = %+v, want root/secreto/registrada", credenciales)
	}
}

// TestAlmacenCifraLaContrasena comprueba que la base no guarda el texto
// plano y que la clave es privada.
func TestAlmacenCifraLaContrasena(t *testing.T) {
	t.Parallel()

	almacen, rutaBD, rutaClave := nuevaBaseDePrueba(t)

	if err := almacen.Guardar(nombreDebian, Credenciales{Usuario: usuarioRoot, Password: passwordPrueba}); err != nil {
		t.Fatalf(msgGuardarOK, err)
	}

	db, err := sql.Open("sqlite", rutaBD)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("cerrar la base: %v", err)
		}
	}()

	var cifrado string
	if err := db.QueryRowContext(context.Background(),
		`SELECT password FROM credenciales WHERE vm = ?`, nombreDebian,
	).Scan(&cifrado); err != nil {
		t.Fatalf("leer la contraseña: %v", err)
	}

	if cifrado == passwordPrueba {
		t.Error("la contraseña se guardó en texto plano")
	}

	info, err := os.Stat(rutaClave)
	if err != nil {
		t.Fatalf("Stat() de la clave: %v", err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Errorf("permisos de la clave = %v, want 0600", info.Mode().Perm())
	}
}

// TestAlmacenRegistradaYBorrar comprueba la marca y el borrado.
func TestAlmacenRegistradaYBorrar(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)

	if err := almacen.Guardar(vmPrueba, Credenciales{Usuario: "admin", Password: "clave", Registrada: true}); err != nil {
		t.Fatalf(msgGuardarOK, err)
	}

	if !almacen.Registrada(vmPrueba) {
		t.Error("Registrada() = false, want true")
	}

	if err := almacen.Borrar(vmPrueba); err != nil {
		t.Fatalf("Borrar() error = %v", err)
	}

	if almacen.Registrada(vmPrueba) {
		t.Error("Registrada() = true tras borrar")
	}

	if _, ok := almacen.Obtener(vmPrueba); ok {
		t.Error("Obtener() encontró credenciales borradas")
	}
}

// TestAlmacenClaveCorrupta rechaza una clave con longitud inválida.
func TestAlmacenClaveCorrupta(t *testing.T) {
	t.Parallel()

	directorio := t.TempDir()
	rutaClave := filepath.Join(directorio, archivoClave)

	if err := os.WriteFile(rutaClave, []byte("corta"), 0o600); err != nil {
		t.Fatalf("preparar la clave: %v", err)
	}

	_, err := NuevoAlmacen(filepath.Join(directorio, "vmcons.db"), rutaClave)
	if err == nil {
		t.Fatal("NuevoAlmacen() error = nil, want el error de clave inválida")
	}
}

// TestAlmacenGuardaLaMaquinaActiva comprueba la persistencia del vínculo.
func TestAlmacenGuardaLaMaquinaActiva(t *testing.T) {
	t.Parallel()

	almacen, rutaBD, rutaClave := nuevaBaseDePrueba(t)

	if err := almacen.GuardarActiva(nombreDebian); err != nil {
		t.Fatalf("GuardarActiva() error = %v", err)
	}

	otro, err := NuevoAlmacen(rutaBD, rutaClave)
	if err != nil {
		t.Fatalf(msgAlmacenOK, err)
	}

	if activa := otro.Activa(); activa != nombreDebian {
		t.Errorf("Activa() = %q, want %q", activa, nombreDebian)
	}
}
