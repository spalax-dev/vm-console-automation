// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"consola-vm/internal/services"
)

const (
	pathHistoryTests = "/api/historial"
	pathUnitsTests   = "/api/units"
)

// failingWriter simula un cliente que corta la respuesta a mitad.
type failingWriter struct {
	header http.Header
	state  int
}

// Header devuelve las cabeceras de la respuesta.
func (e *failingWriter) Header() http.Header {
	if e.header == nil {
		e.header = make(http.Header)
	}

	return e.header
}

// WriteHeader registra el primer estado y descarta la escritura.
func (e *failingWriter) WriteHeader(code int) {
	if e.state == 0 {
		e.state = code
	}
}

// Write siempre falla, como una conexión cerrada.
func (e *failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("conexión cerrada")
}

// TestAccessRegisterErrorInternal devuelve 500 ante un fallo inesperado.
func TestAccessRegisterErrorInternal(t *testing.T) {
	t.Parallel()

	service := &stubAccess{err: errors.New("base caída")}

	rec := runTestCase(t, NewAccess(service).ServeHTTP, pathRegistration, testCaseRequest{
		name: "error interno", method: http.MethodPost, body: `{"usuario":"root","password":"secreto"}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestAccesoOlvidar cubre el borrado del registro y sus errores.
func TestAccessForgetResponses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		err      error
		expected int
	}{
		{"correcto", nil, http.StatusNoContent},
		{"sin máquina", services.ErrNoMachine, http.StatusConflict},
		{"error interno", errors.New("base caída"), http.StatusInternalServerError},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := &stubAccess{err: testCase.err}

			rec := runTestCase(t, NewAccess(service).ServeHTTP, pathRegistration, testCaseRequest{
				name: testCase.name, method: http.MethodDelete,
			})

			if rec.Code != testCase.expected {
				t.Errorf(msgCode, rec.Code, testCase.expected)
			}
		})
	}
}

// TestAuthErrorsInternos cubre los fallos del servicio de autenticación.
func TestAuthErrorsInternos(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	NewAuth(&stubConsole{configureErr: errors.New("base caída")}).State(rec,
		requestWithCookie(http.MethodGet, pathAuthState, "", ""))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Estado() = %d, want 500", rec.Code)
	}

	rec = httptest.NewRecorder()
	NewAuth(&stubConsole{configureErr: errors.New("base caída")}).Configure(rec,
		requestWithCookie(http.MethodPost, "/api/auth/setup", bodyCredentials, ""))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Configurar() = %d, want 500", rec.Code)
	}

	rec = httptest.NewRecorder()
	NewAuth(&stubConsole{configureErr: services.ErrConsoleData}).Configure(rec,
		requestWithCookie(http.MethodPost, "/api/auth/setup", bodyCredentials, ""))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Configurar() con datos inválidos = %d, want 400", rec.Code)
	}

	rec = httptest.NewRecorder()
	NewAuth(&stubConsole{authenticateErr: errors.New("base caída")}).Login(rec,
		requestWithCookie(http.MethodPost, pathAuthLogin, bodyCredentials, ""))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Entrar() = %d, want 500", rec.Code)
	}

	rec = httptest.NewRecorder()
	NewAuth(&stubConsole{tokenErr: errors.New("sin secreto"), autenticado: true}).Login(rec,
		requestWithCookie(http.MethodPost, pathAuthLogin, bodyCredentials, ""))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Entrar() sin sesión = %d, want 500", rec.Code)
	}

	rec = httptest.NewRecorder()
	NewAuth(&stubConsole{}).Logout(rec,
		requestWithCookie(http.MethodGet, "/api/auth/logout", "", ""))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Salir() con GET = %d, want 405", rec.Code)
	}
}

// TestDiagnosticsErrorInternal devuelve 500 ante un fallo inesperado.
func TestDiagnosticsErrorInternal(t *testing.T) {
	t.Parallel()

	rec := runTestCase(t, NewDiagnostics(&stubDiagnostics{err: errors.New("fallo")}).ServeHTTP,
		pathDiagnostics, testCaseRequest{name: "error", method: http.MethodGet})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestServerInvalidRejectsFormat valida el identificador del
// back-end antes de tocar la máquina.
func TestServerInvalidRejectsFormat(t *testing.T) {
	t.Parallel()

	rec := runTestCaseWithPath(t, NewHAProxy(&stubHAProxy{}).Server, pathServers+"/x",
		serverParam, "192.168.1.61", testCaseRequest{name: "sin puerto", method: http.MethodDelete})

	if rec.Code != http.StatusBadRequest {
		t.Errorf(msgCode, rec.Code, http.StatusBadRequest)
	}
}

// TestPowerOnWithoutMachineRejectsPath exige el identificador.
func TestPowerOnWithoutMachineRejectsPath(t *testing.T) {
	t.Parallel()

	rec := runTestCaseWithPath(t, NewMachines(&stubMachines{}).PowerOn, "/api/vms//poweron",
		"vm", "", testCaseRequest{name: "sin máquina", method: http.MethodPost})

	if rec.Code != http.StatusBadRequest {
		t.Errorf(msgCode, rec.Code, http.StatusBadRequest)
	}
}

// TestPowerOnErrorInternal devuelve 500 ante un fallo inesperado.
func TestPowerOnErrorInternal(t *testing.T) {
	t.Parallel()

	service := &stubMachines{errPowerOn: errors.New("hipervisor ocupado")}

	rec := runTestCaseWithPath(t, NewMachines(service).PowerOn, "/api/vms/vm1/poweron",
		"vm", testMachine, testCaseRequest{name: "error", method: http.MethodPost})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestBindErrorInternal devuelve 500 ante un fallo inesperado.
func TestBindErrorInternal(t *testing.T) {
	t.Parallel()

	binder := &stubBinding{err: errors.New("hipervisor ocupado")}
	handler := NewSession(&stubDiscoverer{}, WithBinder(binder))

	rec := runTestCase(t, handler.ServeHTTP, pathSession, testCaseRequest{
		name: "vincular", method: http.MethodPut, body: `{"vm":"vm1"}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestSessionWithoutMachineVinculada devuelve 503 con un mensaje claro.
func TestSessionWithoutMachineVinculada(t *testing.T) {
	t.Parallel()

	handler := NewSession(&stubDiscoverer{err: services.ErrNoMachine})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, pathSession, http.NoBody))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf(msgCode, rec.Code, http.StatusServiceUnavailable)
	}

	if !strings.Contains(rec.Body.String(), "vinculada") {
		t.Errorf("cuerpo = %q, want el mensaje de máquina sin vincular", rec.Body.String())
	}
}

// TestHistoryErrorInternal devuelve 500 cuando la consulta falla.
func TestHistoryErrorInternal(t *testing.T) {
	t.Parallel()

	rec := runTestCase(t, NewHistory(&stubHistory{err: errors.New("base caída")}).ServeHTTP,
		pathHistoryTests, testCaseRequest{name: "error", method: http.MethodGet})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestHistoryWithRegistrationFailedNotInterrumpe la operación.
func TestHistoryWithRegistrationFailedNotInterrumpe(t *testing.T) {
	t.Parallel()

	service := &stubHistory{err: errors.New("base caída")}
	handler := WithHistory(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), service, func() string { return testMachine })

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		pathPackages, strings.NewReader(`{"nombre":"nano"}`)))

	if rec.Code != http.StatusNoContent {
		t.Errorf(msgCode, rec.Code, http.StatusNoContent)
	}
}

// TestRegistrationWriterWritesWithoutState fija 200 y propaga el fallo.
func TestRegistrationWriterWritesWithoutState(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	registration := &registrationWriter{ResponseWriter: rec}

	if _, err := registration.Write([]byte("x")); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	if registration.state != http.StatusOK {
		t.Errorf("estado = %d, want 200", registration.state)
	}

	withFailure := &registrationWriter{ResponseWriter: &failingWriter{}}
	if _, err := withFailure.Write([]byte("x")); err == nil {
		t.Error("Write() error = nil, want el fallo del cliente")
	}
}

// TestReasonOfIgnoresBodiesForeign deja vacío el motivo cuando no aplica.
func TestReasonOfIgnoresBodiesForeign(t *testing.T) {
	t.Parallel()

	if reason := reasonOf(http.StatusOK, []byte(`{"mensaje":"x"}`)); reason != "" {
		t.Errorf("motivoDe() = %q, want vacío para un 200", reason)
	}

	if reason := reasonOf(http.StatusBadRequest, nil); reason != "" {
		t.Errorf("motivoDe() = %q, want vacío sin cuerpo", reason)
	}

	if reason := reasonOf(http.StatusBadRequest, []byte("no es json")); reason != "" {
		t.Errorf("motivoDe() = %q, want vacío con JSON inválido", reason)
	}
}

// TestTruncateLimitaText sin partir caracteres.
func TestTruncateLimitaText(t *testing.T) {
	t.Parallel()

	if text := truncate("corto", 10); text != "corto" {
		t.Errorf("recortar() = %q, want corto", text)
	}

	if text := truncate(strings.Repeat("á", 12), 5); text != "ááááá…" {
		t.Errorf("recortar() = %q, want cinco caracteres y puntos suspensivos", text)
	}
}

// TestLevelOfStateClasifica clasifica advertencias, errores e info.
func TestLevelOfStateClasifica(t *testing.T) {
	t.Parallel()

	testCases := map[int]slog.Level{
		http.StatusOK:                  slog.LevelInfo,
		http.StatusNotFound:            slog.LevelWarn,
		http.StatusServiceUnavailable:  slog.LevelWarn,
		http.StatusInternalServerError: slog.LevelError,
	}

	for state, expected := range testCases {
		if level := stateLevel(state); level != expected {
			t.Errorf("nivelDeEstado(%d) = %v, want %v", state, level, expected)
		}
	}
}

// TestNewRequestWithFailureOfAleatorio usa la hora como respaldo.
func TestNewRequestWithFailureOfAleatorio(t *testing.T) {
	original := readAleatorio

	t.Cleanup(func() { readAleatorio = original })

	readAleatorio = func([]byte) (int, error) { return 0, errors.New("sin entropía") }

	request := newRequestID()
	if _, err := time.Parse("20060102T150405.000000", request); err != nil {
		t.Errorf("nuevaSolicitud() = %q, want el respaldo con formato de hora: %v", request, err)
	}
}

// TestWithObservabilityWithoutLogger usa el logger por omisión.
func TestWithObservabilityWithoutLogger(t *testing.T) {
	t.Parallel()

	handler := WithObservability(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), nil, nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		pathSession, http.NoBody))

	if rec.Code != http.StatusOK {
		t.Errorf(msgCode, rec.Code, http.StatusOK)
	}
}

// TestWriterStateCapturesError fija 200 y trunca el cuerpo de error.
func TestWriterStateCapturesError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	writer := &writerState{ResponseWriter: rec}

	body := strings.Repeat("x", maxBodyLog+10)

	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	if writer.state != http.StatusOK {
		t.Errorf("estado = %d, want 200", writer.state)
	}

	withFailure := &writerState{ResponseWriter: &failingWriter{}}
	if _, err := withFailure.Write([]byte("x")); err == nil {
		t.Error("Write() error = nil, want el fallo del cliente")
	}

	withError := &writerState{ResponseWriter: httptest.NewRecorder()}
	withError.WriteHeader(http.StatusBadRequest)

	if _, err := withError.Write([]byte(body)); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	if withError.captures.Len() != maxBodyLog {
		t.Errorf("captura = %d bytes, want %d", withError.captures.Len(), maxBodyLog)
	}
}

// TestWriteJSONWithBodyNotSerializable solo deja constancia en el log.
func TestWriteJSONWithBodyNotSerializable(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, make(chan int))

	if rec.Code != http.StatusOK {
		t.Errorf(msgCode, rec.Code, http.StatusOK)
	}
}

// TestDownloadKeysWithClienteCaido comprueba que la descarga conserva
// las cabeceras aunque el cliente corte la conexión al escribir.
func TestDownloadKeysWithClienteCaido(t *testing.T) {
	t.Parallel()

	writer := &failingWriter{}

	system := NewSystem(&stubPackages{}, &stubKeys{name: "id_ed25519", content: []byte("k")},
		&stubIssue{}, &stubHostname{})
	system.downloadSSHKeyRoot(writer)

	if writer.state != http.StatusOK {
		t.Errorf("descargarLlaveRoot() estado = %d, want 200", writer.state)
	}

	if kind := writer.Header().Get(headerContentType); kind != "application/octet-stream" {
		t.Errorf("%s = %q, want application/octet-stream", headerContentType, kind)
	}

	if disposicion := writer.Header().Get("Content-Disposition"); !strings.Contains(disposicion, "id_ed25519") {
		t.Errorf("Content-Disposition = %q, want el nombre de la llave", disposicion)
	}

	users := NewUsers(&stubUsers{file: "jorge_ed25519", content: []byte("k")})
	writer = &failingWriter{}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, pathUserJorge+"/key", http.NoBody)
	req.SetPathValue(nameParam, userJorge)
	users.Key(writer, req)

	if writer.state != http.StatusOK {
		t.Errorf("Llave() estado = %d, want 200", writer.state)
	}

	if disposicion := writer.Header().Get("Content-Disposition"); !strings.Contains(disposicion, "jorge_ed25519") {
		t.Errorf("Content-Disposition = %q, want el nombre de la llave", disposicion)
	}
}

// TestUnitsErrorInternal devuelve 500 cuando el listado falla.
func TestUnitsErrorInternal(t *testing.T) {
	t.Parallel()

	rec := runTestCase(t, NewSystemd(&stubSystemd{err: errors.New("systemd caído")}).Units,
		pathUnitsTests, testCaseRequest{name: "error", method: http.MethodGet})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestChangeStateWithoutGet devuelve 500 si la relectura falla.
func TestChangeStateWithoutGet(t *testing.T) {
	t.Parallel()

	service := &stubSystemd{getErr: errors.New("systemd caído")}

	rec := runTestCaseWithPath(t, NewSystemd(service).Unit, pathUnitsTests+"/"+unitCatalogo,
		unitParam, unitCatalogo, testCaseRequest{
			name: "cambiar estado", method: http.MethodPatch, body: `{"activo":true}`,
		})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestUserErrorInternal devuelve 500 cuando la consulta falla.
func TestUserErrorInternal(t *testing.T) {
	t.Parallel()

	service := &stubUsers{err: errors.New("máquina caída")}

	rec := runTestCaseWithPath(t, NewUsers(service).User, pathUserJorge,
		nameParam, userJorge, testCaseRequest{name: "consultar", method: http.MethodGet})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestAyudantesOfHistoryCubrenValuesByOmision comprueba las
// funciones que traducen el estado y el resultado.
func TestAyudantesOfHistoryCubrenValuesByOmision(t *testing.T) {
	t.Parallel()

	if state := stateOf(&registrationWriter{}); state != http.StatusOK {
		t.Errorf("estadoDe() = %d, want 200", state)
	}

	if result := resultOf(http.StatusNotFound); result != "error" {
		t.Errorf("resultadoDe() = %q, want error", result)
	}

	if name := activeName(nil); name != "" {
		t.Errorf("nombreActivo(nil) = %q, want vacío", name)
	}

	if data := readBody(&http.Request{}); len(data) != 0 {
		t.Errorf("leerCuerpo() = %v, want vacío sin cuerpo", data)
	}
}

// TestWithObservabilityWithRequestLarga genera un identificador nuevo.
func TestWithObservabilityWithRequestLarga(t *testing.T) {
	t.Parallel()

	handler := WithObservability(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), func() string { return testMachine }, slog.New(slog.NewTextHandler(&strings.Builder{}, nil)))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, pathSession, http.NoBody)
	req.Header.Set(headerRequest, strings.Repeat("x", 80))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf(msgCode, rec.Code, http.StatusNoContent)
	}

	if len(rec.Header().Get(headerRequest)) != 16 {
		t.Errorf("solicitud = %q, want un identificador nuevo", rec.Header().Get(headerRequest))
	}
}

// TestServerDeletedWithoutBody responde 204 y registra la baja.
func TestServerDeletedWithoutBody(t *testing.T) {
	t.Parallel()

	service := &stubHAProxy{}

	rec := runTestCaseWithPath(t, NewHAProxy(service).Server, pathServers+"/"+serverTests,
		serverParam, serverTests, testCaseRequest{
			name: "eliminar", method: http.MethodDelete,
		})

	if rec.Code != http.StatusNoContent {
		t.Errorf(msgCode, rec.Code, http.StatusNoContent)
	}

	if service.deleted != 1 {
		t.Errorf("eliminados = %d, want 1", service.deleted)
	}
}

// TestContextoWithUser conserva el usuario autenticado.
func TestContextoWithUser(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(t.Context(), userContextKey{}, userJorge)

	if user := UserFrom(ctx); user != userJorge {
		t.Errorf("UsuarioDe() = %q, want %q", user, userJorge)
	}
}

// TestNewRequestReturnsHex comprueba el identificador normal.
func TestNewRequestReturnsHex(t *testing.T) {
	t.Parallel()

	request := newRequestID()
	if len(request) != 16 {
		t.Fatalf("nuevaSolicitud() = %q, want 16 caracteres hex", request)
	}

	if _, err := hex.DecodeString(request); err != nil {
		t.Errorf("nuevaSolicitud() = %q, want hexadecimal válido: %v", request, err)
	}
}

// failingReader devuelve un error al leer el cuerpo.
type failingReader struct{}

// Read siempre falla.
func (failingReader) Read([]byte) (int, error) { return 0, errors.New("cuerpo roto") }

// Close no hace nada.
func (failingReader) Close() error { return nil }

// TestMethodsNotPermitidosInAccess cubre el 405 de los recursos de la
// consola y de los módulos sin método alternativo.
func TestMethodsNotPermitidosInAccess(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		handler http.HandlerFunc
		path    string
	}{
		{"configurar", NewAuth(&stubConsole{}).Configure, "/api/auth/setup"},
		{"entrar", NewAuth(&stubConsole{}).Login, pathAuthLogin},
		{"unidades", NewSystemd(&stubSystemd{}).Units, pathUnitsTests},
		{"usuario", NewUsers(&stubUsers{}).User, pathUserJorge},
		{"llave", NewUsers(&stubUsers{}).Key, pathUserJorge + "/key"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			rec := runTestCase(t, testCase.handler, testCase.path, testCaseRequest{
				name: testCase.name, method: http.MethodPut,
			})

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf(msgCode, rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

// TestLoginWithCredentialsInvalid rechaza el cuerpo antes de consultar.
func TestLoginWithCredentialsInvalid(t *testing.T) {
	t.Parallel()

	rec := runTestCase(t, NewAuth(&stubConsole{}).Login, pathAuthLogin, testCaseRequest{
		name: "usuario inválido", method: http.MethodPost,
		body: `{"usuario":"Jorge","password":"una-clave-larga"}`,
	})

	if rec.Code != http.StatusBadRequest {
		t.Errorf(msgCode, rec.Code, http.StatusBadRequest)
	}
}

// TestChangeStateWithBodyInvalid rechaza un JSON mal formado.
func TestChangeStateWithBodyInvalid(t *testing.T) {
	t.Parallel()

	rec := runTestCaseWithPath(t, NewSystemd(&stubSystemd{}).Unit,
		pathUnitsTests+"/"+unitCatalogo, unitParam, unitCatalogo, testCaseRequest{
			name: "cuerpo inválido", method: http.MethodPatch, body: `{`,
		})

	if rec.Code != http.StatusBadRequest {
		t.Errorf(msgCode, rec.Code, http.StatusBadRequest)
	}
}

// TestUserWithNameInvalid rechaza el identificador de la ruta.
func TestUserWithNameInvalid(t *testing.T) {
	t.Parallel()

	rec := runTestCaseWithPath(t, NewUsers(&stubUsers{}).User,
		pathUserJorge, nameParam, userJorgeMayus, testCaseRequest{
			name: "nombre inválido", method: http.MethodGet,
		})

	if rec.Code != http.StatusBadRequest {
		t.Errorf(msgCode, rec.Code, http.StatusBadRequest)
	}

	rec = runTestCaseWithPath(t, NewUsers(&stubUsers{}).Key,
		pathUserJorge+"/key", nameParam, userJorgeMayus, testCaseRequest{
			name: "llave con nombre inválido", method: http.MethodGet,
		})

	if rec.Code != http.StatusBadRequest {
		t.Errorf(msgCode, rec.Code, http.StatusBadRequest)
	}
}

// TestReadBodyWithFailureOfRead devuelve un mapa vacío.
func TestReadBodyWithFailureOfRead(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, pathPackages, failingReader{})
	req.Header.Set(headerContentType, kindJSONTests)

	if data := readBody(req); len(data) != 0 {
		t.Errorf("leerCuerpo() = %v, want vacío con un cuerpo ilegible", data)
	}
}

// TestObservabilityWithResponseWithoutState supone 200 cuando el handler
// no escribe nada.
func TestObservabilityWithResponseWithoutState(t *testing.T) {
	t.Parallel()

	handler := WithObservability(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		nil, slog.New(slog.NewTextHandler(&strings.Builder{}, nil)))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		pathSession, http.NoBody))

	if rec.Code != http.StatusOK {
		t.Errorf(msgCode, rec.Code, http.StatusOK)
	}
}
