// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Consola para Administración de Máquinas Virtuales GNU/Linux.
//
// Punto de entrada de la aplicación web: registra el servidor HTTP,
// los handlers de cada módulo y sirve los archivos estáticos y los
// módulos HTML de la interfaz. Es el único lugar que conoce a la vez
// los puertos del núcleo y sus adaptadores concretos.
package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"consola-vm/internal/adapters/localproc"
	"consola-vm/internal/adapters/ssh"
	"consola-vm/internal/adapters/vbox"
	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
	"consola-vm/internal/handlers"
	"consola-vm/internal/services"
)

// contenidoWeb contiene la interfaz: la aplicación compilada con Vite y
// la interfaz anterior, que se conserva mientras se retira. Se enumeran
// las carpetas necesarias para no embeber node_modules ni artefactos de
// desarrollo del frontend.
//
//go:embed all:web/dist all:web/modules all:web/static web/index.html
var contenidoWeb embed.FS

// Variables de entorno que configuran la conexión con la máquina.
const (
	envVM        = "VMCONS_VM"         // nombre o UUID de la VM en VirtualBox
	envSSHKey    = "VMCONS_SSH_KEY"    // ruta de la llave privada en el anfitrión
	envGuestUser = "VMCONS_GUEST_USER" // usuario de Guest Additions
	// #nosec G101 -- es el nombre de la variable de entorno, no la credencial.
	envGuestPass  = "VMCONS_GUEST_PASS" // contraseña de Guest Additions
	envDireccion  = "VMCONS_ADDR"       // dirección donde escucha la consola
	envWeb        = "VMCONS_WEB"        // directorio de la interfaz web
	direccionProd = ":8080"
)

// tiempoMaximoPeticion acota la duración de cada petición HTTP.
const tiempoMaximoPeticion = 2 * time.Minute

// main ensambla los adaptadores, los servicios y los handlers, y arranca
// el servidor HTTP de la consola.
func main() {
	if err := ejecutar(); err != nil {
		slog.Error("la consola terminó con error", "error", err)
		os.Exit(1)
	}
}

// ejecutar construye el servidor, lo pone a escuchar y se bloquea hasta
// que el servidor se detiene. Devuelve el error de arranque o de cierre.
func ejecutar() error {
	servidor, err := nuevoServidor()
	if err != nil {
		return err
	}

	if err := imprimirBanner(servidor.Addr, origenInterfaz(), os.Getenv(envVM)); err != nil {
		slog.Warn("no se pudo escribir el banner de arranque", "error", err)
	}

	if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("servir la consola en %q: %w", servidor.Addr, err)
	}

	return nil
}

// nuevoServidor arma el servidor HTTP con sus rutas y tiempos límite.
func nuevoServidor() (*http.Server, error) {
	manejador, err := nuevoMux()
	if err != nil {
		return nil, err
	}

	return &http.Server{
		Addr:              direccion(),
		Handler:           manejador,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      tiempoMaximoPeticion,
		IdleTimeout:       60 * time.Second,
	}, nil
}

// nuevoMux registra los endpoints de la API y el servido de la interfaz.
func nuevoMux() (http.Handler, error) {
	mux := http.NewServeMux()

	ejecutor := localproc.New()
	consola := nuevaConsola(ejecutor)
	sesiones := consola.sesiones

	llaves := services.NuevoLlaves(services.ConfigLlaves{
		Sesiones: sesiones,
		Ejecutor: ejecutor,
		Ruta:     rutaLlave(),
	})

	issue := services.NuevoIssue(sesiones)
	hostname := services.NuevoHostname(sesiones)

	sistema := handlers.NuevoSistema(services.NuevoPaquetes(sesiones), llaves, issue, hostname)
	usuarios := handlers.NuevoUsuarios(services.NuevoUsuarios(services.ConfigUsuarios{
		Sesiones:   sesiones,
		Ejecutor:   ejecutor,
		Directorio: filepath.Join(filepath.Dir(rutaLlave()), "users"),
	}))
	systemd := handlers.NuevoSystemd(services.NuevoSystemd(sesiones))
	haproxy := handlers.NuevoHaproxy(services.NuevoHaproxy(sesiones))

	mux.Handle("/api/session", handlers.NuevoSesion(
		sesiones,
		handlers.ConVinculador(consola.vinculacion),
		handlers.ConRegistro(consola.registro),
	))
	mux.Handle("/api/session/register", handlers.NuevoAcceso(consola.acceso))
	maquinas := handlers.NuevoMaquinas(consola.vinculacion)

	mux.Handle("/api/vms", maquinas)
	mux.HandleFunc("/api/vms/{vm}/poweron", maquinas.Encender)

	mux.HandleFunc("/api/packages", sistema.Paquetes)
	mux.HandleFunc("/api/packages/{name}", sistema.Paquete)
	mux.HandleFunc("/api/keys/root", sistema.LlaveRoot)
	mux.HandleFunc("/api/issue", sistema.Issue)
	mux.HandleFunc("/api/hostname", sistema.Hostname)

	mux.HandleFunc("/api/users", usuarios.Coleccion)
	mux.HandleFunc("/api/users/{name}", usuarios.Usuario)
	mux.HandleFunc("/api/users/{name}/key", usuarios.Llave)

	mux.HandleFunc("/api/units", systemd.Unidades)
	mux.HandleFunc("/api/units/{unit}", systemd.Unidad)
	mux.HandleFunc("/api/units/{unit}/restarts", systemd.Reinicios)

	mux.HandleFunc("/api/haproxy/backend/servers", haproxy.Servidores)
	mux.HandleFunc("/api/haproxy/backend/servers/{server}", haproxy.Servidor)

	if err := servirInterfaz(mux); err != nil {
		return nil, err
	}

	return mux, nil
}

// servirInterfaz publica la aplicación compilada con Vite en la raíz y
// conserva la interfaz anterior en /modules/ mientras se retira.
func servirInterfaz(mux *http.ServeMux) error {
	sub, err := fs.Sub(contenidoWeb, "web")
	if err != nil {
		// El sistema de archivos embebido es inmutable: si «web» no
		// existe, el binario se compiló mal.
		return fmt.Errorf("preparar los archivos web: %w", err)
	}

	dist, err := interfazWeb()
	if err != nil {
		return err
	}

	mux.Handle("/modules/", http.FileServerFS(sub))
	mux.Handle("/static/", http.FileServerFS(sub))

	// El enrutado por fragmento de URL no necesita reescrituras.
	mux.Handle("/assets/", http.FileServerFS(dist))
	mux.Handle("/", http.FileServerFS(dist))

	return nil
}

// consola reúne el descubrimiento de sesión y el inventario de máquinas
// que comparten los manejadores.
type consola struct {
	sesiones    *services.Descubridor
	vinculacion *services.Vinculacion
	acceso      *services.Acceso
	registro    *services.Almacen
}

// nuevaConsola arma el descubrimiento de sesión y el inventario con los
// adaptadores reales: VirtualBox para el estado, la dirección y el
// inventario, SSH como conector preferido y Guest Additions como respaldo.
func nuevaConsola(ejecutor ports.Executor) *consola {
	almacen := services.NuevoAlmacen(rutaCredenciales())
	maquina := vbox.New(ejecutor,
		vbox.WithCredenciales(almacen),
		vbox.WithGuestCredentials(usuarioInvitado(), os.Getenv(envGuestPass)),
	)

	if os.Getenv(envVM) == "" {
		slog.Warn("la consola arranca sin máquina vinculada; elígela en el selector", "variable", envVM)
	}

	conectorSSH := ssh.New(ejecutor, ssh.WithKey(rutaLlave()))

	sesiones := services.NuevoDescubridor(&services.ConfigDescubridor{
		Maquina:  maquina,
		Objetivo: domain.Target{ID: os.Getenv(envVM)},
		Conectores: []ports.Connector{
			conectorSSH,
			maquina,
		},
	})

	return &consola{
		sesiones: sesiones,
		acceso: services.NuevoAcceso(&services.ConfigAcceso{
			Almacen:   almacen,
			Maquina:   maquina,
			Guest:     maquina,
			SSH:       conectorSSH,
			Ejecutor:  ejecutor,
			RutaLlave: rutaLlave(),
			Objetivo:  sesiones.Objetivo,
		}),
		registro: almacen,
		vinculacion: services.NuevoVinculacion(&services.ConfigVinculacion{
			Inventario: maquina,
			Seleccion:  sesiones,
			Energia:    maquina,
			Maquina:    maquina,
			Registro:   almacen,
		}),
	}
}

// direccion devuelve la dirección de escucha configurada.
func direccion() string {
	if valor := os.Getenv(envDireccion); valor != "" {
		return valor
	}

	return direccionProd
}

// rutaCredenciales devuelve la ruta del archivo de credenciales por
// máquina.
func rutaCredenciales() string {
	return filepath.Join(filepath.Dir(rutaLlave()), "credenciales.json")
}

// rutaLlave devuelve la ruta de la llave privada de root: la variable
// de entorno o, por omisión, el directorio de configuración del usuario.
func rutaLlave() string {
	if valor := os.Getenv(envSSHKey); valor != "" {
		return valor
	}

	configuracion, err := os.UserConfigDir()
	if err != nil {
		configuracion = "."
	}

	return filepath.Join(configuracion, "vmcons", "id_ed25519")
}

// usuarioInvitado devuelve el usuario de Guest Additions, «root» por
// omisión.
func usuarioInvitado() string {
	if valor := os.Getenv(envGuestUser); valor != "" {
		return valor
	}

	return "root"
}
