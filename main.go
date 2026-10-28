// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package main arranca la consola de administración de máquinas virtuales
// GNU/Linux: registra el servidor HTTP, los handlers de cada módulo y la
// interfaz web, y es el único lugar que conoce a la vez los puertos del
// núcleo y sus adaptadores concretos.
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
	"strings"
	"time"

	"consola-vm/internal/adapters/localproc"
	"consola-vm/internal/adapters/ssh"
	"consola-vm/internal/adapters/vbox"
	"consola-vm/internal/config"
	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
	"consola-vm/internal/handlers"
	"consola-vm/internal/httpcache"
	"consola-vm/internal/services"
	"consola-vm/internal/terminal"
	"consola-vm/internal/webui"
)

// embeddedWeb contiene la interfaz: la aplicación compilada con Vite y
// la interfaz anterior, que se conserva mientras se retira. Se enumeran
// las carpetas necesarias para no embeber node_modules ni artefactos de
// desarrollo del frontend.
//
//go:embed all:web/dist all:web/modules all:web/static web/index.html
var embeddedWeb embed.FS

// errorKey es la clave de los errores en el log de arranque.
const errorKey = "error"

// maxRequestTime acota la duración de cada petición HTTP.
const maxRequestTime = 2 * time.Minute

// cachePolicies declara cuánto se reutiliza cada lectura de la API. Las
// consultas a la máquina abren sesión y ejecutan órdenes, así que se
// guardan unos segundos; cualquier escritura vacía la caché completa.
func cachePolicies() []httpcache.Policy {
	return []httpcache.Policy{
		{Prefijo: "/api/session", TTL: 3 * time.Second},
		{Prefijo: "/api/vms", TTL: 5 * time.Second},
		{Prefijo: "/api/units", TTL: 5 * time.Second},
		{Prefijo: "/api/packages", TTL: 15 * time.Second},
		{Prefijo: "/api/users", TTL: 15 * time.Second},
		{Prefijo: "/api/haproxy", TTL: 15 * time.Second},
		{Prefijo: "/api/issue", TTL: 30 * time.Second},
		{Prefijo: "/api/hostname", TTL: 30 * time.Second},
	}
}

// main ensambla los adaptadores, los servicios y los handlers, y arranca
// el servidor HTTP de la consola.
func main() {
	configureLog()

	if err := run(); err != nil {
		slog.Error("la consola terminó con error", errorKey, err)
		os.Exit(1)
	}
}

// run construye el servidor, lo pone a escuchar y se bloquea hasta
// que el servidor se detiene. Devuelve el error de arranque o de cierre.
func run() error {
	server, err := newServer()
	if err != nil {
		return err
	}

	if err := terminal.PrintBanner(os.Stdout, server.Addr); err != nil {
		slog.Warn("no se pudo escribir el banner de arranque", errorKey, err)
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("servir la consola en %q: %w", server.Addr, err)
	}

	return nil
}

// newServer arma el servidor HTTP con sus rutas y tiempos límite.
// Devuelve también la máquina vinculada, para el banner de arranque.
func newServer() (*http.Server, error) {
	handler, err := newMux()
	if err != nil {
		return nil, err
	}

	handler = httpcache.New(cachePolicies()).Wrap(handler)

	return &http.Server{
		Addr:              config.Address(),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      maxRequestTime,
		IdleTimeout:       60 * time.Second,
	}, nil
}

// newMux registra los endpoints de la API y el servido de la interfaz.
func newMux() (http.Handler, error) {
	mux := http.NewServeMux()

	ejecutor := localproc.New()
	console := newConsole(ejecutor)
	sessions := console.sessions

	keys := services.NewKeys(services.KeysConfig{
		Sessions: sessions,
		Ejecutor: ejecutor,
		Path:     config.SSHKeyPath(),
	})

	issue := services.NewIssue(sessions)
	hostname := services.NewHostname(sessions)

	system := handlers.NewSystem(services.NewPackages(sessions), keys, issue, hostname)
	users := handlers.NewUsers(services.NewUsers(services.UsersConfig{
		Sessions:  sessions,
		Ejecutor:  ejecutor,
		Directory: filepath.Join(filepath.Dir(config.SSHKeyPath()), "users"),
	}))
	systemd := handlers.NewSystemd(services.NewSystemd(sessions))
	haproxy := handlers.NewHAProxy(services.NewHAProxy(sessions))

	mux.Handle("/api/session", handlers.NewSession(
		sessions,
		handlers.WithBinder(console.binding),
		handlers.WithRegistration(console.registration),
	))

	access := handlers.NewAuth(services.NewConsole(console.registration))
	mux.HandleFunc("/api/auth/estado", access.State)
	mux.HandleFunc("/api/auth/setup", access.Configure)
	mux.HandleFunc("/api/auth/login", access.Login)
	mux.HandleFunc("/api/auth/logout", access.Logout)

	mux.Handle("/api/session/register", handlers.NewAccess(console.access))
	mux.Handle("/api/session/diagnostico", handlers.NewDiagnostics(console.diagnostics))
	machines := handlers.NewMachines(console.binding)

	mux.Handle("/api/vms", machines)
	mux.HandleFunc("/api/vms/{vm}/poweron", machines.PowerOn)

	registerModules(mux, system, users, systemd, haproxy)
	history := registerHistory(mux, console.registration)

	if err := serveWeb(mux); err != nil {
		return nil, err
	}

	return handlers.WithObservability(
		handlers.WithAuthentication(
			handlers.WithHistory(mux, history, console.binding.Active),
			services.NewConsole(console.registration),
		),
		console.binding.Active,
		slog.Default(),
	), nil
}

// registerModules registra los recursos de los cuatro módulos.
func registerModules(
	mux *http.ServeMux, system *handlers.System, users *handlers.Users,
	systemd *handlers.Systemd, haproxy *handlers.HAProxy,
) {
	mux.HandleFunc("/api/packages", system.Packages)
	mux.HandleFunc("/api/packages/{name}", system.Package)
	mux.HandleFunc("/api/keys/root", system.RootKey)
	mux.HandleFunc("/api/keys/root/status", system.RootKeyStatus)
	mux.HandleFunc("/api/issue", system.Issue)
	mux.HandleFunc("/api/hostname", system.Hostname)

	mux.HandleFunc("/api/users", users.Collection)
	mux.HandleFunc("/api/users/{name}", users.User)
	mux.HandleFunc("/api/users/{name}/key", users.Key)

	mux.HandleFunc("/api/units", systemd.Units)
	mux.HandleFunc("/api/units/{unit}", systemd.Unit)
	mux.HandleFunc("/api/units/{unit}/status", systemd.Status)
	mux.HandleFunc("/api/units/{unit}/restarts", systemd.Restarts)

	mux.HandleFunc("/api/haproxy/status", haproxy.Status)
	mux.HandleFunc("/api/haproxy/setup", haproxy.Setup)
	mux.HandleFunc("/api/haproxy/backend/servers", haproxy.Servers)
	mux.HandleFunc("/api/haproxy/backend/servers/{server}", haproxy.Server)
}

// registerHistory registra el endpoint del historial.
func registerHistory(mux *http.ServeMux, store *services.Store) *services.History {
	history := services.NewHistory(store)
	mux.HandleFunc("/api/historial", handlers.NewHistory(history).ServeHTTP)

	return history
}

// serveWeb publica la interfaz web en el enrutador: la aplicación
// compilada en la raíz y la interfaz anterior mientras se retira.
func serveWeb(mux *http.ServeMux) error {
	legacy, err := fs.Sub(embeddedWeb, "web")
	if err != nil {
		// El sistema de archivos embebido es inmutable: si «web» no
		// existe, el binario se compiló mal.
		return fmt.Errorf("preparar los archivos web: %w", err)
	}

	dist, err := webui.Resolve(os.Getenv(config.EnvWeb), embeddedWeb)
	if err != nil {
		return fmt.Errorf("preparar la interfaz web: %w", err)
	}

	webui.Mount(mux, dist, legacy)

	return nil
}

// console reúne el descubrimiento de sesión y el inventario de máquinas
// que comparten los manejadores.
type console struct {
	sessions     *services.Discoverer
	binding      *services.Binding
	access       *services.Access
	diagnostics  *services.DiagnosticsService
	registration *services.Store
}

// newConsole arma el descubrimiento de sesión y el inventario con los
// adaptadores reales: VirtualBox para el estado, la dirección y el
// inventario, SSH como conector preferido y Guest Additions como respaldo.
func newConsole(ejecutor ports.Executor) *console {
	store, err := services.NewStore(config.DatabasePath(), config.CipherKeyPath())
	if err != nil {
		panic(fmt.Errorf("abrir la base de credenciales: %w", err))
	}

	machine := vbox.New(ejecutor,
		vbox.WithCredentials(store),
		vbox.WithGuestCredentials(config.GuestUser(), os.Getenv(config.EnvGuestPass)),
	)

	connectorSSH := ssh.New(ejecutor, ssh.WithKey(config.SSHKeyPath()))

	target := initialTarget(store)

	sessions := services.NewDiscoverer(&services.DiscovererConfig{
		Machine:       machine,
		CurrentTarget: target,
		Connectors: []ports.Connector{
			connectorSSH,
			machine,
		},
		Registration: store,
	})

	return &console{
		sessions: sessions,
		access: services.NewAccess(&services.AccessConfig{
			Store:         store,
			Machine:       machine,
			Guest:         machine,
			SSH:           connectorSSH,
			SSHPassword:   connectorSSH,
			Ejecutor:      ejecutor,
			PathSSHKey:    config.SSHKeyPath(),
			CurrentTarget: sessions.CurrentTarget,
		}),
		diagnostics: services.NewDiagnostics(&services.DiagnosticsConfig{
			Machine:       machine,
			Registration:  store,
			CurrentTarget: sessions.CurrentTarget,
			VersionSystem: machine.Version,
		}),
		registration: store,
		binding: services.NewBinding(&services.BindingConfig{
			Inventory:    machine,
			Selection:    sessions,
			Power:        machine,
			Machine:      machine,
			Registration: store,
			Persistence:  store,
		}),
	}
}

// initialTarget recupera la máquina vinculada o la variable de entorno.
func initialTarget(store *services.Store) domain.Target {
	target := domain.Target{ID: os.Getenv(config.EnvVM)}
	if target.ID == "" {
		target.ID = store.Active()
	}

	if target.ID != "" {
		slog.Info("máquina vinculada", "vm", target.ID)
	} else {
		slog.Warn("la consola arranca sin máquina vinculada; elígela en el selector", "variable", config.EnvVM)
	}

	return target
}

// configureLog elige el formato y el nivel de los logs de la consola.
func configureLog() {
	level := slog.LevelInfo

	switch strings.ToLower(os.Getenv(config.EnvLogLevel)) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	var handler slog.Handler
	if strings.EqualFold(os.Getenv(config.EnvLog), "json") {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	} else {
		handler = terminal.NewConsoleHandler(os.Stderr, level, terminal.ColorEnabled(os.Stderr))
	}

	slog.SetDefault(slog.New(handler))
}
