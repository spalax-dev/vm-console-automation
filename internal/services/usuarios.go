// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// ErrUsuarioExistente indica que la cuenta ya está en la máquina.
var ErrUsuarioExistente = errors.New("el usuario ya existe")

// ErrUsuarioNoEncontrado indica que la cuenta no está en la máquina.
var ErrUsuarioNoEncontrado = errors.New("el usuario no existe")

// ErrLlaveUsuarioNoEncontrada indica que no se ha generado la llave del
// usuario.
var ErrLlaveUsuarioNoEncontrada = errors.New("la llave del usuario no existe")

// ConfigUsuarios reúne las dependencias del servicio de usuarios.
type ConfigUsuarios struct {
	// Sesiones abre la sesión con la máquina.
	Sesiones AbridorSesion
	// Ejecutor lanza ssh-keygen en el anfitrión.
	Ejecutor ports.Executor
	// Directorio guarda las llaves privadas de los usuarios.
	Directorio string
}

// Usuario describe una cuenta local de la máquina.
type Usuario struct {
	// Nombre es el nombre de la cuenta.
	Nombre string
	// Sudo indica si pertenece al grupo sudo.
	Sudo bool
	// Llaves indica si tiene un par de llaves generado.
	Llaves bool
	// UID es el identificador numérico.
	UID int
	// Home es el directorio personal.
	Home string
}

// Usuarios implementa el caso de uso de gestión de cuentas locales.
type Usuarios struct {
	sesiones   AbridorSesion
	ejecutor   ports.Executor
	directorio string
}

// NuevoUsuarios construye el servicio de usuarios con las dependencias
// dadas.
func NuevoUsuarios(config ConfigUsuarios) *Usuarios {
	return &Usuarios{
		sesiones:   config.Sesiones,
		ejecutor:   config.Ejecutor,
		directorio: config.Directorio,
	}
}

// Listar devuelve las cuentas regulares (UID entre 1000 y 65533).
func (u *Usuarios) Listar(ctx context.Context) ([]Usuario, error) {
	sesion, err := u.sesiones.Abrir(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrir la sesión para listar usuarios: %w", err)
	}

	defer cerrarSesion(sesion)

	passwd, err := ejecutarSalida(ctx, sesion, binGetent, "passwd")
	if err != nil {
		return nil, fmt.Errorf("listar los usuarios: %w", err)
	}

	grupo, err := ejecutarSalida(ctx, sesion, binGetent, "group", "sudo")
	if err != nil {
		return nil, fmt.Errorf("consultar el grupo sudo: %w", err)
	}

	miembros := miembrosGrupo(grupo)
	usuarios := []Usuario{}

	for _, linea := range strings.Split(passwd, "\n") {
		campos := strings.Split(linea, ":")
		if len(campos) < 7 {
			continue
		}

		uid, err := strconv.Atoi(campos[2])
		if err != nil || uid < 1000 || uid >= 65534 {
			continue
		}

		usuarios = append(usuarios, Usuario{
			Nombre: campos[0],
			Sudo:   miembros[campos[0]],
			Llaves: u.existeLlave(campos[0]),
			UID:    uid,
			Home:   campos[5],
		})
	}

	return usuarios, nil
}

// Obtener devuelve una cuenta de la máquina.
func (u *Usuarios) Obtener(ctx context.Context, nombre string) (Usuario, error) {
	sesion, err := u.sesiones.Abrir(ctx)
	if err != nil {
		return Usuario{}, fmt.Errorf("abrir la sesión para consultar al usuario: %w", err)
	}

	defer cerrarSesion(sesion)

	passwd, err := buscarUsuario(ctx, sesion, nombre)
	if err != nil {
		return Usuario{}, err
	}

	if passwd == "" {
		return Usuario{}, ErrUsuarioNoEncontrado
	}

	grupo, err := ejecutarSalida(ctx, sesion, binGetent, "group", "sudo")
	if err != nil {
		return Usuario{}, fmt.Errorf("consultar el grupo sudo: %w", err)
	}

	usuario := parsearUsuario(passwd)
	usuario.Sudo = miembrosGrupo(grupo)[nombre]
	usuario.Llaves = u.existeLlave(nombre)

	return usuario, nil
}

// Crear crea una cuenta, la agrega a sudo si se pide y genera sus llaves
// cuando corresponde.
func (u *Usuarios) Crear(ctx context.Context, nombre string, sudo, llaves bool) (Resultado, error) {
	sesion, err := u.sesiones.Abrir(ctx)
	if err != nil {
		return Resultado{}, fmt.Errorf("abrir la sesión para crear el usuario: %w", err)
	}

	defer cerrarSesion(sesion)

	existente, err := buscarUsuario(ctx, sesion, nombre)
	if err != nil {
		return Resultado{}, err
	}

	if existente != "" {
		return Resultado{}, ErrUsuarioExistente
	}

	guion := binUseradd + " -m -s /bin/bash " + domain.Quote(nombre)

	if sudo {
		guion += " && " + binUsermod + " -aG sudo " + domain.Quote(nombre)
	}

	if llaves {
		guion += " && " + binPasswd + " -l " + domain.Quote(nombre)
	}

	cmd := &domain.Command{Path: binSh, Args: []string{"-c", guion}}

	resultado, err := sesion.Run(ctx, cmd)

	salida := Resultado{
		Mensaje:   "Usuario «" + nombre + "» creado.",
		Operacion: cmd.ShellLine(),
		Salida:    resultado.Stdout,
		Stderr:    resultado.Stderr,
		Codigo:    resultado.ExitCode,
	}

	if err != nil {
		return salida, fmt.Errorf("crear el usuario %q: %w", nombre, err)
	}

	if llaves {
		if err := u.instalarLlaves(ctx, sesion, nombre); err != nil {
			return salida, err
		}
	}

	return salida, nil
}

// instalarLlaves genera el par del usuario, instala la pública y ajusta
// los permisos.
func (u *Usuarios) instalarLlaves(ctx context.Context, sesion ports.Session, nombre string) error {
	ruta := u.rutaLlave(nombre)

	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return fmt.Errorf("crear el directorio de llaves: %w", err)
	}

	if err := generarPar(ctx, u.ejecutor, ruta, "vmcons-"+nombre); err != nil {
		return err
	}

	clave, err := leerClavePublica(ruta)
	if err != nil {
		return err
	}

	home := "/home/" + nombre + "/.ssh"
	guion := "install -d -m 700 -o " + domain.Quote(nombre) + " -g " + domain.Quote(nombre) +
		" " + domain.Quote(home) +
		" && printf '%s\\n' " + domain.Quote(clave) + " >> " + domain.Quote(home+"/authorized_keys") +
		" && chown " + domain.Quote(nombre+":"+nombre) + " " + domain.Quote(home+"/authorized_keys") +
		" && chmod 600 " + domain.Quote(home+"/authorized_keys")

	cmd := &domain.Command{Path: binSh, Args: []string{"-c", guion}}

	if _, err := sesion.Run(ctx, cmd); err != nil {
		return fmt.Errorf("instalar las llaves del usuario %q: %w", nombre, err)
	}

	return nil
}

// Eliminar elimina una cuenta y, opcionalmente, sus archivos.
func (u *Usuarios) Eliminar(ctx context.Context, nombre string, archivos bool) (Resultado, error) {
	sesion, err := u.sesiones.Abrir(ctx)
	if err != nil {
		return Resultado{}, fmt.Errorf("abrir la sesión para eliminar al usuario: %w", err)
	}

	defer cerrarSesion(sesion)

	existente, err := buscarUsuario(ctx, sesion, nombre)
	if err != nil {
		return Resultado{}, err
	}

	if existente == "" {
		return Resultado{}, ErrUsuarioNoEncontrado
	}

	guion := binUserdel + " "

	if archivos {
		guion += "-r "
	}

	guion += domain.Quote(nombre)

	cmd := &domain.Command{Path: binSh, Args: []string{"-c", guion}}

	resultado, err := sesion.Run(ctx, cmd)

	salida := Resultado{
		Mensaje:   "Usuario «" + nombre + "» eliminado.",
		Operacion: cmd.ShellLine(),
		Salida:    resultado.Stdout,
		Stderr:    resultado.Stderr,
		Codigo:    resultado.ExitCode,
	}

	if err != nil {
		return salida, fmt.Errorf("eliminar el usuario %q: %w", nombre, err)
	}

	u.borrarLlaves(nombre)

	return salida, nil
}

// DescargarLlave devuelve el nombre y el contenido de la llave privada
// del usuario.
func (u *Usuarios) DescargarLlave(nombre string) (archivo string, contenido []byte, err error) {
	ruta := u.rutaLlave(nombre)

	// #nosec G304 -- la ruta se compone con un nombre de usuario validado.
	contenido, err = os.ReadFile(ruta)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil, ErrLlaveUsuarioNoEncontrada
	}

	if err != nil {
		return "", nil, fmt.Errorf("leer la llave del usuario: %w", err)
	}

	return filepath.Base(ruta), contenido, nil
}

// rutaLlave devuelve la ruta de la llave privada del usuario.
func (u *Usuarios) rutaLlave(nombre string) string {
	return filepath.Join(u.directorio, nombre+"_ed25519")
}

// existeLlave informa si la llave privada del usuario está en el
// anfitrión.
func (u *Usuarios) existeLlave(nombre string) bool {
	_, err := os.Stat(u.rutaLlave(nombre))

	return err == nil
}

// borrarLlaves elimina del anfitrión el par de llaves del usuario.
func (u *Usuarios) borrarLlaves(nombre string) {
	ruta := u.rutaLlave(nombre)

	for _, archivo := range []string{ruta, ruta + ".pub"} {
		if err := os.Remove(archivo); err != nil && !errors.Is(err, fs.ErrNotExist) {
			continue
		}
	}
}

// buscarUsuario devuelve la línea de getent passwd o vacío si no existe.
func buscarUsuario(ctx context.Context, sesion ports.Session, nombre string) (string, error) {
	resultado, err := sesion.Run(ctx, &domain.Command{
		Path: binGetent,
		Args: []string{"passwd", nombre},
	})
	if err != nil {
		var exitErr *domain.ExitError
		if errors.As(err, &exitErr) && exitErr.Code == 2 {
			return "", nil
		}

		return "", fmt.Errorf("consultar al usuario %q: %w", nombre, err)
	}

	return strings.TrimSpace(resultado.Stdout), nil
}

// parsearUsuario extrae los datos de una línea de getent passwd.
func parsearUsuario(linea string) Usuario {
	campos := strings.Split(linea, ":")
	usuario := Usuario{}

	if len(campos) < 7 {
		return usuario
	}

	uid, err := strconv.Atoi(campos[2])
	if err != nil {
		return Usuario{}
	}

	usuario.Nombre = campos[0]
	usuario.UID = uid
	usuario.Home = campos[5]

	return usuario
}

// miembrosGrupo devuelve los miembros de la salida de getent group.
func miembrosGrupo(salida string) map[string]bool {
	miembros := map[string]bool{}

	for _, linea := range strings.Split(salida, "\n") {
		campos := strings.Split(linea, ":")
		if len(campos) < 4 {
			continue
		}

		for _, miembro := range strings.Split(campos[3], ",") {
			if miembro != "" {
				miembros[miembro] = true
			}
		}
	}

	return miembros
}
