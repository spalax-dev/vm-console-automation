// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// archivoConTexto crea un archivo cualquiera y devuelve su ruta.
func archivoConTexto(t *testing.T) string {
	t.Helper()

	ruta := filepath.Join(t.TempDir(), "archivo")
	if err := os.WriteFile(ruta, []byte("x"), 0o600); err != nil {
		t.Fatalf("crear el archivo: %v", err)
	}

	return ruta
}

// TestNuevoAlmacenDirectorioImposible no puede crear el directorio de
// datos cuando la ruta es un archivo.
func TestNuevoAlmacenDirectorioImposible(t *testing.T) {
	t.Parallel()

	rutaBD := filepath.Join(archivoConTexto(t), "vmcons.db")

	if _, err := NuevoAlmacen(rutaBD, filepath.Join(t.TempDir(), "clave")); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo del directorio")
	}
}

// TestNuevoAlmacenClaveIlegible falla si la clave es un directorio.
func TestNuevoAlmacenClaveIlegible(t *testing.T) {
	t.Parallel()

	directorio := t.TempDir()

	if _, err := NuevoAlmacen(filepath.Join(directorio, archivoBDPrueba), directorio); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo de lectura de la clave")
	}
}

// TestNuevoAlmacenClaveNoEscribible falla si no puede guardar la clave.
func TestNuevoAlmacenClaveNoEscribible(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("ejecutando como root: los permisos no impiden escribir")
	}

	directorio := t.TempDir()
	claves := filepath.Join(directorio, "claves")

	if err := os.MkdirAll(claves, 0o500); err != nil {
		t.Fatalf("crear el directorio de claves: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chmod(claves, 0o700); err != nil { // #nosec G302 -- permisos de la prueba.
			t.Logf("restaurar permisos: %v", err)
		}
	})

	if _, err := NuevoAlmacen(filepath.Join(directorio, archivoBDPrueba), filepath.Join(claves, "clave")); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo al guardar la clave")
	}
}

// TestNuevoAlmacenSinEntropia falla si no hay entropía para la clave.
func TestNuevoAlmacenSinEntropia(t *testing.T) {
	original := aleatorio

	t.Cleanup(func() { aleatorio = original })

	aleatorio = func([]byte) (int, error) { return 0, errors.New("sin entropía") }

	directorio := t.TempDir()

	if _, err := NuevoAlmacen(filepath.Join(directorio, archivoBDPrueba), filepath.Join(directorio, "clave")); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo de entropía")
	}
}

// TestNuevoAlmacenSinEsquema falla si la base no admite escritura.
func TestNuevoAlmacenSinEsquema(t *testing.T) {
	t.Parallel()

	directorio := t.TempDir()
	rutaBD := filepath.Join(directorio, archivoBDPrueba)

	if err := os.WriteFile(rutaBD, nil, 0o400); err != nil {
		t.Fatalf("crear la base: %v", err)
	}

	_, err := NuevoAlmacen("file:"+rutaBD+"?mode=ro", filepath.Join(directorio, "clave"))
	if err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo del esquema")
	}
}

// TestGuardarYCargarCredenciales conserva los datos de la máquina.
func TestGuardarYCargarCredenciales(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)

	credenciales := Credenciales{Usuario: "jorge", Password: "secreto", Puerto: 22, Registrada: true}
	if err := almacen.Guardar("debian-1", credenciales); err != nil {
		t.Fatalf("Guardar() error = %v, want nil", err)
	}

	guardadas, ok := almacen.Obtener("debian-1")
	if !ok || guardadas.Password != "secreto" || !guardadas.Registrada {
		t.Errorf("Obtener() = %+v, %v, want las credenciales guardadas", guardadas, ok)
	}

	if _, ok := almacen.Obtener("otra"); ok {
		t.Error("Obtener() = true para una máquina sin credenciales")
	}

	usuario, password, ok := almacen.Guest("debian-1")
	if !ok || usuario != "jorge" || password != "secreto" {
		t.Errorf("Guest() = %q, %q, %v, want jorge, secreto", usuario, password, ok)
	}
}

// TestObtenerConDescifradoFallido devuelve la cuenta sin contraseña.
func TestObtenerConDescifradoFallido(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)

	if err := almacen.Guardar("debian-1", Credenciales{Usuario: "jorge", Password: "secreto"}); err != nil {
		t.Fatalf("Guardar() error = %v, want nil", err)
	}

	if _, err := almacen.db.ExecContext(context.Background(),
		`UPDATE credenciales SET password = 'invalido' WHERE vm = ?`, "debian-1"); err != nil {
		t.Fatalf("corromper la contraseña: %v", err)
	}

	credenciales, ok := almacen.Obtener("debian-1")
	if !ok || credenciales.Password != "" {
		t.Errorf("Obtener() = %+v, %v, want la cuenta sin contraseña", credenciales, ok)
	}
}

// TestCredencialesConBaseCerrada propaga los errores de SQLite.
func TestCredencialesConBaseCerrada(t *testing.T) {
	t.Parallel()

	almacen := conBaseCerrada(t)

	if _, ok := almacen.Obtener("debian-1"); ok {
		t.Error("Obtener() = true con la base cerrada")
	}

	if err := almacen.Guardar("debian-1", Credenciales{}); err == nil {
		t.Error("Guardar() error = nil, want el fallo de SQLite")
	}

	if err := almacen.Borrar("debian-1"); err == nil {
		t.Error("Borrar() error = nil, want el fallo de SQLite")
	}

	if activa := almacen.Activa(); activa != "" {
		t.Errorf("Activa() = %q, want vacío con la base cerrada", activa)
	}

	if err := almacen.GuardarActiva("debian-1"); err == nil {
		t.Error("GuardarActiva() error = nil, want el fallo de SQLite")
	}

	if err := asegurarDescripcionOperaciones(almacen.db); err == nil {
		t.Error("asegurarDescripcionOperaciones() error = nil, want el fallo de SQLite")
	}
}

// TestGuardarSinEntropia propaga el fallo al cifrar.
func TestGuardarSinEntropia(t *testing.T) {
	almacen, _, _ := nuevaBaseDePrueba(t)

	original := aleatorio

	t.Cleanup(func() { aleatorio = original })

	aleatorio = func([]byte) (int, error) { return 0, errors.New("sin entropía") }

	if err := almacen.Guardar("debian-1", Credenciales{Password: "secreto"}); err == nil {
		t.Error("Guardar() error = nil, want el fallo del nonce")
	}
}

// TestCifrarConClaveInvalida rechaza una clave de longitud incorrecta.
func TestCifrarConClaveInvalida(t *testing.T) {
	t.Parallel()

	almacen := &Almacen{clave: []byte("corta")}

	if _, err := almacen.cifrar("secreto"); err == nil {
		t.Error("cifrar() error = nil, want el fallo de la clave")
	}

	if _, err := almacen.descifrar(base64.StdEncoding.EncodeToString([]byte("123456789012"))); err == nil {
		t.Error("descifrar() error = nil, want el fallo de la clave")
	}
}

// TestCifrarConGCMFallido propaga el fallo al preparar GCM.
func TestCifrarConGCMFallido(t *testing.T) {
	original := nuevoGCM

	t.Cleanup(func() { nuevoGCM = original })

	nuevoGCM = func(cipher.Block) (cipher.AEAD, error) { return nil, errors.New("sin GCM") }

	almacen, _, _ := nuevaBaseDePrueba(t)

	if _, err := almacen.cifrar("secreto"); err == nil {
		t.Error("cifrar() error = nil, want el fallo de GCM")
	}

	if _, err := almacen.descifrar(base64.StdEncoding.EncodeToString(make([]byte, 20))); err == nil {
		t.Error("descifrar() error = nil, want el fallo de GCM")
	}
}

// TestDescifrarTextosInvalidos cubre los datos corruptos.
func TestDescifrarTextosInvalidos(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)

	casos := []string{
		"no-es-base64",
		base64.StdEncoding.EncodeToString([]byte("corto")),
		base64.StdEncoding.EncodeToString(make([]byte, 20)),
	}

	for _, texto := range casos {
		if _, err := almacen.descifrar(texto); err == nil {
			t.Errorf("descifrar(%q) error = nil, want un error", texto)
		}
	}
}

// TestExisteColumnaConBaseCerrada propaga el fallo de consulta.
func TestExisteColumnaConBaseCerrada(t *testing.T) {
	t.Parallel()

	almacen := conBaseCerrada(t)

	if _, err := existeColumna(almacen.db, "operaciones", "descripcion"); err == nil {
		t.Error("existeColumna() error = nil, want el fallo de SQLite")
	}
}

// TestNuevoAlmacenSinDirectorioDeClave falla al crear el directorio.
func TestNuevoAlmacenSinDirectorioDeClave(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("ejecutando como root: los permisos no impiden crear")
	}

	directorio := t.TempDir()
	if err := os.Chmod(directorio, 0o500); err != nil { // #nosec G302 -- permisos de la prueba.
		t.Fatalf("restringir el directorio: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chmod(directorio, 0o700); err != nil { // #nosec G302 -- permisos de la prueba.
			t.Logf("restaurar permisos: %v", err)
		}
	})

	rutaClave := filepath.Join(directorio, "sub", "clave")
	if _, err := NuevoAlmacen(filepath.Join(t.TempDir(), archivoBDPrueba), rutaClave); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo del directorio de la clave")
	}
}
