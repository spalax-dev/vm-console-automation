// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// rutaHaproxy es la configuración de HAProxy en la máquina.
const rutaHaproxy = "/etc/haproxy/haproxy.cfg"

// rutaSocketHaproxy es el socket de estadísticas del balanceador; la
// configuración por omisión de Debian lo define en el bloque global.
const rutaSocketHaproxy = "/run/haproxy/admin.sock"

// binSocat consulta el socket de estadísticas dentro de la máquina.
const binSocat = "/usr/bin/socat"

// ordenShowStat pide la tabla de estadísticas por el socket.
const ordenShowStat = `printf 'show stat\n' | ` + binSocat + ` - UNIX-CONNECT:` + rutaSocketHaproxy

// msgLeerConfig es el contexto de error al leer la configuración.
const msgLeerConfig = "leer la configuración de HAProxy: %w"

// ErrBackendNoEncontrado indica que la configuración no tiene un bloque
// backend.
var ErrBackendNoEncontrado = errors.New("no se encontró un back-end en la configuración")

// ErrServidorExistente indica que el servidor ya está en el back-end.
var ErrServidorExistente = errors.New("el servidor ya está en el back-end")

// ErrServidorNoEncontrado indica que el servidor no está en el back-end.
var ErrServidorNoEncontrado = errors.New("el servidor no está en el back-end")

// Servidor describe un servidor del back-end de HAProxy.
type Servidor struct {
	// Direccion es la dirección IPv4 del servidor.
	Direccion string
	// Puerto es el puerto TCP.
	Puerto int
	// Estado es el estado informado por HAProxy: up, down o unknown
	// cuando el socket de estadísticas no responde.
	Estado string
	// nombre es el identificador del servidor en la configuración, que
	// usa la tabla de estadísticas para informar su estado.
	nombre string
}

// Haproxy implementa el caso de uso del back-end de HAProxy.
type Haproxy struct {
	sesiones AbridorSesion
}

// NuevoHaproxy construye el servicio con el abridor de sesiones dado.
func NuevoHaproxy(sesiones AbridorSesion) *Haproxy {
	return &Haproxy{sesiones: sesiones}
}

// Listar devuelve los servidores del primer bloque backend.
func (h *Haproxy) Listar(ctx context.Context) ([]Servidor, error) {
	sesion, err := h.sesiones.Abrir(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrir la sesión para listar el back-end: %w", err)
	}

	defer cerrarSesion(sesion)

	config, err := ejecutarSalida(ctx, sesion, binCat, rutaHaproxy)
	if err != nil {
		return nil, fmt.Errorf(msgLeerConfig, err)
	}

	servidores, encontrado := parsearServidores(config)
	if !encontrado {
		return nil, ErrBackendNoEncontrado
	}

	// El estado de cada servidor se toma del socket de estadísticas. Si la
	// consulta falla, el listado sigue siendo válido con estado unknown.
	if len(servidores) > 0 {
		estados := estadosDeBackend(ctx, sesion)

		for i := range servidores {
			if estado, ok := estados[servidores[i].nombre]; ok {
				servidores[i].Estado = estado
			}
		}
	}

	return servidores, nil
}

// estadosDeBackend consulta el socket de estadísticas y devuelve el estado
// de cada servidor por su nombre en la configuración. Un error de la
// consulta no interrumpe el listado: se devuelve un mapa vacío y los
// servidores conservan el estado unknown.
func estadosDeBackend(ctx context.Context, sesion ports.Session) map[string]string {
	cmd := &domain.Command{Path: binSh, Args: []string{"-c", ordenShowStat}}

	resultado, err := sesion.Run(ctx, cmd)
	if err != nil {
		return map[string]string{}
	}

	return parsearEstados(resultado.Stdout)
}

// parsearEstados traduce el campo status de la salida CSV del socket de
// estadísticas a up, down o unknown. Las filas incompletas o con estados
// que la consola no distingue se ignoran.
func parsearEstados(salida string) map[string]string {
	estados := make(map[string]string)

	for _, linea := range strings.Split(salida, "\n") {
		if linea == "" || strings.HasPrefix(linea, "#") {
			continue
		}

		campos := strings.Split(linea, ",")
		if len(campos) < 18 {
			continue
		}

		if estado, ok := estadoHaproxy(campos[17]); ok {
			estados[campos[1]] = estado
		}
	}

	return estados
}

// estadoHaproxy traduce el estado de HAProxy al vocabulario del contrato.
// Los estados de mantenimiento y de drenaje se informan como down porque
// el servidor no recibe tráfico nuevo.
func estadoHaproxy(estado string) (string, bool) {
	switch estado {
	case "UP":
		return "up", true
	case "DOWN", "MAINT", "DRAIN", "NOLB":
		return "down", true
	default:
		return "", false
	}
}

// Agregar agrega un servidor al back-end y recarga HAProxy.
func (h *Haproxy) Agregar(ctx context.Context, direccion string, puerto int) (Resultado, error) {
	sesion, err := h.sesiones.Abrir(ctx)
	if err != nil {
		return Resultado{}, fmt.Errorf("abrir la sesión para agregar el servidor: %w", err)
	}

	defer cerrarSesion(sesion)

	config, err := ejecutarSalida(ctx, sesion, binCat, rutaHaproxy)
	if err != nil {
		return Resultado{}, fmt.Errorf(msgLeerConfig, err)
	}

	nuevo, err := agregarServidor(config, direccion, puerto)
	if err != nil {
		return Resultado{}, err
	}

	return h.escribir(ctx, sesion, nuevo, "Servidor agregado al back-end.")
}

// Eliminar elimina un servidor del back-end y recarga HAProxy.
func (h *Haproxy) Eliminar(ctx context.Context, direccion string, puerto int) (Resultado, error) {
	sesion, err := h.sesiones.Abrir(ctx)
	if err != nil {
		return Resultado{}, fmt.Errorf("abrir la sesión para eliminar el servidor: %w", err)
	}

	defer cerrarSesion(sesion)

	config, err := ejecutarSalida(ctx, sesion, binCat, rutaHaproxy)
	if err != nil {
		return Resultado{}, fmt.Errorf(msgLeerConfig, err)
	}

	nuevo, err := quitarServidor(config, direccion, puerto)
	if err != nil {
		return Resultado{}, err
	}

	return h.escribir(ctx, sesion, nuevo, "Servidor eliminado del back-end.")
}

// escribir valida la configuración propuesta y, solo si es correcta, la
// instala y recarga HAProxy.
func (h *Haproxy) escribir(
	ctx context.Context, sesion ports.Session, config, mensaje string,
) (Resultado, error) {
	guion := "printf '%s' " + domain.Quote(config) + " > /tmp/haproxy.cfg.vmcons && " +
		binHaproxy + " -c -f /tmp/haproxy.cfg.vmcons && " +
		"mv /tmp/haproxy.cfg.vmcons " + rutaHaproxy + " && " +
		binSystemctl + " reload haproxy"

	cmd := &domain.Command{Path: binSh, Args: []string{"-c", guion}}

	resultado, err := sesion.Run(ctx, cmd)

	salida := Resultado{
		Mensaje:   mensaje,
		Operacion: cmd.ShellLine(),
		Salida:    resultado.Stdout,
		Stderr:    resultado.Stderr,
		Codigo:    resultado.ExitCode,
	}

	if err != nil {
		return salida, fmt.Errorf("validar y recargar HAProxy: %w", err)
	}

	return salida, nil
}

// parsearServidores extrae los servidores del primer bloque backend.
func parsearServidores(config string) (servidores []Servidor, encontrado bool) {
	servidores = []Servidor{}

	inicio, fin := limitesBackend(strings.Split(config, "\n"))
	if inicio < 0 {
		return servidores, false
	}

	for _, linea := range strings.Split(config, "\n")[inicio+1 : fin] {
		campos := strings.Fields(linea)
		if len(campos) < 3 || campos[0] != "server" {
			continue
		}

		destino, err := netip.ParseAddrPort(campos[2])
		if err != nil {
			continue
		}

		servidores = append(servidores, Servidor{
			Direccion: destino.Addr().String(),
			Puerto:    int(destino.Port()),
			Estado:    "unknown",
			nombre:    campos[1],
		})
	}

	return servidores, true
}

// agregarServidor inserta la línea del servidor al final del primer
// bloque backend.
func agregarServidor(config, direccion string, puerto int) (string, error) {
	lineas := strings.Split(config, "\n")

	inicio, fin := limitesBackend(lineas)
	if inicio < 0 {
		return "", ErrBackendNoEncontrado
	}

	for _, linea := range lineas[inicio+1 : fin] {
		if servidorCoincide(linea, direccion, puerto) {
			return "", ErrServidorExistente
		}
	}

	nombre := "srv_" + strings.NewReplacer(".", "_", ":", "_").Replace(direccion) + "_" + strconv.Itoa(puerto)
	nueva := "    server " + nombre + " " + direccion + ":" + strconv.Itoa(puerto) + " check"

	nuevas := make([]string, 0, len(lineas)+1)
	nuevas = append(nuevas, lineas[:fin]...)
	nuevas = append(nuevas, nueva)
	nuevas = append(nuevas, lineas[fin:]...)

	return strings.Join(nuevas, "\n"), nil
}

// quitarServidor elimina la línea del servidor del primer bloque
// backend.
func quitarServidor(config, direccion string, puerto int) (string, error) {
	lineas := strings.Split(config, "\n")

	inicio, fin := limitesBackend(lineas)
	if inicio < 0 {
		return "", ErrBackendNoEncontrado
	}

	nuevas := make([]string, 0, len(lineas))
	nuevas = append(nuevas, lineas[:inicio+1]...)
	encontrado := false

	for _, linea := range lineas[inicio+1 : fin] {
		if servidorCoincide(linea, direccion, puerto) {
			encontrado = true

			continue
		}

		nuevas = append(nuevas, linea)
	}

	if !encontrado {
		return "", ErrServidorNoEncontrado
	}

	nuevas = append(nuevas, lineas[fin:]...)

	return strings.Join(nuevas, "\n"), nil
}

// servidorCoincide informa si la línea es un servidor con el destino
// indicado.
func servidorCoincide(linea, direccion string, puerto int) bool {
	campos := strings.Fields(linea)
	if len(campos) < 3 || campos[0] != "server" {
		return false
	}

	destino, err := netip.ParseAddrPort(campos[2])
	if err != nil {
		return false
	}

	return destino.Addr().String() == direccion && int(destino.Port()) == puerto
}

// limitesBackend devuelve el inicio y el fin del primer bloque backend.
func limitesBackend(lineas []string) (inicio, fin int) {
	inicio = -1

	for i, linea := range lineas {
		limpia := strings.TrimSpace(linea)
		if limpia == "" || strings.HasPrefix(limpia, "#") {
			continue
		}

		if strings.HasPrefix(linea, " ") || strings.HasPrefix(linea, "\t") {
			continue
		}

		campos := strings.Fields(limpia)
		if campos[0] == "backend" {
			if inicio == -1 {
				inicio = i
			} else {
				return inicio, i
			}

			continue
		}

		if inicio != -1 {
			return inicio, i
		}
	}

	if inicio != -1 {
		return inicio, len(lineas)
	}

	return -1, -1
}
