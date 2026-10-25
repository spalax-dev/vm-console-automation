// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAlmacenGuardaYCarga comprueba la persistencia y los permisos.
func TestAlmacenGuardaYCarga(t *testing.T) {
	t.Parallel()

	ruta := filepath.Join(t.TempDir(), archivoCredenciales)
	almacen := NuevoAlmacen(ruta)

	if err := almacen.Guardar("debian-1", Credenciales{Usuario: usuarioRoot, Password: passwordPrueba, Puerto: 22}); err != nil {
		t.Fatalf("Guardar() error = %v, want nil", err)
	}

	info, err := os.Stat(ruta)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Errorf("permisos = %v, want 0600", info.Mode().Perm())
	}

	otro := NuevoAlmacen(ruta)

	credenciales, ok := otro.Obtener("debian-1")
	if !ok {
		t.Fatal("Obtener() no encontró las credenciales")
	}

	if credenciales.Usuario != usuarioRoot || credenciales.Password != passwordPrueba {
		t.Errorf("credenciales = %+v, want root/secreto", credenciales)
	}
}

// TestAlmacenGuestYOlvidar comprueba la resolución para Guest Additions.
func TestAlmacenGuestYOlvidar(t *testing.T) {
	t.Parallel()

	almacen := NuevoAlmacen(filepath.Join(t.TempDir(), archivoCredenciales))

	if err := almacen.Guardar(vmPrueba, Credenciales{Usuario: "admin", Password: "clave"}); err != nil {
		t.Fatalf("Guardar() error = %v, want nil", err)
	}

	usuario, password, ok := almacen.Guest(vmPrueba)
	if !ok || usuario != "admin" || password != "clave" {
		t.Errorf("Guest() = %q, %q, %v, want admin/clave/true", usuario, password, ok)
	}

	if err := almacen.Borrar(vmPrueba); err != nil {
		t.Fatalf("Borrar() error = %v, want nil", err)
	}

	if _, ok := almacen.Obtener(vmPrueba); ok {
		t.Error("Obtener() encontró credenciales borradas")
	}
}

// TestAlmacenArchivoCorrupto arranca vacío si el JSON no es válido.
func TestAlmacenArchivoCorrupto(t *testing.T) {
	t.Parallel()

	ruta := filepath.Join(t.TempDir(), archivoCredenciales)
	if err := os.WriteFile(ruta, []byte("{"), 0o600); err != nil {
		t.Fatalf(msgPrepararArchivo, err)
	}

	almacen := NuevoAlmacen(ruta)

	if _, ok := almacen.Obtener(vmPrueba); ok {
		t.Error("Obtener() devolvió datos de un archivo corrupto")
	}
}
