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

// salidaPasswd simula getent passwd con cuentas del sistema y regulares.
const salidaPasswd = "root:x:0:0:root:/root:/bin/bash\n" +
	"jorge:x:1000:1000:Jorge:/home/jorge:/bin/bash\n" +
	"systemd-network:x:998:998::/:/usr/sbin/nologin\n" +
	"andrey:x:1001:1001::/home/andrey:/bin/bash\n"

// salidaSudo simula getent group sudo.
const salidaSudo = "sudo:x:27:jorge\n"

// nuevoUsuariosDePrueba arma el servicio con sesión y ejecutor falsos.
func nuevoUsuariosDePrueba(t *testing.T, ejecutor *stubExecutor, sesion *stubSession, err error) (*Usuarios, *stubAbridor, string) {
	t.Helper()

	directorio := t.TempDir()
	abridor := &stubAbridor{sesion: sesion, err: err}

	return NuevoUsuarios(ConfigUsuarios{
		Sesiones:   abridor,
		Ejecutor:   ejecutor,
		Directorio: directorio,
	}), abridor, directorio
}

// TestUsuariosListarFiltraCuentasRegulares comprueba el listado y las
// marcas de sudo y llaves.
func TestUsuariosListarFiltraCuentasRegulares(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{results: []domain.Result{
		{Stdout: salidaPasswd},
		{Stdout: salidaSudo},
	}}

	usuarios, _, directorio := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	if err := os.WriteFile(filepath.Join(directorio, "andrey_ed25519"), []byte(contenidoPrivado), 0o600); err != nil {
		t.Fatalf("crear la llave de andrey: %v", err)
	}

	lista, err := usuarios.Listar(context.Background())
	if err != nil {
		t.Fatalf(msgListarOK, err)
	}

	if len(lista) != 2 {
		t.Fatalf("Listar() devolvió %d usuarios, want 2", len(lista))
	}

	if lista[0].Nombre != nombreJorge || !lista[0].Sudo || lista[0].UID != 1000 {
		t.Errorf("primer usuario = %+v, want jorge con sudo y UID 1000", lista[0])
	}

	if lista[1].Nombre != usuarioAndrey || !lista[1].Llaves {
		t.Errorf("segundo usuario = %+v, want andrey con llaves", lista[1])
	}
}

// TestUsuariosObtener comprueba la consulta de una cuenta.
func TestUsuariosObtener(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{results: []domain.Result{
		{Stdout: "jorge:x:1000:1000:Jorge:/home/jorge:/bin/bash\n"},
		{Stdout: salidaSudo},
	}}

	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	usuario, err := usuarios.Obtener(context.Background(), nombreJorge)
	if err != nil {
		t.Fatalf(msgObtenerOK, err)
	}

	if usuario.Nombre != nombreJorge || !usuario.Sudo || usuario.Home != "/home/jorge" {
		t.Errorf("usuario = %+v, want jorge con sudo y home", usuario)
	}
}

// TestUsuariosObtenerInexistente comprueba el error de una cuenta que no
// existe.
func TestUsuariosObtenerInexistente(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{results: []domain.Result{{Stdout: ""}}}

	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	if _, err := usuarios.Obtener(context.Background(), nombreJorge); !errors.Is(err, ErrUsuarioNoEncontrado) {
		t.Fatalf("Obtener() error = %v, want ErrUsuarioNoEncontrado", err)
	}
}

// TestUsuariosCrearConLlaves comprueba la creación con sudo y llaves.
func TestUsuariosCrearConLlaves(t *testing.T) {
	t.Parallel()

	ejecutor := &stubExecutor{}
	sesion := &stubSession{}

	usuarios, abridor, directorio := nuevoUsuariosDePrueba(t, ejecutor, sesion, nil)
	ejecutor.onExecute = func() {
		if err := os.WriteFile(filepath.Join(directorio, "jorge_ed25519.pub"), []byte(clavePublica+"\n"), 0o600); err != nil {
			t.Fatalf(msgEscribirPub, err)
		}
	}

	resultado, err := usuarios.Crear(context.Background(), nombreJorge, true, true)
	if err != nil {
		t.Fatalf("Crear() error = %v, want nil", err)
	}

	if abridor.abiertas != 1 {
		t.Errorf(msgAbrirUna, abridor.abiertas)
	}

	if len(sesion.commands) != 3 {
		t.Fatalf("la sesión ejecutó %d órdenes, want 3", len(sesion.commands))
	}

	creacion := sesion.commands[1].ShellLine()
	for _, esperado := range []string{binUseradd, "-aG sudo", binPasswd + " -l"} {
		if !strings.Contains(creacion, esperado) {
			t.Errorf("orden de creación = %q, want %q", creacion, esperado)
		}
	}

	instalacion := sesion.commands[2].ShellLine()
	if !strings.Contains(instalacion, "authorized_keys") || !strings.Contains(instalacion, clavePublica) {
		t.Errorf("orden de llaves = %q, want authorized_keys con la pública", instalacion)
	}

	if !sesion.closed {
		t.Error(msgSesionAbierta)
	}

	if resultado.Codigo != 0 {
		t.Errorf(msgCodigoCero, resultado.Codigo)
	}
}

// TestUsuariosCrearExistente comprueba el conflicto de una cuenta
// existente.
func TestUsuariosCrearExistente(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{results: []domain.Result{
		{Stdout: lineaJorge},
	}}

	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	if _, err := usuarios.Crear(context.Background(), nombreJorge, false, false); !errors.Is(err, ErrUsuarioExistente) {
		t.Fatalf("Crear() error = %v, want ErrUsuarioExistente", err)
	}

	if len(sesion.commands) != 1 {
		t.Errorf("la sesión ejecutó %d órdenes, want solo la consulta", len(sesion.commands))
	}
}

// TestUsuariosCrearSinLlaves comprueba que sin llaves no se genera el
// par ni se bloquea la contraseña.
func TestUsuariosCrearSinLlaves(t *testing.T) {
	t.Parallel()

	ejecutor := &stubExecutor{}
	sesion := &stubSession{}

	usuarios, _, _ := nuevoUsuariosDePrueba(t, ejecutor, sesion, nil)

	if _, err := usuarios.Crear(context.Background(), usuarioAndrey, false, false); err != nil {
		t.Fatalf("Crear() error = %v, want nil", err)
	}

	if len(ejecutor.calls) != 0 {
		t.Error("ssh-keygen se ejecutó sin llaves")
	}

	if len(sesion.commands) != 2 {
		t.Fatalf(msgOrdenDos, len(sesion.commands))
	}

	if orden := sesion.commands[1].ShellLine(); strings.Contains(orden, "passwd -l") {
		t.Errorf("orden = %q, no debe bloquear la contraseña", orden)
	}
}

// TestUsuariosEliminarConArchivos comprueba userdel -r y el borrado de
// las llaves del anfitrión.
func TestUsuariosEliminarConArchivos(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{results: []domain.Result{
		{Stdout: lineaJorge},
		{},
	}}

	usuarios, _, directorio := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)
	ruta := filepath.Join(directorio, archivoJorge)

	for _, archivo := range []string{ruta, ruta + ".pub"} {
		if err := os.WriteFile(archivo, []byte(contenidoPrivado), 0o600); err != nil {
			t.Fatalf("crear %s: %v", archivo, err)
		}
	}

	if _, err := usuarios.Eliminar(context.Background(), nombreJorge, true); err != nil {
		t.Fatalf(msgEliminarOK, err)
	}

	if orden := sesion.commands[1].ShellLine(); !strings.Contains(orden, "userdel -r") {
		t.Errorf("orden = %q, want userdel -r", orden)
	}

	for _, archivo := range []string{ruta, ruta + ".pub"} {
		if _, err := os.Stat(archivo); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("la llave %s no se borró", archivo)
		}
	}
}

// TestUsuariosEliminarInexistente comprueba el error de una cuenta que
// no existe.
func TestUsuariosEliminarInexistente(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{results: []domain.Result{{Stdout: ""}}}

	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	if _, err := usuarios.Eliminar(context.Background(), nombreJorge, false); !errors.Is(err, ErrUsuarioNoEncontrado) {
		t.Fatalf("Eliminar() error = %v, want ErrUsuarioNoEncontrado", err)
	}
}

// TestUsuariosDescargarLlave devuelve la llave privada del usuario.
func TestUsuariosDescargarLlave(t *testing.T) {
	t.Parallel()

	usuarios, _, directorio := nuevoUsuariosDePrueba(t, &stubExecutor{}, &stubSession{}, nil)
	ruta := filepath.Join(directorio, archivoJorge)

	if err := os.WriteFile(ruta, []byte(contenidoPrivado), 0o600); err != nil {
		t.Fatalf("crear la llave: %v", err)
	}

	nombre, contenido, err := usuarios.DescargarLlave(nombreJorge)
	if err != nil {
		t.Fatalf("DescargarLlave() error = %v, want nil", err)
	}

	if nombre != archivoJorge || string(contenido) != contenidoPrivado {
		t.Errorf("llave = %q %q, want el contenido de jorge", nombre, string(contenido))
	}
}

// TestUsuariosDescargarLlaveInexistente devuelve
// ErrLlaveUsuarioNoEncontrada.
func TestUsuariosDescargarLlaveInexistente(t *testing.T) {
	t.Parallel()

	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, &stubSession{}, nil)

	if _, _, err := usuarios.DescargarLlave(nombreJorge); !errors.Is(err, ErrLlaveUsuarioNoEncontrada) {
		t.Fatalf("DescargarLlave() error = %v, want ErrLlaveUsuarioNoEncontrada", err)
	}
}

// TestUsuariosCrearErrorDeOrden comprueba el fallo de useradd con su
// detalle técnico.
func TestUsuariosCrearErrorDeOrden(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{
		results: []domain.Result{{Stdout: ""}},
		result:  domain.Result{Stderr: "useradd: cannot lock /etc/passwd", ExitCode: 1},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: "useradd"}},
	}

	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	resultado, err := usuarios.Crear(context.Background(), nombreJorge, false, false)
	if err == nil {
		t.Fatal("Crear() error = nil, want el error de useradd")
	}

	if resultado.Codigo != 1 || resultado.Stderr == "" {
		t.Errorf(msgResultadoTecnico, resultado)
	}
}

// TestUsuariosListarSinSesion comprueba que se propaga la ausencia de
// sesión.
func TestUsuariosListarSinSesion(t *testing.T) {
	t.Parallel()

	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, nil, ErrSesionNoDisponible)

	if _, err := usuarios.Listar(context.Background()); !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf(msgListarSinSesion, err)
	}
}

// TestUsuariosListarError comprueba el fallo al listar las cuentas.
func TestUsuariosListarError(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{err: &domain.ExitError{Code: 1, Command: binGetent}}
	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	if _, err := usuarios.Listar(context.Background()); err == nil {
		t.Fatal("Listar() error = nil, want el error de getent")
	}
}

// TestUsuariosObtenerError comprueba un fallo inesperado al consultar.
func TestUsuariosObtenerError(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{errs: []error{&domain.ExitError{Code: 1, Command: binGetent}}}
	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	if _, err := usuarios.Obtener(context.Background(), nombreJorge); err == nil {
		t.Fatal("Obtener() error = nil, want el error de getent")
	}
}

// TestUsuariosObtenerGetentSinCuenta comprueba el código 2 de getent.
func TestUsuariosObtenerGetentSinCuenta(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{errs: []error{&domain.ExitError{Code: 2, Command: binGetent}}}
	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	if _, err := usuarios.Obtener(context.Background(), nombreJorge); !errors.Is(err, ErrUsuarioNoEncontrado) {
		t.Fatalf("Obtener() error = %v, want ErrUsuarioNoEncontrado", err)
	}
}

// TestUsuariosCrearSinSesion comprueba la ausencia de sesión al crear.
func TestUsuariosCrearSinSesion(t *testing.T) {
	t.Parallel()

	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, nil, ErrSesionNoDisponible)

	if _, err := usuarios.Crear(context.Background(), nombreJorge, false, false); !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf("Crear() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestUsuariosCrearSinPublica comprueba el fallo al leer la llave
// pública generada.
func TestUsuariosCrearSinPublica(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{}
	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	if _, err := usuarios.Crear(context.Background(), nombreJorge, false, true); err == nil {
		t.Fatal("Crear() error = nil, want el error de la llave pública")
	}
}

// TestUsuariosCrearErrorDeInstalacion comprueba el fallo al instalar la
// llave pública en la máquina.
func TestUsuariosCrearErrorDeInstalacion(t *testing.T) {
	t.Parallel()

	ejecutor := &stubExecutor{}
	sesion := &stubSession{errs: []error{nil, nil, &domain.ExitError{Code: 1, Command: binSh}}}

	usuarios, _, directorio := nuevoUsuariosDePrueba(t, ejecutor, sesion, nil)
	ejecutor.onExecute = func() {
		if err := os.WriteFile(filepath.Join(directorio, nombreJorge+"_ed25519.pub"), []byte(clavePublica+"\n"), 0o600); err != nil {
			t.Fatalf(msgEscribirPub, err)
		}
	}

	if _, err := usuarios.Crear(context.Background(), nombreJorge, false, true); err == nil {
		t.Fatal("Crear() error = nil, want el error de instalación")
	}
}

// TestUsuariosEliminarErrorDeOrden comprueba el fallo de userdel.
func TestUsuariosEliminarErrorDeOrden(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{
		results: []domain.Result{{Stdout: lineaJorge}},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: binUserdel}},
	}

	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	if _, err := usuarios.Eliminar(context.Background(), nombreJorge, false); err == nil {
		t.Fatal("Eliminar() error = nil, want el error de userdel")
	}
}

// TestUsuariosDescargarLlaveError comprueba un fallo de lectura de la
// llave.
func TestUsuariosDescargarLlaveError(t *testing.T) {
	t.Parallel()

	usuarios, _, directorio := nuevoUsuariosDePrueba(t, &stubExecutor{}, &stubSession{}, nil)

	if err := os.Mkdir(filepath.Join(directorio, nombreJorge+"_ed25519"), 0o700); err != nil {
		t.Fatalf(msgCrearDir, err)
	}

	if _, _, err := usuarios.DescargarLlave(nombreJorge); err == nil {
		t.Fatal("DescargarLlave() error = nil, want el error de lectura")
	}
}

// TestParsearUsuarioInvalido comprueba una línea de passwd incompleta.
func TestParsearUsuarioInvalido(t *testing.T) {
	t.Parallel()

	if usuario := parsearUsuario("invalido"); usuario.Nombre != "" {
		t.Errorf("parsearUsuario() = %+v, want un usuario vacío", usuario)
	}
}

// TestUsuariosListarErrorDeGrupo comprueba el fallo al consultar sudo.
func TestUsuariosListarErrorDeGrupo(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{
		results: []domain.Result{{Stdout: salidaPasswd}},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: binGetent}},
	}
	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	if _, err := usuarios.Listar(context.Background()); err == nil {
		t.Fatal("Listar() error = nil, want el error del grupo sudo")
	}
}

// TestUsuariosObtenerErrorDeGrupo comprueba el fallo al consultar sudo
// al obtener una cuenta.
func TestUsuariosObtenerErrorDeGrupo(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{
		results: []domain.Result{{Stdout: lineaJorge}},
		errs:    []error{nil, &domain.ExitError{Code: 1, Command: binGetent}},
	}
	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	if _, err := usuarios.Obtener(context.Background(), nombreJorge); err == nil {
		t.Fatal("Obtener() error = nil, want el error del grupo sudo")
	}
}

// TestUsuariosCrearErrorDeDirectorio comprueba el fallo al crear el
// directorio de llaves.
func TestUsuariosCrearErrorDeDirectorio(t *testing.T) {
	t.Parallel()

	archivo := filepath.Join(t.TempDir(), "archivo")
	if err := os.WriteFile(archivo, []byte("x"), 0o600); err != nil {
		t.Fatalf(msgPrepararArchivo, err)
	}

	usuarios := NuevoUsuarios(ConfigUsuarios{
		Sesiones:   &stubAbridor{sesion: &stubSession{}},
		Ejecutor:   &stubExecutor{},
		Directorio: archivo,
	})

	if _, err := usuarios.Crear(context.Background(), nombreJorge, false, true); err == nil {
		t.Fatal("Crear() error = nil, want el error del directorio")
	}
}

// TestUsuariosListarLimitesDeUID comprueba los extremos del rango de
// cuentas regulares (1000–65533) y descarta las líneas mal formadas o
// con UTF-8 en el nombre.
func TestUsuariosListarLimitesDeUID(t *testing.T) {
	t.Parallel()

	passwd := "root:x:0:0:root:/root:/bin/bash\n" +
		"justo-antes:x:999:999::/home/justo-antes:/bin/bash\n" +
		"borde-inferior:x:1000:1000::/home/borde-inferior:/bin/bash\n" +
		"borde-superior:x:65533:65533::/home/borde-superior:/bin/bash\n" +
		"nadie:x:65534:65534::/nonexistent:/usr/sbin/nologin\n" +
		"fuera-de-rango:x:65535:65535::/home/fuera-de-rango:/bin/bash\n" +
		"sin-numero:x:abc:abc::/home/sin-numero:/bin/bash\n" +
		"incompleta:x:1001:1001\n" +
		"josé:x:1002:1002::/home/josé:/bin/bash\n"

	sesion := &stubSession{results: []domain.Result{
		{Stdout: passwd},
		{Stdout: ""},
	}}

	usuarios, _, _ := nuevoUsuariosDePrueba(t, &stubExecutor{}, sesion, nil)

	lista, err := usuarios.Listar(context.Background())
	if err != nil {
		t.Fatalf(msgListarOK, err)
	}

	esperados := []struct {
		nombre string
		uid    int
	}{
		{"borde-inferior", 1000},
		{"borde-superior", 65533},
		{"josé", 1002},
	}

	if len(lista) != len(esperados) {
		t.Fatalf("Listar() devolvió %d usuarios (%+v), want %d", len(lista), lista, len(esperados))
	}

	for i, esperado := range esperados {
		if lista[i].Nombre != esperado.nombre || lista[i].UID != esperado.uid {
			t.Errorf("usuario[%d] = %+v, want %q con UID %d", i, lista[i], esperado.nombre, esperado.uid)
		}
	}
}
