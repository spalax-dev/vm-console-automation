// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"encoding/base64"
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

// TestNuevoAlmacenConDirectorioDeDatosImposible falla cuando el
// directorio de la base es en realidad un archivo.
func TestNuevoAlmacenConDirectorioDeDatosImposible(t *testing.T) {
	t.Parallel()

	rutaBD := filepath.Join(archivoConTexto(t), "vmcons.db")

	if _, err := NuevoAlmacen(rutaBD, filepath.Join(t.TempDir(), "clave")); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo del directorio")
	}
}

// TestNuevoAlmacenConClaveIlegible falla si la clave es un directorio.
func TestNuevoAlmacenConClaveIlegible(t *testing.T) {
	t.Parallel()

	directorio := t.TempDir()

	if _, err := NuevoAlmacen(filepath.Join(directorio, archivoBDPrueba), directorio); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo de lectura de la clave")
	}
}

// TestNuevoAlmacenNoPuedeCrearElDirectorioDeLaClave falla cuando el
// directorio padre de la clave es un archivo. No depende de permisos ni
// del usuario que ejecuta las pruebas.
func TestNuevoAlmacenNoPuedeCrearElDirectorioDeLaClave(t *testing.T) {
	t.Parallel()

	archivo := archivoConTexto(t)
	rutaClave := filepath.Join(archivo, "sub", "clave")

	if _, err := NuevoAlmacen(filepath.Join(t.TempDir(), archivoBDPrueba), rutaClave); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo al crear el directorio de la clave")
	}
}

// TestNuevoAlmacenConBaseQueEsDirectorio falla al crear el esquema si la
// ruta de la base apunta a un directorio.
func TestNuevoAlmacenConBaseQueEsDirectorio(t *testing.T) {
	t.Parallel()

	if _, err := NuevoAlmacen(t.TempDir(), filepath.Join(t.TempDir(), "clave")); err == nil {
		t.Error("NuevoAlmacen() error = nil, want el fallo al crear el esquema")
	}
}

// TestAlmacenGuardarActualizaCredenciales comprueba que guardar dos veces
// la misma máquina actualiza la fila en lugar de duplicarla.
func TestAlmacenGuardarActualizaCredenciales(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)

	primera := Credenciales{Usuario: "jorge", Password: "uno", Puerto: 22, Direccion: "10.0.0.1", Registrada: true}
	if err := almacen.Guardar(nombreDebian, primera); err != nil {
		t.Fatalf("Guardar() error = %v, want nil", err)
	}

	segunda := Credenciales{Usuario: "andrey", Password: "dos", Puerto: 2222, Registrada: false}
	if err := almacen.Guardar(nombreDebian, segunda); err != nil {
		t.Fatalf("Guardar() error = %v, want nil", err)
	}

	guardadas, ok := almacen.Obtener(nombreDebian)
	if !ok {
		t.Fatal("Obtener() no encontró las credenciales actualizadas")
	}

	if guardadas != segunda {
		t.Errorf("Obtener() = %+v, want %+v", guardadas, segunda)
	}

	var total int
	if err := almacen.db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM credenciales WHERE vm = ?`, nombreDebian).Scan(&total); err != nil {
		t.Fatalf("contar las credenciales: %v", err)
	}

	if total != 1 {
		t.Errorf("filas = %d, want 1", total)
	}
}

// TestObtenerConDescifradoFallido devuelve la cuenta sin contraseña.
func TestObtenerConDescifradoFallido(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)

	if err := almacen.Guardar(nombreDebian, Credenciales{Usuario: "jorge", Password: "secreto"}); err != nil {
		t.Fatalf("Guardar() error = %v, want nil", err)
	}

	if _, err := almacen.db.ExecContext(context.Background(),
		`UPDATE credenciales SET password = 'invalido' WHERE vm = ?`, nombreDebian); err != nil {
		t.Fatalf("corromper la contraseña: %v", err)
	}

	credenciales, ok := almacen.Obtener(nombreDebian)
	if !ok || credenciales.Password != "" {
		t.Errorf("Obtener() = %+v, %v, want la cuenta sin contraseña", credenciales, ok)
	}
}

// TestCredencialesConBaseCerrada propaga los errores de SQLite.
func TestCredencialesConBaseCerrada(t *testing.T) {
	t.Parallel()

	almacen := conBaseCerrada(t)

	if _, ok := almacen.Obtener(nombreDebian); ok {
		t.Error("Obtener() = true con la base cerrada")
	}

	if _, _, ok := almacen.Guest(nombreDebian); ok {
		t.Error("Guest() = true con la base cerrada")
	}

	if almacen.Registrada(nombreDebian) {
		t.Error("Registrada() = true con la base cerrada")
	}

	if err := almacen.Guardar(nombreDebian, Credenciales{}); err == nil {
		t.Error("Guardar() error = nil, want el fallo de SQLite")
	}

	if err := almacen.Borrar(nombreDebian); err == nil {
		t.Error("Borrar() error = nil, want el fallo de SQLite")
	}

	if activa := almacen.Activa(); activa != "" {
		t.Errorf("Activa() = %q, want vacío con la base cerrada", activa)
	}

	if err := almacen.GuardarActiva(nombreDebian); err == nil {
		t.Error("GuardarActiva() error = nil, want el fallo de SQLite")
	}

	if err := asegurarDescripcionOperaciones(almacen.db); err == nil {
		t.Error("asegurarDescripcionOperaciones() error = nil, want el fallo de SQLite")
	}
}

// TestDescifrarTextosInvalidos cubre los datos corruptos.
func TestDescifrarTextosInvalidos(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)

	casos := []struct {
		nombre string
		texto  string
	}{
		{"no es base64", "no-es-base64"},
		{"cifrado incompleto", base64.StdEncoding.EncodeToString([]byte("corto"))},
		{"sin autenticación", base64.StdEncoding.EncodeToString(make([]byte, 20))},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			if _, err := almacen.descifrar(caso.texto); err == nil {
				t.Errorf("descifrar(%q) error = nil, want un error", caso.texto)
			}
		})
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
