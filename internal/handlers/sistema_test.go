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
	testCaseValid              = "válido"
	testCaseMethodNotPermitido = "método no permitido"
	testCaseBodyInvalid        = "cuerpo inválido"
	unitCatalogo               = "catalogo.service"
	testCaseList               = "listar"
	testCaseGet                = "consultar"
	testCaseDelete             = "eliminar"
	testCaseNameInvalid        = "nombre inválido"
	testCaseUserInvalid        = "usuario inválido"
	userJorge                  = "jorge"
	userJorgeMayus             = "Jorge"
	pathUserJorge              = "/api/users/jorge"
	nameNano                   = "nano"
	serverTests                = "192.168.1.61:8080"
	pathPackages               = "/api/packages"
	pathPkgNano                = pathPackages + "/" + nameNano
	operationInstallNano       = "apt-get install -y nano"
	pathSSHKeyRoot             = "/api/keys/root"
	msgCodeBody                = "código = %d, want %d (cuerpo %s)"
	pathIssue                  = "/api/issue"
	pathHostname               = "/api/hostname"
	hostTest                   = "vm1"
	pathUsers                  = "/api/users"
	sshKeyPrivateTests         = "privada"
	testCaseInexistente        = "inexistente"
	stateActiveTests           = "active"
	propietarioCatalogo        = "catalogo"
	pathUnit                   = "/api/units/x"
	pathRestarts               = "/api/units/x/restarts"
	testCaseErrorRead          = "error de lectura"
	testCaseUnitInvalida       = "unidad inválida"
	pathServers                = "/api/haproxy/backend/servers"
	pathServer                 = pathServers + "/x"
	ipServer                   = "192.168.1.61"
	bodyServer                 = `{"direccion":"192.168.1.61","puerto":8080}`
	bodyActivar                = `{"activo":true}`
	textBienvenido             = "Bienvenido"
)

// stubPackages es un servicio de paquetes falso con resultados
// preprogramados.
type stubPackages struct {
	lists  []services.Package
	result services.Outcome
	err    error
	name   string
}

// List devuelve la lista preprogramada.
func (s *stubPackages) List(context.Context) ([]services.Package, error) {
	return s.lists, s.err
}

// Install registra el nombre y devuelve el resultado preprogramado.
func (s *stubPackages) Install(_ context.Context, name string) (services.Outcome, error) {
	s.name = name

	return s.result, s.err
}

// Delete registra el nombre y devuelve el resultado preprogramado.
func (s *stubPackages) Delete(_ context.Context, name string) (services.Outcome, error) {
	s.name = name

	return s.result, s.err
}

// stubKeys es un servicio de llaves falso con resultados
// preprogramados.
type stubKeys struct {
	exists       bool
	result       services.Outcome
	err          error
	name         string
	content      []byte
	downloadsErr error
	generadas    int
}

// Generate registra la llamada y devuelve el resultado preprogramado.
func (s *stubKeys) Generate(context.Context) (services.Outcome, error) {
	s.generadas++

	return s.result, s.err
}

// Exists informa si la llave está en el anfitrión.
func (s *stubKeys) Exists() bool {
	return s.exists
}

// Download devuelve la llave preprogramada.
func (s *stubKeys) Download() (name string, content []byte, err error) {
	return s.name, s.content, s.downloadsErr
}

// stubIssue es un servicio de /etc/issue falso con resultados
// preprogramados.
type stubIssue struct {
	config     services.IssueConfig
	result     services.Outcome
	err        error
	escrito    services.IssueConfig
	escrituras int
}

// Get devuelve la configuración preprogramada.
func (s *stubIssue) Get(context.Context) (services.IssueConfig, error) {
	return s.config, s.err
}

// Write registra la configuración y devuelve el resultado.
func (s *stubIssue) Write(_ context.Context, config services.IssueConfig) (services.Outcome, error) {
	s.escrito = config
	s.escrituras++

	return s.result, s.err
}

// stubHostname es un servicio de hostname falso con resultados
// preprogramados.
type stubHostname struct {
	config     services.HostnameConfig
	result     services.Outcome
	err        error
	escrito    services.HostnameConfig
	escrituras int
}

// Get devuelve la configuración preprogramada.
func (s *stubHostname) Get(context.Context) (services.HostnameConfig, error) {
	return s.config, s.err
}

// Write registra la configuración y devuelve el resultado.
func (s *stubHostname) Write(_ context.Context, config services.HostnameConfig) (services.Outcome, error) {
	s.escrito = config
	s.escrituras++

	return s.result, s.err
}

// newSystemTests arma el handler de sistema con los servicios
// falsos.
func newSystemTests() *System {
	return NewSystem(&stubPackages{}, &stubKeys{}, &stubIssue{}, &stubHostname{})
}

// testCaseRequest describe una petición de prueba contra un handler.
type testCaseRequest struct {
	// name identifica el caso en los mensajes de error.
	name string
	// method es el método HTTP de la petición.
	method string
	// body es el JSON enviado, vacío si no aplica.
	body string
	// code es el estado HTTP esperado.
	code int
	// field es el campo de validación esperado en un 400, si aplica.
	field string
}

// runTestCase lanza una petición contra el handler y devuelve la
// respuesta grabada.
func runTestCase(t *testing.T, handler http.HandlerFunc, path string, testCase testCaseRequest) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), testCase.method, path, strings.NewReader(testCase.body))
	req.Header.Set(headerContentType, kindJSONTests)

	rec := httptest.NewRecorder()
	handler(rec, req)

	return rec
}

// runTestCaseWithPath lanza una petición con un parámetro de ruta.
func runTestCaseWithPath(
	t *testing.T, handler http.HandlerFunc, path, key, value string, testCase testCaseRequest,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), testCase.method, path, strings.NewReader(testCase.body))
	req.Header.Set(headerContentType, kindJSONTests)
	req.SetPathValue(key, value)

	rec := httptest.NewRecorder()
	handler(rec, req)

	return rec
}

// checkTestCase valida el código HTTP y, en los errores de validación,
// que el cuerpo nombre el campo rechazado.
func checkTestCase(t *testing.T, rec *httptest.ResponseRecorder, testCase testCaseRequest) {
	t.Helper()

	if rec.Code != testCase.code {
		t.Fatalf("%s: código = %d, want %d (cuerpo %s)", testCase.name, rec.Code, testCase.code, rec.Body.String())
	}

	if testCase.field == "" {
		return
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("%s: respuesta no es JSON válido: %v", testCase.name, err)
	}

	if body["campo"] != testCase.field {
		t.Errorf("%s: campo = %v, want %q", testCase.name, body["campo"], testCase.field)
	}

	if body["ok"] != false {
		t.Errorf("%s: ok = %v, want false", testCase.name, body["ok"])
	}
}

// TestPackagesList cubre el listado de paquetes instalados.
func TestPackagesList(t *testing.T) {
	t.Parallel()

	service := &stubPackages{lists: []services.Package{{Name: nameNano, Version: "7.2-1"}}}
	rec := runTestCase(t, NewSystem(service, &stubKeys{}, &stubIssue{}, &stubHostname{}).Packages, pathPackages, testCaseRequest{
		name: testCaseList, method: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodeBody, rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Packages []packageResponse `json:"paquetes"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if len(body.Packages) != 1 || body.Packages[0].Name != nameNano {
		t.Errorf("paquetes = %+v, want [%s]", body.Packages, nameNano)
	}
}

// TestPackagesInstall cubre la instalación de un paquete.
func TestPackagesInstall(t *testing.T) {
	t.Parallel()

	service := &stubPackages{result: services.Outcome{
		Message:   "Paquete «nano» instalado.",
		Operation: operationInstallNano,
	}}

	rec := runTestCase(t, NewSystem(service, &stubKeys{}, &stubIssue{}, &stubHostname{}).Packages, pathPackages, testCaseRequest{
		name: testCaseValid, method: http.MethodPost, body: `{"nombre":"nano"}`,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf(msgCodeBody, rec.Code, http.StatusCreated, rec.Body.String())
	}

	if got := rec.Header().Get(headerLocation); got != pathPkgNano {
		t.Errorf("Location = %q, want %q", got, pathPkgNano)
	}

	if service.name != nameNano {
		t.Errorf("nombre = %q, want %q", service.name, nameNano)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if body["ok"] != true {
		t.Errorf("ok = %v, want true", body["ok"])
	}
}

// TestPackagesValidacion cubre los errores de entrada de la colección.
func TestPackagesValidacion(t *testing.T) {
	t.Parallel()

	handler := newSystemTests().Packages
	testCases := []testCaseRequest{
		{name: "nombre con shell", method: http.MethodPost, body: `{"nombre":"nano; rm -rf /"}`, code: http.StatusBadRequest, field: nameField},
		{name: testCaseBodyInvalid, method: http.MethodPost, body: `{`, code: http.StatusBadRequest},
		{name: testCaseMethodNotPermitido, method: http.MethodPut, code: http.StatusMethodNotAllowed},
	}

	for _, testCase := range testCases {
		rec := runTestCase(t, handler, pathPackages, testCase)
		checkTestCase(t, rec, testCase)
	}
}

// TestPackagesErrorOfCommand cubre el fallo de APT con código distinto de
// cero.
func TestPackagesErrorOfCommand(t *testing.T) {
	t.Parallel()

	service := &stubPackages{
		result: services.Outcome{
			Operation: operationInstallNano,
			Stderr:    "E: Unable to locate package nano",
			Code:      100,
		},
		err: &domain.ExitError{Code: 100, Command: operationInstallNano},
	}

	rec := runTestCase(t, NewSystem(service, &stubKeys{}, &stubIssue{}, &stubHostname{}).Packages, pathPackages, testCaseRequest{
		name: "fallo de APT", method: http.MethodPost, body: `{"nombre":"nano"}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if body["codigo"] != float64(100) {
		t.Errorf("codigo = %v, want 100", body["codigo"])
	}

	if body["stderr"] != "E: Unable to locate package nano" {
		t.Errorf("stderr = %v, want el stderr de APT", body["stderr"])
	}
}

// TestPackagesWithoutSession cubre la ausencia de sesión con la máquina.
func TestPackagesWithoutSession(t *testing.T) {
	t.Parallel()

	service := &stubPackages{err: services.ErrSessionUnavailable}

	rec := runTestCase(t, NewSystem(service, &stubKeys{}, &stubIssue{}, &stubHostname{}).Packages, pathPackages, testCaseRequest{
		name: "sin sesión", method: http.MethodGet,
	})

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(msgCode, rec.Code, http.StatusServiceUnavailable)
	}
}

// TestPkg cubre la eliminación de un paquete por su nombre.
func TestPkg(t *testing.T) {
	t.Parallel()

	handler := newSystemTests().Package
	testCases := []struct {
		testCaseRequest
		name string
	}{
		{testCaseRequest{name: testCaseDelete, method: http.MethodDelete, code: http.StatusNoContent}, nameNano},
		{testCaseRequest{name: testCaseNameInvalid, method: http.MethodDelete, code: http.StatusBadRequest, field: nameParam}, "nano; rm"},
		{testCaseRequest{name: testCaseMethodNotPermitido, method: http.MethodGet, code: http.StatusMethodNotAllowed}, nameNano},
	}

	for _, testCase := range testCases {
		rec := runTestCaseWithPath(t, handler, "/api/packages/"+nameNano, nameParam, testCase.name, testCase.testCaseRequest)
		checkTestCase(t, rec, testCase.testCaseRequest)
	}
}

// TestPkgDelete comprueba que el nombre de la ruta llega al
// servicio y que la respuesta no lleva cuerpo.
func TestPkgDelete(t *testing.T) {
	t.Parallel()

	service := &stubPackages{}
	handler := NewSystem(service, &stubKeys{}, &stubIssue{}, &stubHostname{}).Package

	rec := runTestCaseWithPath(t, handler, pathPkgNano, nameParam, nameNano, testCaseRequest{
		name: testCaseDelete, method: http.MethodDelete,
	})

	if rec.Code != http.StatusNoContent {
		t.Fatalf(msgCode, rec.Code, http.StatusNoContent)
	}

	if service.name != nameNano {
		t.Errorf("nombre = %q, want %q", service.name, nameNano)
	}

	if rec.Body.Len() != 0 {
		t.Errorf("cuerpo = %q, want vacío", rec.Body.String())
	}
}

// TestSSHKeyRootStatus informa si la llave de conexión existe.
func TestSSHKeyRootStatus(t *testing.T) {
	t.Parallel()

	service := &stubKeys{exists: true}
	rec := runTestCase(t, NewSystem(&stubPackages{}, service, &stubIssue{}, &stubHostname{}).RootKeyStatus,
		pathSSHKeyRoot+"/status", testCaseRequest{name: "estado de la llave", method: http.MethodGet, code: http.StatusOK})

	if !strings.Contains(rec.Body.String(), `"existe":true`) {
		t.Errorf("cuerpo = %q, want existe=true", rec.Body.String())
	}
}

// TestSSHKeyRootGenerate cubre la generación del par de llaves de root.
func TestSSHKeyRootGenerate(t *testing.T) {
	t.Parallel()

	service := &stubKeys{result: services.Outcome{
		Message: "Llave SSH de root generada e instalada.",
	}}

	rec := runTestCase(t, NewSystem(&stubPackages{}, service, &stubIssue{}, &stubHostname{}).RootKey, pathSSHKeyRoot, testCaseRequest{
		name: "generar", method: http.MethodPost,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf(msgCodeBody, rec.Code, http.StatusCreated, rec.Body.String())
	}

	if got := rec.Header().Get(headerLocation); got != pathSSHKeyRoot {
		t.Errorf("Location = %q, want /api/keys/root", got)
	}

	if service.generadas != 1 {
		t.Errorf("Generar() se llamó %d veces, want 1", service.generadas)
	}
}

// TestSSHKeyRootGenerateExisting cubre el conflicto de una llave que ya
// existe.
func TestSSHKeyRootGenerateExisting(t *testing.T) {
	t.Parallel()

	service := &stubKeys{err: services.ErrKeyExists}

	rec := runTestCase(t, NewSystem(&stubPackages{}, service, &stubIssue{}, &stubHostname{}).RootKey, pathSSHKeyRoot, testCaseRequest{
		name: "llave existente", method: http.MethodPost,
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf(msgCode, rec.Code, http.StatusConflict)
	}
}

// TestSSHKeyRootDownload cubre la descarga de la llave privada.
func TestSSHKeyRootDownload(t *testing.T) {
	t.Parallel()

	service := &stubKeys{name: "id_ed25519", content: []byte(sshKeyPrivateTests)}

	rec := runTestCase(t, NewSystem(&stubPackages{}, service, &stubIssue{}, &stubHostname{}).RootKey, pathSSHKeyRoot, testCaseRequest{
		name: "descargar", method: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="id_ed25519"` {
		t.Errorf("Content-Disposition = %q", got)
	}

	if rec.Body.String() != sshKeyPrivateTests {
		t.Errorf("cuerpo = %q, want la llave privada", rec.Body.String())
	}
}

// TestSSHKeyRootWithoutSSHKey cubre la descarga cuando no hay llave.
func TestSSHKeyRootWithoutSSHKey(t *testing.T) {
	t.Parallel()

	service := &stubKeys{downloadsErr: services.ErrKeyNotFound}

	rec := runTestCase(t, NewSystem(&stubPackages{}, service, &stubIssue{}, &stubHostname{}).RootKey, pathSSHKeyRoot, testCaseRequest{
		name: "sin llave", method: http.MethodGet,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCode, rec.Code, http.StatusNotFound)
	}
}

// TestSSHKeyRootMethod cubre el método no permitido.
func TestSSHKeyRootMethod(t *testing.T) {
	t.Parallel()

	rec := runTestCase(t, newSystemTests().RootKey, pathSSHKeyRoot, testCaseRequest{
		name: testCaseMethodNotPermitido, method: http.MethodPut,
	})

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(msgCode, rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestIssueGet cubre la lectura de /etc/issue.
func TestIssueGet(t *testing.T) {
	t.Parallel()

	service := &stubIssue{config: services.IssueConfig{
		Message:  textBienvenido,
		NameHost: true,
		IP:       true,
		Content:  "Bienvenido\nHostname: \\n\nIP: \\4\n",
	}}

	rec := runTestCase(t, NewSystem(&stubPackages{}, &stubKeys{}, service, &stubHostname{}).Issue, pathIssue, testCaseRequest{
		name: testCaseGet, method: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if body["mensaje"] != textBienvenido {
		t.Errorf("mensaje = %v, want Bienvenido", body["mensaje"])
	}

	if body["nombre_host"] != true || body["ip"] != true {
		t.Errorf("opciones = %v, want nombre_host e ip activos", body)
	}
}

// TestIssueWrite cubre el reemplazo de /etc/issue.
func TestIssueWrite(t *testing.T) {
	t.Parallel()

	service := &stubIssue{result: services.Outcome{Message: "Entrada del sistema actualizada."}}

	rec := runTestCase(t, NewSystem(&stubPackages{}, &stubKeys{}, service, &stubHostname{}).Issue, pathIssue, testCaseRequest{
		name:   testCaseValid,
		method: http.MethodPut,
		body:   `{"mensaje":"Bienvenido","nombre_host":true,"fecha":true,"hora":true,"ip":true}`,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	if service.escrituras != 1 {
		t.Errorf("Escribir() se llamó %d veces, want 1", service.escrituras)
	}

	if service.escrito.Message != textBienvenido || !service.escrito.NameHost {
		t.Errorf("configuración = %+v, want mensaje y nombre_host", service.escrito)
	}
}

// TestIssueValidacion cubre las reglas del mensaje y de las opciones.
func TestIssueValidacion(t *testing.T) {
	t.Parallel()

	handler := newSystemTests().Issue
	testCases := []testCaseRequest{
		{
			name:   "mensaje largo",
			method: http.MethodPut,
			body:   `{"mensaje":"` + strings.Repeat("a", 600) + `","nombre_host":false,"fecha":false,"hora":false,"ip":false}`,
			code:   http.StatusBadRequest,
			field:  messageField,
		},
		{
			name:   "sin mensaje ni opciones",
			method: http.MethodPut,
			body:   `{"mensaje":"","nombre_host":false,"fecha":false,"hora":false,"ip":false}`,
			code:   http.StatusBadRequest,
			field:  messageField,
		},
		{name: testCaseBodyInvalid, method: http.MethodPut, body: `{`, code: http.StatusBadRequest},
		{name: testCaseMethodNotPermitido, method: http.MethodPost, code: http.StatusMethodNotAllowed},
	}

	for _, testCase := range testCases {
		rec := runTestCase(t, handler, pathIssue, testCase)
		checkTestCase(t, rec, testCase)
	}
}

// TestIssueErrorOfCommand cubre el fallo al escribir /etc/issue.
func TestIssueErrorOfCommand(t *testing.T) {
	t.Parallel()

	service := &stubIssue{
		result: services.Outcome{Code: 1, Stderr: "Read-only file system"},
		err:    &domain.ExitError{Code: 1, Command: "sh"},
	}

	rec := runTestCase(t, NewSystem(&stubPackages{}, &stubKeys{}, service, &stubHostname{}).Issue, pathIssue, testCaseRequest{
		name: "fallo de escritura", method: http.MethodPut, body: `{"mensaje":"hola"}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestHostnameGet cubre la lectura del nombre de host.
func TestHostnameGet(t *testing.T) {
	t.Parallel()

	service := &stubHostname{config: services.HostnameConfig{
		Name:        hostTest,
		IP:          "192.168.1.50",
		UpdateHosts: true,
	}}

	rec := runTestCase(t, NewSystem(&stubPackages{}, &stubKeys{}, &stubIssue{}, service).Hostname, pathHostname, testCaseRequest{
		name: testCaseGet, method: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if body["nombre"] != hostTest {
		t.Errorf("nombre = %v, want vm1", body["nombre"])
	}

	if body["ip"] != "192.168.1.50" {
		t.Errorf("ip = %v, want 192.168.1.50", body["ip"])
	}

	if body["actualizar_hosts"] != true {
		t.Errorf("actualizar_hosts = %v, want true", body["actualizar_hosts"])
	}
}

// TestHostnameWrite cubre el reemplazo del nombre de host.
func TestHostnameWrite(t *testing.T) {
	t.Parallel()

	service := &stubHostname{result: services.Outcome{Message: "Nombre de host actualizado."}}

	rec := runTestCase(t, NewSystem(&stubPackages{}, &stubKeys{}, &stubIssue{}, service).Hostname, pathHostname, testCaseRequest{
		name:   testCaseValid,
		method: http.MethodPut,
		body:   `{"nombre":"nodo-01","ip":"192.168.1.60","actualizar_hosts":true}`,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	if service.escrituras != 1 {
		t.Errorf("Escribir() se llamó %d veces, want 1", service.escrituras)
	}

	if service.escrito.Name != "nodo-01" || !service.escrito.UpdateHosts {
		t.Errorf("configuración = %+v, want nodo-01 con hosts", service.escrito)
	}
}

// TestHostnameValidacion cubre el nombre RFC 1123 y la dirección IP.
func TestHostnameValidacion(t *testing.T) {
	t.Parallel()

	handler := newSystemTests().Hostname
	testCases := []testCaseRequest{
		{name: "nombre con espacio", method: http.MethodPut, body: `{"nombre":"mi equipo","ip":"192.168.1.60"}`, code: http.StatusBadRequest, field: nameField},
		{name: "IP inválida", method: http.MethodPut, body: `{"nombre":"nodo-web-01","ip":"999.1.1.1"}`, code: http.StatusBadRequest, field: ipField},
		{name: testCaseBodyInvalid, method: http.MethodPut, body: `{`, code: http.StatusBadRequest},
		{name: testCaseMethodNotPermitido, method: http.MethodPost, code: http.StatusMethodNotAllowed},
	}

	for _, testCase := range testCases {
		rec := runTestCase(t, handler, pathHostname, testCase)
		checkTestCase(t, rec, testCase)
	}
}

// TestHostnameErrorOfCommand cubre el fallo al cambiar el nombre.
func TestHostnameErrorOfCommand(t *testing.T) {
	t.Parallel()

	service := &stubHostname{
		result: services.Outcome{Code: 127, Stderr: "hostnamectl: not found"},
		err:    &domain.ExitError{Code: 127, Command: "hostnamectl"},
	}

	rec := runTestCase(t, NewSystem(&stubPackages{}, &stubKeys{}, &stubIssue{}, service).Hostname, pathHostname, testCaseRequest{
		name: "fallo de hostnamectl", method: http.MethodPut, body: `{"nombre":"nodo-01","ip":"192.168.1.60"}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestPkgErrorOfCommand cubre el fallo del servicio al eliminar.
func TestPkgErrorOfCommand(t *testing.T) {
	t.Parallel()

	service := &stubPackages{err: &domain.ExitError{Code: 100, Command: "apt-get remove"}}
	handler := NewSystem(service, &stubKeys{}, &stubIssue{}, &stubHostname{}).Package

	rec := runTestCaseWithPath(t, handler, "/api/packages/nano", nameParam, nameNano, testCaseRequest{
		name: "fallo de APT", method: http.MethodDelete,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestSSHKeyRootErrorGeneric cubre un fallo inesperado al generar la
// llave.
func TestSSHKeyRootErrorGeneric(t *testing.T) {
	t.Parallel()

	service := &stubKeys{err: errors.New("fallo inesperado")}
	handler := NewSystem(&stubPackages{}, service, &stubIssue{}, &stubHostname{}).RootKey

	rec := runTestCase(t, handler, pathSSHKeyRoot, testCaseRequest{
		name: "error genérico", method: http.MethodPost,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestSSHKeyRootErrorOfDownloads cubre un fallo inesperado al leer la
// llave privada.
func TestSSHKeyRootErrorOfDownloads(t *testing.T) {
	t.Parallel()

	service := &stubKeys{downloadsErr: errors.New("fallo de lectura")}
	handler := NewSystem(&stubPackages{}, service, &stubIssue{}, &stubHostname{}).RootKey

	rec := runTestCase(t, handler, pathSSHKeyRoot, testCaseRequest{
		name: "error de descarga", method: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestIssueErrorOfRead cubre el fallo al consultar /etc/issue.
func TestIssueErrorOfRead(t *testing.T) {
	t.Parallel()

	service := &stubIssue{err: errRead}

	rec := runTestCase(t, NewSystem(&stubPackages{}, &stubKeys{}, service, &stubHostname{}).Issue, pathIssue, testCaseRequest{
		name: testCaseErrorRead, method: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestHostnameErrorOfRead cubre el fallo al consultar el nombre.
func TestHostnameErrorOfRead(t *testing.T) {
	t.Parallel()

	service := &stubHostname{err: errRead}

	rec := runTestCase(t, NewSystem(&stubPackages{}, &stubKeys{}, &stubIssue{}, service).Hostname, pathHostname, testCaseRequest{
		name: testCaseErrorRead, method: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}
