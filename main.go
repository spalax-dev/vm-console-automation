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
	"context"
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
	servidor := nuevoServidor()

	slog.Info("consola vmcons escuchando", "direccion", servidor.Addr)

	if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("servir la consola en %q: %w", servidor.Addr, err)
	}

	return nil
}

// nuevoServidor arma el servidor HTTP con sus rutas y tiempos límite.
func nuevoServidor() *http.Server {
	return &http.Server{
		Addr:              direccion(),
		Handler:           nuevoMux(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      tiempoMaximoPeticion,
		IdleTimeout:       60 * time.Second,
	}
}

// nuevoMux registra los endpoints de la API y el servido de la interfaz.
func nuevoMux() http.Handler {
	mux := http.NewServeMux()

	sesiones := descubridor()
	ejecutor := localproc.New()

	llaves := services.NuevoLlaves(services.ConfigLlaves{
		Sesiones: sesiones,
		Ejecutor: ejecutor,
		Ruta:     rutaLlave(),
	})

	issue := services.NuevoIssue(sesiones)
	hostname := services.NuevoHostname(sesiones)

	sistema := handlers.NuevoSistema(services.NuevoPaquetes(sesiones), llaves, issue, hostname)
	usuarios := handlers.NuevoUsuarios()
	systemd := handlers.NuevoSystemd()
	haproxy := handlers.NuevoHaproxy()

	mux.Handle("/api/session", handlers.NuevoSesion(sesiones))

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

	sub, err := fs.Sub(contenidoWeb, "web")
	if err != nil {
		// El sistema de archivos embebido es inmutable: si «web» no
		// existe, el binario se compiló mal.
		panic(fmt.Errorf("preparar los archivos web: %w", err))
	}

	// La aplicación compilada con Vite se sirve en la raíz; el enrutado
	// por fragmento de URL no necesita reescrituras en el servidor.
	dist, err := fs.Sub(contenidoWeb, "web/dist")
	if err != nil {
		panic(fmt.Errorf("preparar la aplicación web: %w", err))
	}

	// La interfaz anterior se conserva accesible mientras se retira.
	mux.Handle("/modules/", http.FileServerFS(sub))
	mux.Handle("/static/", http.FileServerFS(sub))

	mux.Handle("/assets/", http.FileServerFS(dist))
	mux.Handle("/", http.FileServerFS(dist))

	return mux
}

// sesiones reúne el descubrimiento y la apertura de sesiones que
// necesitan el handler de sesión y los servicios de los módulos.
type sesiones interface {
	handlers.DescubridorSesion
	services.AbridorSesion
}

// descubridor construye el caso de uso de descubrimiento de sesión con
// los adaptadores reales: VirtualBox para el estado y la dirección, SSH
// como conector preferido y Guest Additions como respaldo.
func descubridor() sesiones {
	idVM := os.Getenv(envVM)
	if idVM == "" {
		slog.Warn("no se configuró la VM; /api/session responderá 503", "variable", envVM)

		return sinConfigurar{}
	}

	ejecutor := localproc.New()

	return services.NuevoDescubridor(&services.ConfigDescubridor{
		Maquina:  vbox.New(ejecutor, vbox.WithGuestCredentials(usuarioInvitado(), os.Getenv(envGuestPass))),
		Objetivo: domain.Target{ID: idVM},
		Conectores: []ports.Connector{
			ssh.New(ejecutor, ssh.WithKey(rutaLlave())),
			vbox.New(ejecutor, vbox.WithGuestCredentials(usuarioInvitado(), os.Getenv(envGuestPass))),
		},
	})
}

// sinConfigurar responde «sin sesión» cuando falta la variable con el
// nombre de la VM, en lugar de fallar al arrancar la consola.
type sinConfigurar struct{}

// Descubrir implementa handlers.DescubridorSesion.
func (sinConfigurar) Descubrir(context.Context) (services.InfoSesion, error) {
	return services.InfoSesion{}, fmt.Errorf(
		"%w: configure la variable %s", services.ErrSesionNoDisponible, envVM,
	)
}

// Abrir implementa services.AbridorSesion.
func (sinConfigurar) Abrir(context.Context) (ports.Session, error) {
	return nil, fmt.Errorf(
		"%w: configure la variable %s", services.ErrSesionNoDisponible, envVM,
	)
}

// direccion devuelve la dirección de escucha configurada.
func direccion() string {
	if valor := os.Getenv(envDireccion); valor != "" {
		return valor
	}

	return direccionProd
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
