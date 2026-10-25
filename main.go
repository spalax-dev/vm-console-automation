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

// contenidoWeb contiene la interfaz: la aplicación compilada con Vite y
// la interfaz anterior, que se conserva mientras se retira. Se enumeran
// las carpetas necesarias para no embeber node_modules ni artefactos de
// desarrollo del frontend.
//
//go:embed all:web/dist all:web/modules all:web/static web/index.html
var contenidoWeb embed.FS

// Variables de entorno que configuran la conexión con la máquina.
// claveError es la clave de los errores en el log de arranque.
const claveError = "error"

const (
	envLog       = "VMCONS_LOG"
	envLogLevel  = "VMCONS_LOG_LEVEL"
	envVM        = "VMCONS_VM"         // nombre o UUID de la VM en VirtualBox
	envSSHKey    = "VMCONS_SSH_KEY"    // ruta de la llave privada en el anfitrión
	envGuestUser = "VMCONS_GUEST_USER" // usuario de Guest Additions
	// #nosec G101 -- es el nombre de la variable de entorno, no la credencial.
	envGuestPass  = "VMCONS_GUEST_PASS" // contraseña de Guest Additions
	envDireccion  = "VMCONS_ADDR"       // dirección donde escucha la consola
	envWeb        = "VMCONS_WEB"        // directorio de la interfaz web
	envDB         = "VMCONS_DB"         // ruta de la base SQLite
	direccionProd = "127.0.0.1:8080"
)

// tiempoMaximoPeticion acota la duración de cada petición HTTP.
const tiempoMaximoPeticion = 2 * time.Minute

// politicasCache declara cuánto se reutiliza cada lectura de la API. Las
// consultas a la máquina abren sesión y ejecutan órdenes, así que se
// guardan unos segundos; cualquier escritura vacía la caché completa.
func politicasCache() []httpcache.Politica {
	return []httpcache.Politica{
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
	configurarLog()

	if err := ejecutar(); err != nil {
		slog.Error("la consola terminó con error", claveError, err)
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

	if err := imprimirBanner(servidor.Addr); err != nil {
		slog.Warn("no se pudo escribir el banner de arranque", claveError, err)
	}

	if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("servir la consola en %q: %w", servidor.Addr, err)
	}

	return nil
}

// nuevoServidor arma el servidor HTTP con sus rutas y tiempos límite.
// Devuelve también la máquina vinculada, para el banner de arranque.
func nuevoServidor() (*http.Server, error) {
	manejador, err := nuevoMux()
	if err != nil {
		return nil, err
	}

	manejador = httpcache.Nuevo(politicasCache()).Envolver(manejador)

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
	mux.Handle("/api/session/diagnostico", handlers.NuevoDiagnostico(consola.diagnostico))
	maquinas := handlers.NuevoMaquinas(consola.vinculacion)

	mux.Handle("/api/vms", maquinas)
	mux.HandleFunc("/api/vms/{vm}/poweron", maquinas.Encender)

	registrarModulos(mux, sistema, usuarios, systemd, haproxy)
	historial := registrarHistorial(mux, consola.registro)

	if err := servirInterfaz(mux); err != nil {
		return nil, err
	}

	return handlers.ConObservabilidad(
		handlers.ConHistorial(mux, historial, consola.vinculacion.Activa),
		consola.vinculacion.Activa,
		slog.Default(),
	), nil
}

// registrarModulos registra los recursos de los cuatro módulos.
func registrarModulos(
	mux *http.ServeMux, sistema *handlers.Sistema, usuarios *handlers.Usuarios,
	systemd *handlers.Systemd, haproxy *handlers.Haproxy,
) {
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
}

// registrarHistorial registra el endpoint del historial.
func registrarHistorial(mux *http.ServeMux, almacen *services.Almacen) *services.Historial {
	historial := services.NuevoHistorial(almacen)
	mux.HandleFunc("/api/historial", handlers.NuevoHistorial(historial).ServeHTTP)

	return historial
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
	diagnostico *services.DiagnosticoServicio
	registro    *services.Almacen
}

// nuevaConsola arma el descubrimiento de sesión y el inventario con los
// adaptadores reales: VirtualBox para el estado, la dirección y el
// inventario, SSH como conector preferido y Guest Additions como respaldo.
func nuevaConsola(ejecutor ports.Executor) *consola {
	almacen, err := services.NuevoAlmacen(rutaBaseDatos(), rutaClaveCifrado())
	if err != nil {
		panic(fmt.Errorf("abrir la base de credenciales: %w", err))
	}

	maquina := vbox.New(ejecutor,
		vbox.WithCredenciales(almacen),
		vbox.WithGuestCredentials(usuarioInvitado(), os.Getenv(envGuestPass)),
	)

	conectorSSH := ssh.New(ejecutor, ssh.WithKey(rutaLlave()))

	objetivo := domain.Target{ID: os.Getenv(envVM)}
	if objetivo.ID == "" {
		objetivo.ID = almacen.Activa()
	}

	if objetivo.ID != "" {
		slog.Info("máquina vinculada", "vm", objetivo.ID)
	} else {
		slog.Warn("la consola arranca sin máquina vinculada; elígela en el selector", "variable", envVM)
	}

	sesiones := services.NuevoDescubridor(&services.ConfigDescubridor{
		Maquina:  maquina,
		Objetivo: objetivo,
		Conectores: []ports.Connector{
			conectorSSH,
			maquina,
		},
	})

	return &consola{
		sesiones: sesiones,
		acceso: services.NuevoAcceso(&services.ConfigAcceso{
			Almacen:     almacen,
			Maquina:     maquina,
			Guest:       maquina,
			SSH:         conectorSSH,
			SSHPassword: conectorSSH,
			Ejecutor:    ejecutor,
			RutaLlave:   rutaLlave(),
			Objetivo:    sesiones.Objetivo,
		}),
		diagnostico: services.NuevoDiagnostico(&services.ConfigDiagnostico{
			Maquina:        maquina,
			Registro:       almacen,
			Objetivo:       sesiones.Objetivo,
			VersionSistema: maquina.Version,
		}),
		registro: almacen,
		vinculacion: services.NuevoVinculacion(&services.ConfigVinculacion{
			Inventario:   maquina,
			Seleccion:    sesiones,
			Energia:      maquina,
			Maquina:      maquina,
			Registro:     almacen,
			Persistencia: almacen,
		}),
	}
}

// configurarLog elige el formato y el nivel de los logs de la consola.
func configurarLog() {
	nivel := slog.LevelInfo

	switch strings.ToLower(os.Getenv(envLogLevel)) {
	case "debug":
		nivel = slog.LevelDebug
	case "warn":
		nivel = slog.LevelWarn
	case "error":
		nivel = slog.LevelError
	}

	opciones := &slog.HandlerOptions{Level: nivel}

	var manejador slog.Handler
	if strings.EqualFold(os.Getenv(envLog), "json") {
		manejador = slog.NewJSONHandler(os.Stderr, opciones)
	} else {
		manejador = slog.NewTextHandler(os.Stderr, opciones)
	}

	slog.SetDefault(slog.New(manejador))
}

// direccion devuelve la dirección de escucha configurada.
func direccion() string {
	if valor := os.Getenv(envDireccion); valor != "" {
		return valor
	}

	return direccionProd
}

// rutaBaseDatos devuelve la ruta de la base SQLite de la consola.
func rutaBaseDatos() string {
	if valor := os.Getenv(envDB); valor != "" {
		return valor
	}

	return filepath.Join("data", "vmcons.db")
}

// rutaClaveCifrado devuelve la clave local que cifra las contraseñas.
func rutaClaveCifrado() string {
	return filepath.Join(filepath.Dir(rutaBaseDatos()), "credenciales.key")
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
