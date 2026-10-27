// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"consola-vm/internal/services"
)

// errRead simula un fallo de lectura de la llave.
var errRead = errors.New("fallo de lectura")

// stubUsers es un servicio de usuarios falso con resultados
// preprogramados.
type stubUsers struct {
	lists        []services.User
	user         services.User
	result       services.Outcome
	err          error
	name         string
	sudo         bool
	keys         bool
	files        bool
	file         string
	content      []byte
	downloadsErr error
	created      int
	deleted      int
	generated    int
	password     string
	regenerate   bool
}

// List devuelve la lista preprogramada.
func (s *stubUsers) List(context.Context) ([]services.User, error) {
	return s.lists, s.err
}

// Get devuelve el usuario preprogramado.
func (s *stubUsers) Get(_ context.Context, name string) (services.User, error) {
	s.name = name

	return s.user, s.err
}

// Create registra los argumentos y devuelve el resultado.
func (s *stubUsers) Create(_ context.Context, name string, sudo, keys bool, password string) (services.Outcome, error) {
	s.name = name
	s.sudo = sudo
	s.keys = keys
	s.password = password
	s.created++

	return s.result, s.err
}

// Delete registra los argumentos y devuelve el resultado.
func (s *stubUsers) Delete(_ context.Context, name string, files bool) (services.Outcome, error) {
	s.name = name
	s.files = files
	s.deleted++

	return s.result, s.err
}

// GenerateKey registra la generación y devuelve el resultado.
func (s *stubUsers) GenerateKey(_ context.Context, name string, regenerate bool) (services.Outcome, error) {
	s.name = name
	s.regenerate = regenerate
	s.generated++

	return s.result, s.err
}

// DownloadKey devuelve la llave preprogramada.
func (s *stubUsers) DownloadKey(name string) (file string, content []byte, err error) {
	s.name = name

	return s.file, s.content, s.downloadsErr
}

// TestUsersList cubre el listado de cuentas.
func TestUsersList(t *testing.T) {
	t.Parallel()

	service := &stubUsers{lists: []services.User{
		{Name: userJorge, Sudo: true, UID: 1000, Home: "/home/jorge"},
	}}

	rec := runTestCase(t, NewUsers(service).Collection, pathUsers, testCaseRequest{
		name: testCaseList, method: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	var body userList
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if len(body.Users) != 1 || body.Users[0].Name != userJorge {
		t.Errorf("usuarios = %+v, want jorge", body.Users)
	}
}

// TestUsersCreate cubre la creación de una cuenta.
func TestUsersCreate(t *testing.T) {
	t.Parallel()

	service := &stubUsers{result: services.Outcome{Message: "Usuario «jorge» creado."}}

	rec := runTestCase(t, NewUsers(service).Collection, pathUsers, testCaseRequest{
		name: "crear", method: http.MethodPost, body: `{"nombre":"jorge","sudo":true,"llaves":true,"password":"clave-123"}`,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf(msgCode, rec.Code, http.StatusCreated)
	}

	if got := rec.Header().Get(headerLocation); got != pathUserJorge {
		t.Errorf("Location = %q, want %q", got, pathUserJorge)
	}

	if service.created != 1 || !service.sudo || !service.keys || service.password != "clave-123" {
		t.Errorf("Crear() = %+v, want jorge con sudo, llaves y contraseña", service)
	}
}

// TestUsersCreateExisting cubre el conflicto de una cuenta que ya
// existe.
func TestUsersCreateExisting(t *testing.T) {
	t.Parallel()

	service := &stubUsers{err: services.ErrUserExists}

	rec := runTestCase(t, NewUsers(service).Collection, pathUsers, testCaseRequest{
		name: "existente", method: http.MethodPost, body: `{"nombre":"jorge"}`,
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf(msgCode, rec.Code, http.StatusConflict)
	}
}

// TestUsersValidacion cubre las reglas de nombre y método.
func TestUsersValidacion(t *testing.T) {
	t.Parallel()

	handler := NewUsers(&stubUsers{}).Collection
	testCases := []testCaseRequest{
		{name: "con mayúsculas", method: http.MethodPost, body: `{"nombre":"José","sudo":false}`, code: http.StatusBadRequest, field: nameField},
		{name: "root", method: http.MethodPost, body: `{"nombre":"root","sudo":true}`, code: http.StatusBadRequest, field: nameField},
		{name: "contraseña con salto", method: http.MethodPost, body: `{"nombre":"jorge","password":"a\nb"}`, code: http.StatusBadRequest, field: passwordField},
		{name: testCaseBodyInvalid, method: http.MethodPost, body: `{`, code: http.StatusBadRequest},
		{name: testCaseMethodNotPermitido, method: http.MethodPut, code: http.StatusMethodNotAllowed},
	}

	for _, testCase := range testCases {
		rec := runTestCase(t, handler, pathUsers, testCase)
		checkTestCase(t, rec, testCase)
	}
}

// TestUserGet cubre la consulta de una cuenta.
func TestUserGet(t *testing.T) {
	t.Parallel()

	service := &stubUsers{user: services.User{
		Name: "jorge", Sudo: true, UID: 1000, Home: "/home/jorge",
	}}

	rec := runTestCaseWithPath(t, NewUsers(service).User, pathUserJorge, nameParam, userJorge, testCaseRequest{
		name: testCaseGet, method: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	var body userResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf(msgJSONInvalid, err)
	}

	if body.Name != userJorge || !body.Sudo {
		t.Errorf("usuario = %+v, want jorge con sudo", body)
	}
}

// TestUserNotFound cubre la consulta de una cuenta inexistente.
func TestUserNotFound(t *testing.T) {
	t.Parallel()

	service := &stubUsers{err: services.ErrUserNotFound}

	rec := runTestCaseWithPath(t, NewUsers(service).User, pathUserJorge, nameParam, userJorge, testCaseRequest{
		name: testCaseInexistente, method: http.MethodGet,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCode, rec.Code, http.StatusNotFound)
	}
}

// TestUserDelete cubre la eliminación con archivos.
func TestUserDelete(t *testing.T) {
	t.Parallel()

	service := &stubUsers{}

	rec := runTestCaseWithPath(t, NewUsers(service).User, pathUserJorge+"?files=true", nameParam, userJorge, testCaseRequest{
		name: testCaseDelete, method: http.MethodDelete,
	})

	if rec.Code != http.StatusNoContent {
		t.Fatalf(msgCode, rec.Code, http.StatusNoContent)
	}

	if service.deleted != 1 || !service.files {
		t.Errorf("Eliminar() = %+v, want con archivos", service)
	}
}

// TestUserFilesInvalid cubre el parámetro files no booleano.
func TestUserFilesInvalid(t *testing.T) {
	t.Parallel()

	rec := runTestCaseWithPath(t, NewUsers(&stubUsers{}).User, pathUserJorge+"?files=quizá", nameParam, userJorge, testCaseRequest{
		name: "archivos inválido", method: http.MethodDelete,
	})

	checkTestCase(t, rec, testCaseRequest{name: "archivos inválido", code: http.StatusBadRequest, field: filesParam})
}

// TestUserSSHKey cubre la descarga de la llave privada.
func TestUserSSHKey(t *testing.T) {
	t.Parallel()

	service := &stubUsers{file: "jorge_ed25519", content: []byte(sshKeyPrivateTests)}

	rec := runTestCaseWithPath(t, NewUsers(service).Key, pathUserJorge+"/key", nameParam, userJorge, testCaseRequest{
		name: "descargar", method: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCode, rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="jorge_ed25519"` {
		t.Errorf("Content-Disposition = %q", got)
	}

	if rec.Body.String() != sshKeyPrivateTests {
		t.Errorf("cuerpo = %q, want la llave privada", rec.Body.String())
	}
}

// TestUserSSHKeyInexistente cubre la llave que no existe.
func TestUserSSHKeyInexistente(t *testing.T) {
	t.Parallel()

	service := &stubUsers{downloadsErr: services.ErrUserKeyNotFound}

	rec := runTestCaseWithPath(t, NewUsers(service).Key, pathUserJorge+"/key", nameParam, userJorge, testCaseRequest{
		name: "sin llave", method: http.MethodGet,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCode, rec.Code, http.StatusNotFound)
	}
}

// TestUserSSHKeyError cubre un fallo inesperado al leer la llave.
func TestUserSSHKeyError(t *testing.T) {
	t.Parallel()

	service := &stubUsers{downloadsErr: errRead}

	rec := runTestCaseWithPath(t, NewUsers(service).Key, pathUserJorge+"/key", nameParam, userJorge, testCaseRequest{
		name: testCaseErrorRead, method: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestUsersListError cubre el fallo al listar cuentas.
func TestUsersListError(t *testing.T) {
	t.Parallel()

	service := &stubUsers{err: errRead}

	rec := runTestCase(t, NewUsers(service).Collection, pathUsers, testCaseRequest{
		name: testCaseList, method: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestUsersCreateError cubre el fallo de useradd.
func TestUsersCreateError(t *testing.T) {
	t.Parallel()

	service := &stubUsers{err: errRead}

	rec := runTestCase(t, NewUsers(service).Collection, pathUsers, testCaseRequest{
		name: "fallo de creación", method: http.MethodPost, body: `{"nombre":"jorge"}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestUserErrorGeneric cubre un fallo inesperado al consultar.
func TestUserErrorGeneric(t *testing.T) {
	t.Parallel()

	service := &stubUsers{err: errRead}

	rec := runTestCaseWithPath(t, NewUsers(service).User, pathUserJorge, nameParam, userJorge, testCaseRequest{
		name: "error de consulta", method: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestUserDeleteNotFound cubre la eliminación de una cuenta
// inexistente.
func TestUserDeleteNotFound(t *testing.T) {
	t.Parallel()

	service := &stubUsers{err: services.ErrUserNotFound}

	rec := runTestCaseWithPath(t, NewUsers(service).User, pathUserJorge, nameParam, userJorge, testCaseRequest{
		name: testCaseInexistente, method: http.MethodDelete,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCode, rec.Code, http.StatusNotFound)
	}
}

// TestUserDeleteError cubre el fallo de userdel.
func TestUserDeleteError(t *testing.T) {
	t.Parallel()

	service := &stubUsers{err: errRead}

	rec := runTestCaseWithPath(t, NewUsers(service).User, pathUserJorge, nameParam, userJorge, testCaseRequest{
		name: "fallo de eliminación", method: http.MethodDelete,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCode, rec.Code, http.StatusInternalServerError)
	}
}

// TestUserSSHKeyMethod cubre un método no permitido en la llave.
func TestUserSSHKeyMethod(t *testing.T) {
	t.Parallel()

	rec := runTestCaseWithPath(t, NewUsers(&stubUsers{}).Key, pathUserJorge+"/key", nameParam, userJorge, testCaseRequest{
		name: testCaseMethodNotPermitido, method: http.MethodPut,
	})

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(msgCode, rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestUsersGenerateKey comprueba la generación de la llave de un usuario.
func TestUsersGenerateKey(t *testing.T) {
	t.Parallel()

	service := &stubUsers{result: services.Outcome{Message: "Llave generada."}}
	rec := runTestCaseWithPath(t, NewUsers(service).Key, pathUserJorge+"/key", nameParam, userJorge, testCaseRequest{
		name:   "generar llave",
		method: http.MethodPost,
		code:   http.StatusCreated,
	})

	if service.generated != 1 || service.name != userJorge {
		t.Errorf("GenerateKey() = %d, %q, want 1, %q", service.generated, service.name, userJorge)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("código = %d, want %d", rec.Code, http.StatusCreated)
	}
}

// TestUsersGenerateKeyExisting comprueba el conflicto de una llave existente.
func TestUsersGenerateKeyExisting(t *testing.T) {
	t.Parallel()

	service := &stubUsers{err: services.ErrUserKeyExists}
	runTestCaseWithPath(t, NewUsers(service).Key, pathUserJorge+"/key", nameParam, userJorge, testCaseRequest{
		name:   "llave existente",
		method: http.MethodPost,
		code:   http.StatusConflict,
	})
}

// TestUsersGenerateKeyNotFound comprueba el usuario inexistente.
func TestUsersGenerateKeyNotFound(t *testing.T) {
	t.Parallel()

	service := &stubUsers{err: services.ErrUserNotFound}
	runTestCaseWithPath(t, NewUsers(service).Key, pathUserJorge+"/key", nameParam, userJorge, testCaseRequest{
		name:   "usuario inexistente",
		method: http.MethodPost,
		code:   http.StatusNotFound,
	})
}

// TestUsersGenerateKeyRoot rechaza la generación por la ruta de usuarios.
func TestUsersGenerateKeyRoot(t *testing.T) {
	t.Parallel()

	runTestCaseWithPath(t, NewUsers(&stubUsers{}).Key, pathUsers+"/root/key", nameParam, "root", testCaseRequest{
		name:   "root no se genera aquí",
		method: http.MethodPost,
		code:   http.StatusBadRequest,
	})
}

// TestUsersGenerateKeyRegenerate comprueba la opción de regenerar.
func TestUsersGenerateKeyRegenerate(t *testing.T) {
	t.Parallel()

	service := &stubUsers{result: services.Outcome{Message: "Llave regenerada."}}
	runTestCaseWithPath(t, NewUsers(service).Key, pathUserJorge+"/key?regenerar=true", nameParam, userJorge, testCaseRequest{
		name:   "regenerar llave",
		method: http.MethodPost,
		code:   http.StatusCreated,
	})

	if !service.regenerate {
		t.Error("regenerate = false, want true")
	}
}
