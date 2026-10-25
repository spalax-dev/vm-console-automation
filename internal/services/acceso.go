// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// ErrCredencialesInvalidas indica que Guest Additions rechazó el
// usuario o la contraseña.
var ErrCredencialesInvalidas = errors.New("las credenciales no permiten ejecutar órdenes")

// ConfigAcceso reúne las dependencias para registrar una máquina en la
// consola.
type ConfigAcceso struct {
	// Almacen guarda las credenciales y el estado de registro.
	Almacen *Almacen
	// Maquina consulta el estado y la dirección de la máquina activa.
	Maquina ports.Machine
	// Guest abre una sesión por Guest Additions con las credenciales
	// guardadas.
	Guest ports.Connector
	// SSH comprueba que la llave quede operativa tras la adecuación.
	SSH ports.Connector
	// SSHPassword abre una sesión con usuario y contraseña cuando Guest
	// Additions no puede autenticar.
	SSHPassword ports.PasswordConnector
	// Ejecutor genera la llave de root en el anfitrión.
	Ejecutor ports.Executor
	// RutaLlave es la llave privada de root en el anfitrión.
	RutaLlave string
	// Objetivo devuelve la máquina vinculada.
	Objetivo func() domain.Target
}

// Acceso registra una máquina: guarda sus credenciales, prepara SSH y
// deja constancia del estado para no volver a pedirlas.
type Acceso struct {
	almacen     *Almacen
	maquina     ports.Machine
	guest       ports.Connector
	ssh         ports.Connector
	sshPassword ports.PasswordConnector
	ejecutor    ports.Executor
	rutaLlave   string
	objetivo    func() domain.Target
}

// NuevoAcceso construye el servicio de registro.
func NuevoAcceso(config *ConfigAcceso) *Acceso {
	return &Acceso{
		almacen:     config.Almacen,
		maquina:     config.Maquina,
		guest:       config.Guest,
		ssh:         config.SSH,
		sshPassword: config.SSHPassword,
		ejecutor:    config.Ejecutor,
		rutaLlave:   config.RutaLlave,
		objetivo:    config.Objetivo,
	}
}

// Registrar guarda las credenciales, instala SSH con la llave de root y
// verifica la conexión. Si algo falla, deja la máquina sin registrar y
// restaura las credenciales anteriores.
func (a *Acceso) Registrar(ctx context.Context, usuario, password string) (Resultado, error) {
	objetivo, direccion, err := a.resolver(ctx)
	if err != nil {
		return Resultado{}, err
	}

	anterior, existia := a.almacen.Obtener(objetivo.ID)

	if err := a.almacen.Guardar(objetivo.ID, Credenciales{
		Usuario:    usuario,
		Password:   password,
		Puerto:     22,
		Registrada: false,
	}); err != nil {
		return Resultado{}, err
	}

	exito := false

	defer func() {
		if !exito {
			a.restaurar(objetivo.ID, anterior, existia)
		}
	}()

	salida, err := a.prepararSSH(ctx, objetivo, direccion, password)
	if errors.Is(err, ErrCredencialesInvalidas) {
		errorGuest := err

		salida, err = a.prepararSSHPassword(ctx, objetivo, direccion, usuario, password)
		if err != nil {
			err = fmt.Errorf("%w (Guest Additions: %w; SSH: %w)", ErrCredencialesInvalidas, errorGuest, err)
		}
	}

	if err != nil {
		slog.Warn("registrar la máquina", "vm", objetivo.ID, "usuario", usuario, claveErrorLog, err)

		return salida, err
	}

	if err := a.almacen.Guardar(objetivo.ID, Credenciales{
		Usuario:    usuario,
		Password:   password,
		Puerto:     22,
		Registrada: true,
	}); err != nil {
		return salida, err
	}

	exito = true

	return salida, nil
}

// prepararSSH valida las credenciales, instala SSH con la llave de root
// y comprueba que la conexión por llave funcione.
func (a *Acceso) prepararSSH(
	ctx context.Context, objetivo domain.Target, direccion netip.Addr, password string,
) (Resultado, error) {
	clave, err := a.asegurarLlave(ctx)
	if err != nil {
		return Resultado{}, err
	}

	sesion, err := a.guest.Connect(ctx, domain.Target{ID: objetivo.ID, Address: direccion})
	if err != nil {
		return Resultado{}, fmt.Errorf("abrir la sesión de Guest Additions: %w", err)
	}

	defer cerrarSesion(sesion)

	verificacion, err := sesion.Run(ctx, &domain.Command{Path: "/usr/bin/id", Args: []string{"-u"}})
	if err != nil {
		return errorDeCredenciales(verificacion, err)
	}

	return a.instalarConSesion(ctx, sesion, objetivo, direccion, clave, strings.TrimSpace(verificacion.Stdout), password)
}

// prepararSSHPassword registra una máquina usando SSH con contraseña,
// como respaldo cuando Guest Additions rechaza las credenciales.
func (a *Acceso) prepararSSHPassword(
	ctx context.Context, objetivo domain.Target, direccion netip.Addr, usuario, password string,
) (Resultado, error) {
	if a.sshPassword == nil {
		return Resultado{}, fmt.Errorf("%w: no hay respaldo SSH disponible", ErrCredencialesInvalidas)
	}

	clave, err := a.asegurarLlave(ctx)
	if err != nil {
		return Resultado{}, err
	}

	sesion, err := a.sshPassword.ConnectPassword(
		ctx, domain.Target{ID: objetivo.ID, Address: direccion}, usuario, password,
	)
	if err != nil {
		return Resultado{}, fmt.Errorf("%w: %w", ErrCredencialesInvalidas, err)
	}

	defer cerrarSesion(sesion)

	verificacion, err := sesion.Run(ctx, &domain.Command{Path: "/usr/bin/id", Args: []string{"-u"}})
	if err != nil {
		return errorDeCredenciales(verificacion, err)
	}

	return a.instalarConSesion(ctx, sesion, objetivo, direccion, clave, strings.TrimSpace(verificacion.Stdout), password)
}

// instalarConSesion ejecuta la adecuación dentro de una sesión ya
// autenticada y verifica la llave.
func (a *Acceso) instalarConSesion(
	ctx context.Context, sesion ports.Session, objetivo domain.Target, direccion netip.Addr,
	clave, uid, password string,
) (Resultado, error) {
	cmd := guionInstalacion(clave)
	operacion := cmd.ShellLine()

	if uid != "0" {
		cmd = guionInstalacionSudo(clave, password)
		operacion = "instalar SSH y configurar sshd con sudo"
	}

	resultado, err := sesion.Run(ctx, cmd)
	salida := Resultado{
		Mensaje:   "Máquina registrada: SSH quedó preparado para su administración.",
		Operacion: operacion,
		Salida:    resultado.Stdout,
		Stderr:    resultado.Stderr,
		Codigo:    resultado.ExitCode,
	}

	if err != nil {
		return salida, fmt.Errorf("preparar SSH en la máquina: %w", err)
	}

	if err := a.ssh.Probe(ctx, domain.Target{ID: objetivo.ID, Address: direccion}); err != nil {
		return salida, fmt.Errorf("verificar el acceso SSH: %w", err)
	}

	return salida, nil
}

// errorDeCredenciales arma el error de autenticación con el stderr de la
// orden de verificación.
func errorDeCredenciales(resultado domain.Result, err error) (Resultado, error) {
	detalle := strings.TrimSpace(resultado.Stderr)
	if detalle == "" {
		detalle = err.Error()
	}

	return Resultado{
		Codigo: resultado.ExitCode,
		Stderr: resultado.Stderr,
	}, fmt.Errorf("%w: %s", ErrCredencialesInvalidas, detalle)
}

// restaurar deja el almacén como estaba antes del intento de registro.
func (a *Acceso) restaurar(vm string, anterior Credenciales, existia bool) {
	if existia {
		if err := a.almacen.Guardar(vm, anterior); err != nil {
			slog.Warn("restaurar las credenciales", claveErrorLog, err)
		}

		return
	}

	if err := a.almacen.Borrar(vm); err != nil {
		slog.Warn("borrar las credenciales", claveErrorLog, err)
	}
}

// asegurarLlave devuelve la llave pública de root, generando el par si
// todavía no existe en el anfitrión.
func (a *Acceso) asegurarLlave(ctx context.Context) (string, error) {
	if _, err := os.Stat(a.rutaLlave); err == nil {
		return leerClavePublica(a.rutaLlave)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("comprobar la llave %q: %w", a.rutaLlave, err)
	}

	if err := os.MkdirAll(filepath.Dir(a.rutaLlave), 0o700); err != nil {
		return "", fmt.Errorf(msgCrearDirLlaves, err)
	}

	if err := generarPar(ctx, a.ejecutor, a.rutaLlave, "vmcons-root"); err != nil {
		return "", err
	}

	return leerClavePublica(a.rutaLlave)
}

// Olvidar elimina el registro y las credenciales de la máquina activa.
func (a *Acceso) Olvidar() error {
	objetivo := a.objetivo()
	if objetivo.ID == "" {
		return ErrSinMaquina
	}

	return a.almacen.Borrar(objetivo.ID)
}

// resolver comprueba la máquina activa y devuelve su dirección.
func (a *Acceso) resolver(ctx context.Context) (domain.Target, netip.Addr, error) {
	objetivo := a.objetivo()
	if objetivo.ID == "" {
		return domain.Target{}, netip.Addr{}, fmt.Errorf("%w: elige una máquina del inventario", ErrSinMaquina)
	}

	estado, err := a.maquina.State(ctx, objetivo)
	if err != nil {
		return domain.Target{}, netip.Addr{}, fmt.Errorf(msgEstadoMaquina, err)
	}

	if estado != domain.StateRunning {
		return domain.Target{}, netip.Addr{}, fmt.Errorf(
			"%w: la máquina %q está en estado %q", ErrSesionNoDisponible, objetivo.ID, estado,
		)
	}

	direccion, err := a.maquina.Address(ctx, objetivo)
	if err != nil {
		return domain.Target{}, netip.Addr{}, fmt.Errorf("descubrir la dirección de la máquina: %w", err)
	}

	return objetivo, direccion, nil
}
