// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// archivoBDPrueba es el nombre de la base temporal.
const archivoBDPrueba = "vmcons.db"

// nuevaBaseDePrueba crea la base y la clave en un directorio temporal.
func nuevaBaseDePrueba(t *testing.T) (almacen *Almacen, rutaBD, rutaClave string) {
	t.Helper()

	directorio := t.TempDir()
	rutaBD = filepath.Join(directorio, archivoBDPrueba)
	rutaClave = filepath.Join(directorio, archivoClave)

	almacen, err := NuevoAlmacen(rutaBD, rutaClave)
	if err != nil {
		t.Fatalf(msgAlmacenOK, err)
	}

	t.Cleanup(func() {
		if err := almacen.Close(); err != nil {
			t.Errorf("cerrar almacen de prueba: %v", err)
		}
	})

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

	t.Cleanup(func() {
		if err := otro.Close(); err != nil {
			t.Errorf("cerrar segundo almacen: %v", err)
		}
	})

	credenciales, ok := otro.Obtener(nombreDebian)
	if !ok {
		t.Fatal("Obtener() no encontró las credenciales")
	}

	if credenciales.Usuario != usuarioRoot || credenciales.Password != passwordPrueba || !credenciales.Registrada {
		t.Errorf("credenciales = %+v, want root/secreto/registrada", credenciales)
	}

	usuario, password, ok := otro.Guest(nombreDebian)
	if !ok || usuario != usuarioRoot || password != passwordPrueba {
		t.Errorf("Guest() = %q, %q, %v, want root, secreto, true", usuario, password, ok)
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

	if info.Mode().Perm() != 0o600 && runtime.GOOS != "windows" {
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

	_, err := NuevoAlmacen(filepath.Join(directorio, archivoBDPrueba), rutaClave)
	if err == nil {
		t.Fatal("NuevoAlmacen() error = nil, want el error de clave inválida")
	}
}

// TestAlmacenMigraElHistorial comprueba que una base anterior recibe la
// columna descripcion y conserva sus operaciones.
func TestAlmacenMigraElHistorial(t *testing.T) {
	t.Parallel()

	directorio := t.TempDir()
	rutaBD := filepath.Join(directorio, archivoBDPrueba)
	rutaClave := filepath.Join(directorio, archivoClave)

	db, err := sql.Open("sqlite", rutaBD)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	if _, err := db.ExecContext(t.Context(), `CREATE TABLE operaciones (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		fecha_hora TEXT NOT NULL,
		vm TEXT NOT NULL,
		modulo TEXT NOT NULL,
		operacion TEXT NOT NULL,
		resultado TEXT NOT NULL,
		detalle TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatalf("crear la tabla anterior: %v", err)
	}

	if _, err := db.ExecContext(t.Context(), `INSERT INTO operaciones
		(fecha_hora, vm, modulo, operacion, resultado, detalle)
		VALUES ('2026-10-25T10:00:00Z', 'debian-1', 'paquetes',
		'POST /api/packages', 'ok', 'HTTP 201')`); err != nil {
		t.Fatalf("insertar la operación anterior: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("cerrar la base anterior: %v", err)
	}

	almacen, err := NuevoAlmacen(rutaBD, rutaClave)
	if err != nil {
		t.Fatalf(msgAlmacenOK, err)
	}

	t.Cleanup(func() {
		if err := almacen.Close(); err != nil {
			t.Errorf("cerrar almacen migrado: %v", err)
		}
	})

	historial := NuevoHistorial(almacen)

	operaciones, err := historial.Listar(context.Background(), &FiltroHistorial{})
	if err != nil {
		t.Fatalf(msgListarError, err)
	}

	if len(operaciones) != 1 {
		t.Fatalf("Listar() devolvió %d, want 1", len(operaciones))
	}

	if operaciones[0].Descripcion != "" {
		t.Errorf("Descripcion de la fila anterior = %q, want vacía", operaciones[0].Descripcion)
	}

	if err := historial.Registrar(context.Background(), &Operacion{
		VM: nombreDebian, Modulo: moduloPaquetes, Operacion: "POST /api/packages",
		Descripcion: descripcionNano, Resultado: "ok",
	}); err != nil {
		t.Fatalf("Registrar() error = %v", err)
	}

	operaciones, err = historial.Listar(context.Background(), &FiltroHistorial{})
	if err != nil {
		t.Fatalf(msgListarError, err)
	}

	if len(operaciones) != 2 || operaciones[0].Descripcion != descripcionNano {
		t.Errorf("operaciones = %+v, want la descripción nueva", operaciones)
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

	t.Cleanup(func() {
		if err := otro.Close(); err != nil {
			t.Errorf("cerrar segundo almacen: %v", err)
		}
	})

	if activa := otro.Activa(); activa != nombreDebian {
		t.Errorf("Activa() = %q, want %q", activa, nombreDebian)
	}
}

// TestAlmacenCloseSinBase comprueba que cerrar un almacén sin base de
// datos no falla, tanto con receptor nulo como con la conexión vacía.
func TestAlmacenCloseSinBase(t *testing.T) {
	t.Parallel()

	var nulo *Almacen

	if err := nulo.Close(); err != nil {
		t.Fatalf("Close() con receptor nulo error = %v, want nil", err)
	}

	vacio := &Almacen{}

	if err := vacio.Close(); err != nil {
		t.Fatalf("Close() sin conexión error = %v, want nil", err)
	}
}

// conectorCierre es un conector de prueba cuyo cierre falla.
type conectorCierre struct{}

// Connect devuelve una conexión de prueba.
func (conectorCierre) Connect(context.Context) (driver.Conn, error) {
	return conexionCierre{}, nil
}

// Driver devuelve el controlador de prueba.
func (conectorCierre) Driver() driver.Driver { return controladorCierre{} }

// Close devuelve el error configurado al cerrar la base.
func (conectorCierre) Close() error { return errCierreFallido }

// controladorCierre es el controlador del conector de prueba.
type controladorCierre struct{}

// Open no se usa porque el conector entrega la conexión directamente.
func (controladorCierre) Open(string) (driver.Conn, error) {
	return conexionCierre{}, nil
}

// conexionCierre es una conexión de prueba que no ejecuta sentencias.
type conexionCierre struct{}

// Prepare no está soportado por la conexión de prueba.
func (conexionCierre) Prepare(string) (driver.Stmt, error) {
	return nil, errCierreFallido
}

// Close devuelve el error configurado.
func (conexionCierre) Close() error { return nil }

// Begin no está soportado por la conexión de prueba.
func (conexionCierre) Begin() (driver.Tx, error) { return nil, errCierreFallido }

// errCierreFallido es el error que devuelve el conector de prueba.
var errCierreFallido = errors.New("cierre fallido")

// TestAlmacenCloseDevuelveError comprueba que Close propaga el fallo del
// cierre de la base de datos.
func TestAlmacenCloseDevuelveError(t *testing.T) {
	t.Parallel()

	almacen := &Almacen{db: sql.OpenDB(conectorCierre{})}

	if err := almacen.Close(); !errors.Is(err, errCierreFallido) {
		t.Fatalf("Close() error = %v, want %v", err, errCierreFallido)
	}
}
