// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"consola-vm/internal/core/domain"
)

const (
	// clavePublica simula la llave pública que genera ssh-keygen.
	clavePublica = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI ejemplo vmcons-root"
	// nombreLlave es el nombre por omisión de la llave.
	nombreLlave = "id_ed25519"
	// contenidoPrivado simula el contenido de la llave privada.
	contenidoPrivado = "-----BEGIN OPENSSH PRIVATE KEY-----\nfalso\n-----END OPENSSH PRIVATE KEY-----\n"
)

// stubExecutor registra las órdenes del anfitrión y permite simular sus
// efectos secundarios con onExecute.
type stubExecutor struct {
	calls     [][]string
	result    domain.Result
	err       error
	onExecute func()
}

// Execute registra la orden y devuelve la respuesta preprogramada.
func (e *stubExecutor) Execute(_ context.Context, program string, args ...string) (domain.Result, error) {
	e.calls = append(e.calls, append([]string{program}, args...))

	if e.onExecute != nil {
		e.onExecute()
	}

	return e.result, e.err
}

// nuevoLlavesDePrueba arma el servicio con un ejecutor y una sesión
// falsos sobre una ruta temporal.
func nuevoLlavesDePrueba(t *testing.T, ejecutor *stubExecutor, sesion *stubSession, abrirErr error) (*Llaves, *stubAbridor, string) {
	t.Helper()

	ruta := filepath.Join(t.TempDir(), nombreLlave)
	abridor := &stubAbridor{sesion: sesion, err: abrirErr}

	return NuevoLlaves(ConfigLlaves{
		Sesiones: abridor,
		Ejecutor: ejecutor,
		Ruta:     ruta,
	}), abridor, ruta
}

// TestLlavesGenerarInstalaClave comprueba la generación, instalación y
// configuración de sshd.
func TestLlavesGenerarInstalaClave(t *testing.T) {
	t.Parallel()

	ejecutor := &stubExecutor{}
	sesion := &stubSession{}

	llaves, abridor, ruta := nuevoLlavesDePrueba(t, ejecutor, sesion, nil)
	ejecutor.onExecute = func() {
		if err := os.WriteFile(ruta, []byte(contenidoPrivado), 0o600); err != nil {
			t.Fatalf("escribir la llave privada: %v", err)
		}

		if err := os.WriteFile(ruta+".pub", []byte(clavePublica+"\n"), 0o600); err != nil {
			t.Fatalf(msgEscribirPub, err)
		}
	}

	resultado, err := llaves.Generar(context.Background())
	if err != nil {
		t.Fatalf("Generar() error = %v, want nil", err)
	}

	if abridor.abiertas != 1 {
		t.Errorf(msgAbrirUna, abridor.abiertas)
	}

	if len(ejecutor.calls) != 1 {
		t.Fatalf("ssh-keygen se ejecutó %d veces, want 1", len(ejecutor.calls))
	}

	llamada := ejecutor.calls[0]
	if llamada[0] != "ssh-keygen" {
		t.Errorf("programa = %q, want ssh-keygen", llamada[0])
	}

	if !contiene(llamada, "-t") || !contiene(llamada, "ed25519") || !contiene(llamada, ruta) {
		t.Errorf("argumentos = %v, want tipo ed25519 y ruta %q", llamada, ruta)
	}

	if len(sesion.commands) != 1 {
		t.Fatalf(msgOrdenUnica, len(sesion.commands))
	}

	orden := sesion.commands[0].ShellLine()
	if !strings.Contains(orden, "authorized_keys") || !strings.Contains(orden, "AAAAC3NzaC1lZDI1NTE5") {
		t.Errorf("orden de authorized_keys = %q", orden)
	}

	if !strings.Contains(orden, "sshd_config.d/99-vmcons.conf") ||
		!strings.Contains(orden, "PasswordAuthentication no") {
		t.Errorf("orden de sshd = %q", orden)
	}

	if !strings.Contains(orden, "openssh-server") || !strings.Contains(orden, "enable --now ssh") {
		t.Errorf("orden = %q, want instalar y arrancar SSH", orden)
	}

	if !sesion.closed {
		t.Error(msgSesionAbierta)
	}

	if resultado.Codigo != 0 {
		t.Errorf(msgCodigoCero, resultado.Codigo)
	}

	if resultado.Mensaje == "" {
		t.Error("Mensaje vacío, want el mensaje de éxito")
	}
}

// contiene informa si la lista incluye el valor.
func contiene(valores []string, buscado string) bool {
	for _, valor := range valores {
		if valor == buscado {
			return true
		}
	}

	return false
}

// TestLlavesGenerarExistente comprueba que no se sobrescribe una llave
// existente.
func TestLlavesGenerarExistente(t *testing.T) {
	t.Parallel()

	ejecutor := &stubExecutor{}
	llaves, abridor, ruta := nuevoLlavesDePrueba(t, ejecutor, &stubSession{}, nil)

	if err := os.WriteFile(ruta, []byte(contenidoPrivado), 0o600); err != nil {
		t.Fatalf("escribir la llave existente: %v", err)
	}

	if _, err := llaves.Generar(context.Background()); !errors.Is(err, ErrLlaveExistente) {
		t.Fatalf("Generar() error = %v, want ErrLlaveExistente", err)
	}

	if len(ejecutor.calls) != 0 {
		t.Error("ssh-keygen se ejecutó con una llave existente")
	}

	if abridor.abiertas != 0 {
		t.Error("se abrió una sesión con una llave existente")
	}
}

// TestLlavesGenerarErrorDeKeygen comprueba que el fallo de ssh-keygen
// se propaga.
func TestLlavesGenerarErrorDeKeygen(t *testing.T) {
	t.Parallel()

	ejecutor := &stubExecutor{err: &domain.ExitError{Code: 1, Command: "ssh-keygen"}}
	sesion := &stubSession{}

	llaves, _, _ := nuevoLlavesDePrueba(t, ejecutor, sesion, nil)

	if _, err := llaves.Generar(context.Background()); err == nil {
		t.Fatal("Generar() error = nil, want el error de ssh-keygen")
	}

	if !sesion.closed {
		t.Error(msgSesionAbierta)
	}
}

// TestLlavesGenerarSinSesion comprueba que se propaga la ausencia de
// sesión.
func TestLlavesGenerarSinSesion(t *testing.T) {
	t.Parallel()

	llaves, _, _ := nuevoLlavesDePrueba(t, &stubExecutor{}, nil, ErrSesionNoDisponible)

	if _, err := llaves.Generar(context.Background()); !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf("Generar() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestLlavesGenerarErrorDeInstalacion comprueba que el fallo al instalar
// la clave conserva el detalle técnico.
func TestLlavesGenerarErrorDeInstalacion(t *testing.T) {
	t.Parallel()

	ejecutor := &stubExecutor{}
	sesion := &stubSession{
		result: domain.Result{Stderr: "Permission denied", ExitCode: 1},
		err:    &domain.ExitError{Code: 1, Command: "sh"},
	}

	llaves, _, ruta := nuevoLlavesDePrueba(t, ejecutor, sesion, nil)
	ejecutor.onExecute = func() {
		if err := os.WriteFile(ruta+".pub", []byte(clavePublica+"\n"), 0o600); err != nil {
			t.Fatalf(msgEscribirPub, err)
		}
	}

	resultado, err := llaves.Generar(context.Background())
	if err == nil {
		t.Fatal("Generar() error = nil, want el error de instalación")
	}

	if resultado.Codigo != 1 {
		t.Errorf("Codigo = %d, want 1", resultado.Codigo)
	}

	if resultado.Stderr != "Permission denied" {
		t.Errorf(msgStderrSesion, resultado.Stderr)
	}
}

// TestLlavesDescargar devuelve el contenido de la llave privada.
func TestLlavesDescargar(t *testing.T) {
	t.Parallel()

	llaves, _, ruta := nuevoLlavesDePrueba(t, &stubExecutor{}, &stubSession{}, nil)

	if err := os.WriteFile(ruta, []byte(contenidoPrivado), 0o600); err != nil {
		t.Fatalf("escribir la llave: %v", err)
	}

	nombre, contenido, err := llaves.Descargar()
	if err != nil {
		t.Fatalf("Descargar() error = %v, want nil", err)
	}

	if nombre != nombreLlave {
		t.Errorf("nombre = %q, want id_ed25519", nombre)
	}

	if string(contenido) != contenidoPrivado {
		t.Errorf("contenido = %q, want %q", string(contenido), contenidoPrivado)
	}
}

// TestLlavesDescargarInexistente devuelve ErrLlaveNoEncontrada.
func TestLlavesDescargarInexistente(t *testing.T) {
	t.Parallel()

	llaves, _, _ := nuevoLlavesDePrueba(t, &stubExecutor{}, &stubSession{}, nil)

	if _, _, err := llaves.Descargar(); !errors.Is(err, ErrLlaveNoEncontrada) {
		t.Fatalf("Descargar() error = %v, want ErrLlaveNoEncontrada", err)
	}
}

// TestLlavesGenerarStatError comprueba el fallo al comprobar la ruta de
// la llave.
func TestLlavesGenerarStatError(t *testing.T) {
	t.Parallel()

	archivo := filepath.Join(t.TempDir(), "archivo")
	if err := os.WriteFile(archivo, []byte("x"), 0o600); err != nil {
		t.Fatalf(msgPrepararArchivo, err)
	}

	llaves := NuevoLlaves(ConfigLlaves{
		Sesiones: &stubAbridor{},
		Ejecutor: &stubExecutor{},
		Ruta:     filepath.Join(archivo, "id_ed25519"),
	})

	if _, err := llaves.Generar(context.Background()); err == nil {
		t.Fatal("Generar() error = nil, want el error al comprobar la ruta")
	}
}

// TestLlavesGenerarErrorDeDirectorio comprueba el fallo al crear el
// directorio de llaves.
func TestLlavesGenerarErrorDeDirectorio(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("como root no se puede provocar un error de permisos")
	}

	directorio := filepath.Join(t.TempDir(), "solo-lectura")
	if err := os.Mkdir(directorio, 0o500); err != nil {
		t.Fatalf(msgCrearDir, err)
	}

	t.Cleanup(func() {
		// #nosec G302 -- es un directorio, no un archivo.
		if err := os.Chmod(directorio, 0o700); err != nil {
			t.Logf("restaurar permisos: %v", err)
		}
	})

	llaves := NuevoLlaves(ConfigLlaves{
		Sesiones: &stubAbridor{},
		Ejecutor: &stubExecutor{},
		Ruta:     filepath.Join(directorio, "sub", "id_ed25519"),
	})

	if _, err := llaves.Generar(context.Background()); err == nil {
		t.Fatal("Generar() error = nil, want el error al crear el directorio")
	}
}

// TestLlavesGenerarSinPublica comprueba el fallo al leer la llave
// pública.
func TestLlavesGenerarSinPublica(t *testing.T) {
	t.Parallel()

	llaves, _, _ := nuevoLlavesDePrueba(t, &stubExecutor{}, &stubSession{}, nil)

	if _, err := llaves.Generar(context.Background()); err == nil {
		t.Fatal("Generar() error = nil, want el error al leer la llave pública")
	}
}

// TestLlavesDescargarErrorDeLectura comprueba el fallo de lectura de la
// llave privada.
func TestLlavesDescargarErrorDeLectura(t *testing.T) {
	t.Parallel()

	llaves, _, ruta := nuevoLlavesDePrueba(t, &stubExecutor{}, &stubSession{}, nil)

	if err := os.Mkdir(ruta, 0o700); err != nil {
		t.Fatalf(msgCrearDir, err)
	}

	if _, _, err := llaves.Descargar(); err == nil {
		t.Fatal("Descargar() error = nil, want el error de lectura")
	}
}
