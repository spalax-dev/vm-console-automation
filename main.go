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
	"time"

	"consola-vm/internal/adapters/localproc"
	"consola-vm/internal/adapters/ssh"
	"consola-vm/internal/adapters/vbox"
	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
	"consola-vm/internal/handlers"
	"consola-vm/internal/services"
)

// contenidoWeb contiene la interfaz: módulos HTML y archivos estáticos.
//
//go:embed all:web
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

	mux.Handle("/api/sesion", handlers.NuevoSesion(descubridor()))

	sub, err := fs.Sub(contenidoWeb, "web")
	if err != nil {
		// El sistema de archivos embebido es inmutable: si «web» no
		// existe, el binario se compiló mal.
		panic(fmt.Errorf("preparar los archivos web: %w", err))
	}

	mux.Handle("/modules/", http.FileServerFS(sub))
	mux.Handle("/static/", http.FileServerFS(sub))
	mux.Handle("/", http.RedirectHandler("/modules/index.html", http.StatusSeeOther))

	return mux
}

// descubridor construye el caso de uso de descubrimiento de sesión con
// los adaptadores reales: VirtualBox para el estado y la dirección, SSH
// como conector preferido y Guest Additions como respaldo.
func descubridor() handlers.DescubridorSesion {
	idVM := os.Getenv(envVM)
	if idVM == "" {
		slog.Warn("no se configuró la VM; /api/sesion responderá 503", "variable", envVM)

		return sinConfigurar{}
	}

	ejecutor := localproc.New()

	return services.NuevoDescubridor(&services.ConfigDescubridor{
		Maquina:  vbox.New(ejecutor, vbox.WithGuestCredentials(usuarioInvitado(), os.Getenv(envGuestPass))),
		Objetivo: domain.Target{ID: idVM},
		Conectores: []ports.Connector{
			ssh.New(ejecutor, ssh.WithKey(os.Getenv(envSSHKey))),
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

// direccion devuelve la dirección de escucha configurada.
func direccion() string {
	if valor := os.Getenv(envDireccion); valor != "" {
		return valor
	}

	return direccionProd
}

// usuarioInvitado devuelve el usuario de Guest Additions, «root» por
// omisión.
func usuarioInvitado() string {
	if valor := os.Getenv(envGuestUser); valor != "" {
		return valor
	}

	return "root"
}
