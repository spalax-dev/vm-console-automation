// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/services"
)

// Valores repetidos en las pruebas de los handlers.
const (
	casoValido            = "válido"
	casoMetodoNoPermitido = "método no permitido"
	casoCuerpoInvalido    = "cuerpo inválido"
	unidadCatalogo        = "catalogo.service"
	casoListar            = "listar"
	casoConsultar         = "consultar"
	casoEliminar          = "eliminar"
	casoNombreInvalido    = "nombre inválido"
	usuarioJorge          = "jorge"
	usuarioJorgeMayus     = "Jorge"
	rutaUsuarioJorge      = "/api/users/jorge"
	nombreNano            = "nano"
	servidorPrueba        = "192.168.1.61:8080"
	rutaPaquetes          = "/api/packages"
	rutaPaqueteNano       = rutaPaquetes + "/" + nombreNano
	operacionInstalarNano = "apt-get install -y nano"
	rutaLlaveRoot         = "/api/keys/root"
	msgCodigoCuerpo       = "código = %d, want %d (cuerpo %s)"
	rutaIssue             = "/api/issue"
	rutaHostname          = "/api/hostname"
	hostDebian            = "debian13"
	textoBienvenido       = "Bienvenido"
)

// stubPaquetes es un servicio de paquetes falso con resultados
// preprogramados.
type stubPaquetes struct {
	lista     []services.Paquete
	resultado services.Resultado
	err       error
	nombre    string
}

// Listar devuelve la lista preprogramada.
func (s *stubPaquetes) Listar(context.Context) ([]services.Paquete, error) {
	return s.lista, s.err
}

// Instalar registra el nombre y devuelve el resultado preprogramado.
func (s *stubPaquetes) Instalar(_ context.Context, nombre string) (services.Resultado, error) {
	s.nombre = nombre

	return s.resultado, s.err
}

// Eliminar registra el nombre y devuelve el resultado preprogramado.
func (s *stubPaquetes) Eliminar(_ context.Context, nombre string) (services.Resultado, error) {
	s.nombre = nombre

	return s.resultado, s.err
}

// stubLlaves es un servicio de llaves falso con resultados
// preprogramados.
type stubLlaves struct {
	resultado   services.Resultado
	err         error
	nombre      string
	contenido   []byte
	descargaErr error
	generadas   int
}

// Generar registra la llamada y devuelve el resultado preprogramado.
func (s *stubLlaves) Generar(context.Context) (services.Resultado, error) {
	s.generadas++

	return s.resultado, s.err
}

// Descargar devuelve la llave preprogramada.
func (s *stubLlaves) Descargar() (nombre string, contenido []byte, err error) {
	return s.nombre, s.contenido, s.descargaErr
}

// stubIssue es un servicio de /etc/issue falso con resultados
// preprogramados.
type stubIssue struct {
	config     services.ConfiguracionIssue
	resultado  services.Resultado
	err        error
	escrito    services.ConfiguracionIssue
	escrituras int
}

// Obtener devuelve la configuración preprogramada.
func (s *stubIssue) Obtener(context.Context) (services.ConfiguracionIssue, error) {
	return s.config, s.err
}

// Escribir registra la configuración y devuelve el resultado.
func (s *stubIssue) Escribir(_ context.Context, config services.ConfiguracionIssue) (services.Resultado, error) {
	s.escrito = config
	s.escrituras++

	return s.resultado, s.err
}

// stubHostname es un servicio de hostname falso con resultados
// preprogramados.
type stubHostname struct {
	config     services.ConfiguracionHostname
	resultado  services.Resultado
	err        error
	escrito    services.ConfiguracionHostname
	escrituras int
}

// Obtener devuelve la configuración preprogramada.
func (s *stubHostname) Obtener(context.Context) (services.ConfiguracionHostname, error) {
	return s.config, s.err
}

// Escribir registra la configuración y devuelve el resultado.
func (s *stubHostname) Escribir(_ context.Context, config services.ConfiguracionHostname) (services.Resultado, error) {
	s.escrito = config
	s.escrituras++

	return s.resultado, s.err
}

// nuevoSistemaPrueba arma el handler de sistema con los servicios
// falsos.
func nuevoSistemaPrueba() *Sistema {
	return NuevoSistema(&stubPaquetes{}, &stubLlaves{}, &stubIssue{}, &stubHostname{})
}

// casoPeticion describe una petición de prueba contra un handler.
type casoPeticion struct {
	// nombre identifica el caso en los mensajes de error.
	nombre string
	// metodo es el método HTTP de la petición.
	metodo string
	// cuerpo es el JSON enviado, vacío si no aplica.
	cuerpo string
	// codigo es el estado HTTP esperado.
	codigo int
	// campo es el campo de validación esperado en un 400, si aplica.
	campo string
}

// ejecutarCaso lanza una petición contra el handler y devuelve la
// respuesta grabada.
func ejecutarCaso(t *testing.T, handler http.HandlerFunc, ruta string, caso casoPeticion) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), caso.metodo, ruta, strings.NewReader(caso.cuerpo))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler(rec, req)

	return rec
}

// ejecutarCasoConPath lanza una petición con un parámetro de ruta.
func ejecutarCasoConPath(
	t *testing.T, handler http.HandlerFunc, ruta, clave, valor string, caso casoPeticion,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), caso.metodo, ruta, strings.NewReader(caso.cuerpo))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue(clave, valor)

	rec := httptest.NewRecorder()
	handler(rec, req)

	return rec
}

// comprobarCaso valida el código HTTP y, en los errores de validación,
// que el cuerpo nombre el campo rechazado.
func comprobarCaso(t *testing.T, rec *httptest.ResponseRecorder, caso casoPeticion) {
	t.Helper()

	if rec.Code != caso.codigo {
		t.Fatalf("%s: código = %d, want %d (cuerpo %s)", caso.nombre, rec.Code, caso.codigo, rec.Body.String())
	}

	if caso.campo == "" {
		return
	}

	var cuerpo map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf("%s: respuesta no es JSON válido: %v", caso.nombre, err)
	}

	if cuerpo["campo"] != caso.campo {
		t.Errorf("%s: campo = %v, want %q", caso.nombre, cuerpo["campo"], caso.campo)
	}

	if cuerpo["ok"] != false {
		t.Errorf("%s: ok = %v, want false", caso.nombre, cuerpo["ok"])
	}
}

// TestPaquetesListar cubre el listado de paquetes instalados.
func TestPaquetesListar(t *testing.T) {
	t.Parallel()

	servicio := &stubPaquetes{lista: []services.Paquete{{Nombre: nombreNano, Version: "7.2-1"}}}
	rec := ejecutarCaso(t, NuevoSistema(servicio, &stubLlaves{}, &stubIssue{}, &stubHostname{}).Paquetes, rutaPaquetes, casoPeticion{
		nombre: casoListar, metodo: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigoCuerpo, rec.Code, http.StatusOK, rec.Body.String())
	}

	var cuerpo struct {
		Paquetes []paqueteRespuesta `json:"paquetes"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if len(cuerpo.Paquetes) != 1 || cuerpo.Paquetes[0].Nombre != nombreNano {
		t.Errorf("paquetes = %+v, want [%s]", cuerpo.Paquetes, nombreNano)
	}
}

// TestPaquetesInstalar cubre la instalación de un paquete.
func TestPaquetesInstalar(t *testing.T) {
	t.Parallel()

	servicio := &stubPaquetes{resultado: services.Resultado{
		Mensaje:   "Paquete «nano» instalado.",
		Operacion: operacionInstalarNano,
	}}

	rec := ejecutarCaso(t, NuevoSistema(servicio, &stubLlaves{}, &stubIssue{}, &stubHostname{}).Paquetes, rutaPaquetes, casoPeticion{
		nombre: casoValido, metodo: http.MethodPost, cuerpo: `{"nombre":"nano"}`,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf(msgCodigoCuerpo, rec.Code, http.StatusCreated, rec.Body.String())
	}

	if got := rec.Header().Get("Location"); got != rutaPaqueteNano {
		t.Errorf("Location = %q, want %q", got, rutaPaqueteNano)
	}

	if servicio.nombre != nombreNano {
		t.Errorf("nombre = %q, want %q", servicio.nombre, nombreNano)
	}

	var cuerpo map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if cuerpo["ok"] != true {
		t.Errorf("ok = %v, want true", cuerpo["ok"])
	}
}

// TestPaquetesValidacion cubre los errores de entrada de la colección.
func TestPaquetesValidacion(t *testing.T) {
	t.Parallel()

	handler := nuevoSistemaPrueba().Paquetes
	casos := []casoPeticion{
		{nombre: "nombre con shell", metodo: http.MethodPost, cuerpo: `{"nombre":"nano; rm -rf /"}`, codigo: http.StatusBadRequest, campo: campoNombre},
		{nombre: casoCuerpoInvalido, metodo: http.MethodPost, cuerpo: `{`, codigo: http.StatusBadRequest},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodPut, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, rutaPaquetes, caso)
		comprobarCaso(t, rec, caso)
	}
}

// TestPaquetesErrorDeOrden cubre el fallo de APT con código distinto de
// cero.
func TestPaquetesErrorDeOrden(t *testing.T) {
	t.Parallel()

	servicio := &stubPaquetes{
		resultado: services.Resultado{
			Operacion: operacionInstalarNano,
			Stderr:    "E: Unable to locate package nano",
			Codigo:    100,
		},
		err: &domain.ExitError{Code: 100, Command: operacionInstalarNano},
	}

	rec := ejecutarCaso(t, NuevoSistema(servicio, &stubLlaves{}, &stubIssue{}, &stubHostname{}).Paquetes, rutaPaquetes, casoPeticion{
		nombre: "fallo de APT", metodo: http.MethodPost, cuerpo: `{"nombre":"nano"}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}

	var cuerpo map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if cuerpo["codigo"] != float64(100) {
		t.Errorf("codigo = %v, want 100", cuerpo["codigo"])
	}

	if cuerpo["stderr"] != "E: Unable to locate package nano" {
		t.Errorf("stderr = %v, want el stderr de APT", cuerpo["stderr"])
	}
}

// TestPaquetesSinSesion cubre la ausencia de sesión con la máquina.
func TestPaquetesSinSesion(t *testing.T) {
	t.Parallel()

	servicio := &stubPaquetes{err: services.ErrSesionNoDisponible}

	rec := ejecutarCaso(t, NuevoSistema(servicio, &stubLlaves{}, &stubIssue{}, &stubHostname{}).Paquetes, rutaPaquetes, casoPeticion{
		nombre: "sin sesión", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(msgCodigo, rec.Code, http.StatusServiceUnavailable)
	}
}

// TestPaquete cubre la eliminación de un paquete por su nombre.
func TestPaquete(t *testing.T) {
	t.Parallel()

	handler := nuevoSistemaPrueba().Paquete
	casos := []struct {
		casoPeticion
		nombre string
	}{
		{casoPeticion{nombre: casoEliminar, metodo: http.MethodDelete, codigo: http.StatusNoContent}, nombreNano},
		{casoPeticion{nombre: casoNombreInvalido, metodo: http.MethodDelete, codigo: http.StatusBadRequest, campo: campoName}, "nano; rm"},
		{casoPeticion{nombre: casoMetodoNoPermitido, metodo: http.MethodGet, codigo: http.StatusMethodNotAllowed}, nombreNano},
	}

	for _, caso := range casos {
		rec := ejecutarCasoConPath(t, handler, "/api/packages/"+nombreNano, campoName, caso.nombre, caso.casoPeticion)
		comprobarCaso(t, rec, caso.casoPeticion)
	}
}

// TestPaqueteEliminar comprueba que el nombre de la ruta llega al
// servicio y que la respuesta no lleva cuerpo.
func TestPaqueteEliminar(t *testing.T) {
	t.Parallel()

	servicio := &stubPaquetes{}
	handler := NuevoSistema(servicio, &stubLlaves{}, &stubIssue{}, &stubHostname{}).Paquete

	rec := ejecutarCasoConPath(t, handler, rutaPaqueteNano, campoName, nombreNano, casoPeticion{
		nombre: casoEliminar, metodo: http.MethodDelete,
	})

	if rec.Code != http.StatusNoContent {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNoContent)
	}

	if servicio.nombre != nombreNano {
		t.Errorf("nombre = %q, want %q", servicio.nombre, nombreNano)
	}

	if rec.Body.Len() != 0 {
		t.Errorf("cuerpo = %q, want vacío", rec.Body.String())
	}
}

// TestLlaveRootGenerar cubre la generación del par de llaves de root.
func TestLlaveRootGenerar(t *testing.T) {
	t.Parallel()

	servicio := &stubLlaves{resultado: services.Resultado{
		Mensaje: "Llave SSH de root generada e instalada.",
	}}

	rec := ejecutarCaso(t, NuevoSistema(&stubPaquetes{}, servicio, &stubIssue{}, &stubHostname{}).LlaveRoot, rutaLlaveRoot, casoPeticion{
		nombre: "generar", metodo: http.MethodPost,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf(msgCodigoCuerpo, rec.Code, http.StatusCreated, rec.Body.String())
	}

	if got := rec.Header().Get("Location"); got != rutaLlaveRoot {
		t.Errorf("Location = %q, want /api/keys/root", got)
	}

	if servicio.generadas != 1 {
		t.Errorf("Generar() se llamó %d veces, want 1", servicio.generadas)
	}
}

// TestLlaveRootGenerarExistente cubre el conflicto de una llave que ya
// existe.
func TestLlaveRootGenerarExistente(t *testing.T) {
	t.Parallel()

	servicio := &stubLlaves{err: services.ErrLlaveExistente}

	rec := ejecutarCaso(t, NuevoSistema(&stubPaquetes{}, servicio, &stubIssue{}, &stubHostname{}).LlaveRoot, rutaLlaveRoot, casoPeticion{
		nombre: "llave existente", metodo: http.MethodPost,
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf(msgCodigo, rec.Code, http.StatusConflict)
	}
}

// TestLlaveRootDescargar cubre la descarga de la llave privada.
func TestLlaveRootDescargar(t *testing.T) {
	t.Parallel()

	servicio := &stubLlaves{nombre: "id_ed25519", contenido: []byte("privada")}

	rec := ejecutarCaso(t, NuevoSistema(&stubPaquetes{}, servicio, &stubIssue{}, &stubHostname{}).LlaveRoot, rutaLlaveRoot, casoPeticion{
		nombre: "descargar", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="id_ed25519"` {
		t.Errorf("Content-Disposition = %q", got)
	}

	if rec.Body.String() != "privada" {
		t.Errorf("cuerpo = %q, want la llave privada", rec.Body.String())
	}
}

// TestLlaveRootSinLlave cubre la descarga cuando no hay llave.
func TestLlaveRootSinLlave(t *testing.T) {
	t.Parallel()

	servicio := &stubLlaves{descargaErr: services.ErrLlaveNoEncontrada}

	rec := ejecutarCaso(t, NuevoSistema(&stubPaquetes{}, servicio, &stubIssue{}, &stubHostname{}).LlaveRoot, rutaLlaveRoot, casoPeticion{
		nombre: "sin llave", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNotFound)
	}
}

// TestLlaveRootMetodo cubre el método no permitido.
func TestLlaveRootMetodo(t *testing.T) {
	t.Parallel()

	rec := ejecutarCaso(t, nuevoSistemaPrueba().LlaveRoot, rutaLlaveRoot, casoPeticion{
		nombre: casoMetodoNoPermitido, metodo: http.MethodPut,
	})

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(msgCodigo, rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestIssueObtener cubre la lectura de /etc/issue.
func TestIssueObtener(t *testing.T) {
	t.Parallel()

	servicio := &stubIssue{config: services.ConfiguracionIssue{
		Mensaje:    textoBienvenido,
		NombreHost: true,
		IP:         true,
		Contenido:  "Bienvenido\nHostname: \\n\nIP: \\4\n",
	}}

	rec := ejecutarCaso(t, NuevoSistema(&stubPaquetes{}, &stubLlaves{}, servicio, &stubHostname{}).Issue, rutaIssue, casoPeticion{
		nombre: "consultar", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	var cuerpo map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if cuerpo["mensaje"] != textoBienvenido {
		t.Errorf("mensaje = %v, want Bienvenido", cuerpo["mensaje"])
	}

	if cuerpo["nombre_host"] != true || cuerpo["ip"] != true {
		t.Errorf("opciones = %v, want nombre_host e ip activos", cuerpo)
	}
}

// TestIssueEscribir cubre el reemplazo de /etc/issue.
func TestIssueEscribir(t *testing.T) {
	t.Parallel()

	servicio := &stubIssue{resultado: services.Resultado{Mensaje: "Entrada del sistema actualizada."}}

	rec := ejecutarCaso(t, NuevoSistema(&stubPaquetes{}, &stubLlaves{}, servicio, &stubHostname{}).Issue, rutaIssue, casoPeticion{
		nombre: casoValido,
		metodo: http.MethodPut,
		cuerpo: `{"mensaje":"Bienvenido","nombre_host":true,"fecha":true,"hora":true,"ip":true}`,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	if servicio.escrituras != 1 {
		t.Errorf("Escribir() se llamó %d veces, want 1", servicio.escrituras)
	}

	if servicio.escrito.Mensaje != textoBienvenido || !servicio.escrito.NombreHost {
		t.Errorf("configuración = %+v, want mensaje y nombre_host", servicio.escrito)
	}
}

// TestIssueValidacion cubre las reglas del mensaje y de las opciones.
func TestIssueValidacion(t *testing.T) {
	t.Parallel()

	handler := nuevoSistemaPrueba().Issue
	casos := []casoPeticion{
		{
			nombre: "mensaje largo",
			metodo: http.MethodPut,
			cuerpo: `{"mensaje":"` + strings.Repeat("a", 600) + `","nombre_host":false,"fecha":false,"hora":false,"ip":false}`,
			codigo: http.StatusBadRequest,
			campo:  campoMensaje,
		},
		{
			nombre: "mensaje con salto de línea",
			metodo: http.MethodPut,
			cuerpo: `{"mensaje":"linea1\nlinea2","nombre_host":false,"fecha":false,"hora":false,"ip":false}`,
			codigo: http.StatusBadRequest,
			campo:  campoMensaje,
		},
		{
			nombre: "sin mensaje ni opciones",
			metodo: http.MethodPut,
			cuerpo: `{"mensaje":"","nombre_host":false,"fecha":false,"hora":false,"ip":false}`,
			codigo: http.StatusBadRequest,
			campo:  campoMensaje,
		},
		{nombre: casoCuerpoInvalido, metodo: http.MethodPut, cuerpo: `{`, codigo: http.StatusBadRequest},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodPost, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, rutaIssue, caso)
		comprobarCaso(t, rec, caso)
	}
}

// TestIssueErrorDeOrden cubre el fallo al escribir /etc/issue.
func TestIssueErrorDeOrden(t *testing.T) {
	t.Parallel()

	servicio := &stubIssue{
		resultado: services.Resultado{Codigo: 1, Stderr: "Read-only file system"},
		err:       &domain.ExitError{Code: 1, Command: "sh"},
	}

	rec := ejecutarCaso(t, NuevoSistema(&stubPaquetes{}, &stubLlaves{}, servicio, &stubHostname{}).Issue, rutaIssue, casoPeticion{
		nombre: "fallo de escritura", metodo: http.MethodPut, cuerpo: `{"mensaje":"hola"}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestHostnameObtener cubre la lectura del nombre de host.
func TestHostnameObtener(t *testing.T) {
	t.Parallel()

	servicio := &stubHostname{config: services.ConfiguracionHostname{
		Nombre:          hostDebian,
		IP:              "192.168.1.50",
		ActualizarHosts: true,
	}}

	rec := ejecutarCaso(t, NuevoSistema(&stubPaquetes{}, &stubLlaves{}, &stubIssue{}, servicio).Hostname, rutaHostname, casoPeticion{
		nombre: "consultar", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	var cuerpo map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if cuerpo["nombre"] != hostDebian {
		t.Errorf("nombre = %v, want debian13", cuerpo["nombre"])
	}

	if cuerpo["ip"] != "192.168.1.50" {
		t.Errorf("ip = %v, want 192.168.1.50", cuerpo["ip"])
	}

	if cuerpo["actualizar_hosts"] != true {
		t.Errorf("actualizar_hosts = %v, want true", cuerpo["actualizar_hosts"])
	}
}

// TestHostnameEscribir cubre el reemplazo del nombre de host.
func TestHostnameEscribir(t *testing.T) {
	t.Parallel()

	servicio := &stubHostname{resultado: services.Resultado{Mensaje: "Nombre de host actualizado."}}

	rec := ejecutarCaso(t, NuevoSistema(&stubPaquetes{}, &stubLlaves{}, &stubIssue{}, servicio).Hostname, rutaHostname, casoPeticion{
		nombre: casoValido,
		metodo: http.MethodPut,
		cuerpo: `{"nombre":"nodo-01","ip":"192.168.1.60","actualizar_hosts":true}`,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	if servicio.escrituras != 1 {
		t.Errorf("Escribir() se llamó %d veces, want 1", servicio.escrituras)
	}

	if servicio.escrito.Nombre != "nodo-01" || !servicio.escrito.ActualizarHosts {
		t.Errorf("configuración = %+v, want nodo-01 con hosts", servicio.escrito)
	}
}

// TestHostnameValidacion cubre el nombre RFC 1123 y la dirección IP.
func TestHostnameValidacion(t *testing.T) {
	t.Parallel()

	handler := nuevoSistemaPrueba().Hostname
	casos := []casoPeticion{
		{nombre: "nombre con espacio", metodo: http.MethodPut, cuerpo: `{"nombre":"mi equipo","ip":"192.168.1.60"}`, codigo: http.StatusBadRequest, campo: campoNombre},
		{nombre: "IP inválida", metodo: http.MethodPut, cuerpo: `{"nombre":"nodo-web-01","ip":"999.1.1.1"}`, codigo: http.StatusBadRequest, campo: campoIP},
		{nombre: casoCuerpoInvalido, metodo: http.MethodPut, cuerpo: `{`, codigo: http.StatusBadRequest},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodPost, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, rutaHostname, caso)
		comprobarCaso(t, rec, caso)
	}
}

// TestHostnameErrorDeOrden cubre el fallo al cambiar el nombre.
func TestHostnameErrorDeOrden(t *testing.T) {
	t.Parallel()

	servicio := &stubHostname{
		resultado: services.Resultado{Codigo: 127, Stderr: "hostnamectl: not found"},
		err:       &domain.ExitError{Code: 127, Command: "hostnamectl"},
	}

	rec := ejecutarCaso(t, NuevoSistema(&stubPaquetes{}, &stubLlaves{}, &stubIssue{}, servicio).Hostname, rutaHostname, casoPeticion{
		nombre: "fallo de hostnamectl", metodo: http.MethodPut, cuerpo: `{"nombre":"nodo-01","ip":"192.168.1.60"}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestPaqueteErrorDeOrden cubre el fallo del servicio al eliminar.
func TestPaqueteErrorDeOrden(t *testing.T) {
	t.Parallel()

	servicio := &stubPaquetes{err: &domain.ExitError{Code: 100, Command: "apt-get remove"}}
	handler := NuevoSistema(servicio, &stubLlaves{}, &stubIssue{}, &stubHostname{}).Paquete

	rec := ejecutarCasoConPath(t, handler, "/api/packages/nano", campoName, nombreNano, casoPeticion{
		nombre: "fallo de APT", metodo: http.MethodDelete,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestLlaveRootErrorGenerico cubre un fallo inesperado al generar la
// llave.
func TestLlaveRootErrorGenerico(t *testing.T) {
	t.Parallel()

	servicio := &stubLlaves{err: errors.New("fallo inesperado")}
	handler := NuevoSistema(&stubPaquetes{}, servicio, &stubIssue{}, &stubHostname{}).LlaveRoot

	rec := ejecutarCaso(t, handler, rutaLlaveRoot, casoPeticion{
		nombre: "error genérico", metodo: http.MethodPost,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestLlaveRootErrorDeDescarga cubre un fallo inesperado al leer la
// llave privada.
func TestLlaveRootErrorDeDescarga(t *testing.T) {
	t.Parallel()

	servicio := &stubLlaves{descargaErr: errors.New("fallo de lectura")}
	handler := NuevoSistema(&stubPaquetes{}, servicio, &stubIssue{}, &stubHostname{}).LlaveRoot

	rec := ejecutarCaso(t, handler, rutaLlaveRoot, casoPeticion{
		nombre: "error de descarga", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}
