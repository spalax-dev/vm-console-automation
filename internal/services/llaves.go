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
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// ErrLlaveExistente indica que la llave privada de root ya está en el
// anfitrión y no se sobrescribe.
var ErrLlaveExistente = errors.New("la llave SSH de root ya existe")

// ErrLlaveNoEncontrada indica que todavía no se ha generado la llave de
// root.
var ErrLlaveNoEncontrada = errors.New("la llave SSH de root no existe")

// ConfigLlaves reúne las dependencias del servicio de llaves.
type ConfigLlaves struct {
	// Sesiones abre la sesión con la máquina para instalar la llave.
	Sesiones AbridorSesion
	// Ejecutor lanza ssh-keygen en el equipo anfitrión.
	Ejecutor ports.Executor
	// Ruta es la ruta de la llave privada en el anfitrión.
	Ruta string
}

// Llaves implementa el caso de uso de las llaves SSH de root.
type Llaves struct {
	sesiones AbridorSesion
	ejecutor ports.Executor
	ruta     string
}

// NuevoLlaves construye el servicio de llaves con las dependencias dadas.
func NuevoLlaves(config ConfigLlaves) *Llaves {
	return &Llaves{
		sesiones: config.Sesiones,
		ejecutor: config.Ejecutor,
		ruta:     config.Ruta,
	}
}

// Generar crea el par de llaves en el anfitrión, instala la pública en
// authorized_keys y deshabilita el acceso por contraseña. Devuelve
// ErrLlaveExistente si la llave privada ya está presente.
func (l *Llaves) Generar(ctx context.Context) (Resultado, error) {
	if _, err := os.Stat(l.ruta); err == nil {
		return Resultado{}, ErrLlaveExistente
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Resultado{}, fmt.Errorf("comprobar la llave %q: %w", l.ruta, err)
	}

	if err := os.MkdirAll(filepath.Dir(l.ruta), 0o700); err != nil {
		return Resultado{}, fmt.Errorf("crear el directorio de llaves: %w", err)
	}

	// La sesión se abre antes de generar la llave: mientras no exista, el
	// descubrimiento usa Guest Additions, único canal disponible.
	sesion, err := l.sesiones.Abrir(ctx)
	if err != nil {
		return Resultado{}, fmt.Errorf("abrir la sesión con la máquina: %w", err)
	}

	defer cerrarSesion(sesion)

	if err := generarPar(ctx, l.ejecutor, l.ruta, "vmcons-root"); err != nil {
		return Resultado{}, err
	}

	clave, err := leerClavePublica(l.ruta)
	if err != nil {
		return Resultado{}, err
	}

	cmd := guionInstalacion(clave)

	resultado, err := sesion.Run(ctx, cmd)
	salida := Resultado{
		Mensaje:   "Llave SSH de root generada e instalada; el acceso por contraseña quedó deshabilitado.",
		Operacion: cmd.ShellLine(),
		Salida:    resultado.Stdout,
		Stderr:    resultado.Stderr,
		Codigo:    resultado.ExitCode,
	}

	if err != nil {
		return salida, fmt.Errorf("instalar la llave y configurar sshd: %w", err)
	}

	return salida, nil
}

// Descargar devuelve el nombre y el contenido de la llave privada, sin
// exponerla en los logs.
func (l *Llaves) Descargar() (nombre string, contenido []byte, err error) {
	contenido, err = os.ReadFile(l.ruta)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil, ErrLlaveNoEncontrada
	}

	if err != nil {
		return "", nil, fmt.Errorf("leer la llave privada: %w", err)
	}

	return filepath.Base(l.ruta), contenido, nil
}

// generarPar crea un par de llaves ed25519 en el anfitrión.
func generarPar(ctx context.Context, ejecutor ports.Executor, ruta, comentario string) error {
	if _, err := ejecutor.Execute(
		ctx, "ssh-keygen", "-t", "ed25519", "-N", "", "-f", ruta, "-C", comentario,
	); err != nil {
		return fmt.Errorf("generar el par de llaves: %w", err)
	}

	return nil
}

// leerClavePublica devuelve el contenido de la llave pública sin el
// salto de línea final.
func leerClavePublica(ruta string) (string, error) {
	// #nosec G304 -- la ruta proviene de la configuración de la consola.
	clave, err := os.ReadFile(ruta + ".pub")
	if err != nil {
		return "", fmt.Errorf("leer la llave pública: %w", err)
	}

	return strings.TrimSpace(string(clave)), nil
}

// guionInstalacion construye el guion POSIX que instala la llave pública
// y deshabilita el acceso por contraseña.
func guionInstalacion(clave string) *domain.Command {
	configuracion := "PasswordAuthentication no\nPermitRootLogin prohibit-password\n"

	guion := "if [ ! -x /usr/sbin/sshd ]; then " +
		"DEBIAN_FRONTEND=noninteractive apt-get install -y openssh-server; fi && " +
		"install -d -m 700 /root/.ssh && " +
		"printf '%s\\n' " + domain.Quote(clave) + " >> /root/.ssh/authorized_keys && " +
		"chmod 600 /root/.ssh/authorized_keys && " +
		"install -d -m 755 /etc/ssh/sshd_config.d && " +
		"printf '%s' " + domain.Quote(configuracion) + " > /etc/ssh/sshd_config.d/99-vmcons.conf && " +
		"systemctl enable --now ssh"

	return &domain.Command{Path: binSh, Args: []string{"-c", guion}}
}
