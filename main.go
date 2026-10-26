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
	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
	"consola-vm/internal/handlers"
	"consola-vm/internal/httpcache"
	"consola-vm/internal/services"
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

// Variables de entorno que configuran la conexión con la máquina.
const (
	envLog       = "VMCONS_LOG"
	envLogLevel  = "VMCONS_LOG_LEVEL"
	envVM        = "VMCONS_VM"         // nombre o UUID de la VM en VirtualBox
	envSSHKey    = "VMCONS_SSH_KEY"    // ruta de la llave privada en el anfitrión
	envGuestUser = "VMCONS_GUEST_USER" // usuario de Guest Additions
	// #nosec G101 -- es el nombre de la variable de entorno, no la credencial.
	envGuestPass = "VMCONS_GUEST_PASS" // contraseña de Guest Additions
	envAddress   = "VMCONS_ADDR"       // dirección donde escucha la consola
	envWeb       = "VMCONS_WEB"        // directorio de la interfaz web
	envDB        = "VMCONS_DB"         // ruta de la base SQLite
	addressProd  = "127.0.0.1:8080"
)

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

	if err := printBanner(server.Addr); err != nil {
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
		Addr:              address(),
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
		Path:     sshKeyPath(),
	})

	issue := services.NewIssue(sessions)
	hostname := services.NewHostname(sessions)

	system := handlers.NewSystem(services.NewPackages(sessions), keys, issue, hostname)
	users := handlers.NewUsers(services.NewUsers(services.UsersConfig{
		Sessions:  sessions,
		Ejecutor:  ejecutor,
		Directory: filepath.Join(filepath.Dir(sshKeyPath()), "users"),
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
	mux.HandleFunc("/api/issue", system.Issue)
	mux.HandleFunc("/api/hostname", system.Hostname)

	mux.HandleFunc("/api/users", users.Collection)
	mux.HandleFunc("/api/users/{name}", users.User)
	mux.HandleFunc("/api/users/{name}/key", users.Key)

	mux.HandleFunc("/api/units", systemd.Units)
	mux.HandleFunc("/api/units/{unit}", systemd.Unit)
	mux.HandleFunc("/api/units/{unit}/status", systemd.Status)
	mux.HandleFunc("/api/units/{unit}/restarts", systemd.Restarts)

	mux.HandleFunc("/api/haproxy/backend/servers", haproxy.Servers)
	mux.HandleFunc("/api/haproxy/backend/servers/{server}", haproxy.Server)
}

// registerHistory registra el endpoint del historial.
func registerHistory(mux *http.ServeMux, store *services.Store) *services.History {
	history := services.NewHistory(store)
	mux.HandleFunc("/api/historial", handlers.NewHistory(history).ServeHTTP)

	return history
}

// serveWeb publica la aplicación compilada con Vite en la raíz y
// conserva la interfaz anterior en /modules/ mientras se retira.
func serveWeb(mux *http.ServeMux) error {
	sub, err := fs.Sub(embeddedWeb, "web")
	if err != nil {
		// El sistema de archivos embebido es inmutable: si «web» no
		// existe, el binario se compiló mal.
		return fmt.Errorf("preparar los archivos web: %w", err)
	}

	dist, err := webInterface()
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
	store, err := services.NewStore(databasePath(), cipherKeyPath())
	if err != nil {
		panic(fmt.Errorf("abrir la base de credenciales: %w", err))
	}

	machine := vbox.New(ejecutor,
		vbox.WithCredentials(store),
		vbox.WithGuestCredentials(guestUser(), os.Getenv(envGuestPass)),
	)

	connectorSSH := ssh.New(ejecutor, ssh.WithKey(sshKeyPath()))

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
			PathSSHKey:    sshKeyPath(),
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
	target := domain.Target{ID: os.Getenv(envVM)}
	if target.ID == "" {
		target.ID = store.Active()
	}

	if target.ID != "" {
		slog.Info("máquina vinculada", "vm", target.ID)
	} else {
		slog.Warn("la consola arranca sin máquina vinculada; elígela en el selector", "variable", envVM)
	}

	return target
}

// configureLog elige el formato y el nivel de los logs de la consola.
func configureLog() {
	level := slog.LevelInfo

	switch strings.ToLower(os.Getenv(envLogLevel)) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	var handler slog.Handler
	if strings.EqualFold(os.Getenv(envLog), "json") {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	} else {
		handler = newConsoleHandler(os.Stderr, level, colorEnabledFor(os.Stderr))
	}

	slog.SetDefault(slog.New(handler))
}

// address devuelve la dirección de escucha configurada.
func address() string {
	if value := os.Getenv(envAddress); value != "" {
		return value
	}

	return addressProd
}

// databasePath devuelve la ruta de la base SQLite de la consola.
func databasePath() string {
	if value := os.Getenv(envDB); value != "" {
		return value
	}

	return filepath.Join("data", "vmcons.db")
}

// cipherKeyPath devuelve la clave local que cifra las contraseñas.
func cipherKeyPath() string {
	return filepath.Join(filepath.Dir(databasePath()), "credenciales.key")
}

// sshKeyPath devuelve la ruta de la llave privada de root: la variable
// de entorno o, por omisión, el directorio de configuración del usuario.
func sshKeyPath() string {
	if value := os.Getenv(envSSHKey); value != "" {
		return value
	}

	config, err := os.UserConfigDir()
	if err != nil {
		config = "."
	}

	return filepath.Join(config, "vmcons", "id_ed25519")
}

// guestUser devuelve el usuario de Guest Additions, «root» por
// omisión.
func guestUser() string {
	if value := os.Getenv(envGuestUser); value != "" {
		return value
	}

	return "root"
}
