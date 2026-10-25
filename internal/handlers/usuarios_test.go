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

// errLectura simula un fallo de lectura de la llave.
var errLectura = errors.New("fallo de lectura")

// stubUsuarios es un servicio de usuarios falso con resultados
// preprogramados.
type stubUsuarios struct {
	lista       []services.Usuario
	usuario     services.Usuario
	resultado   services.Resultado
	err         error
	nombre      string
	sudo        bool
	llaves      bool
	archivos    bool
	archivo     string
	contenido   []byte
	descargaErr error
	creados     int
	eliminados  int
}

// Listar devuelve la lista preprogramada.
func (s *stubUsuarios) Listar(context.Context) ([]services.Usuario, error) {
	return s.lista, s.err
}

// Obtener devuelve el usuario preprogramado.
func (s *stubUsuarios) Obtener(_ context.Context, nombre string) (services.Usuario, error) {
	s.nombre = nombre

	return s.usuario, s.err
}

// Crear registra los argumentos y devuelve el resultado.
func (s *stubUsuarios) Crear(_ context.Context, nombre string, sudo, llaves bool) (services.Resultado, error) {
	s.nombre = nombre
	s.sudo = sudo
	s.llaves = llaves
	s.creados++

	return s.resultado, s.err
}

// Eliminar registra los argumentos y devuelve el resultado.
func (s *stubUsuarios) Eliminar(_ context.Context, nombre string, archivos bool) (services.Resultado, error) {
	s.nombre = nombre
	s.archivos = archivos
	s.eliminados++

	return s.resultado, s.err
}

// DescargarLlave devuelve la llave preprogramada.
func (s *stubUsuarios) DescargarLlave(nombre string) (archivo string, contenido []byte, err error) {
	s.nombre = nombre

	return s.archivo, s.contenido, s.descargaErr
}

// TestUsuariosListar cubre el listado de cuentas.
func TestUsuariosListar(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{lista: []services.Usuario{
		{Nombre: "jorge", Sudo: true, UID: 1000, Home: "/home/jorge"},
	}}

	rec := ejecutarCaso(t, NuevoUsuarios(servicio).Coleccion, rutaUsuarios, casoPeticion{
		nombre: casoListar, metodo: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	var cuerpo listaUsuarios
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if len(cuerpo.Usuarios) != 1 || cuerpo.Usuarios[0].Nombre != usuarioJorge {
		t.Errorf("usuarios = %+v, want jorge", cuerpo.Usuarios)
	}
}

// TestUsuariosCrear cubre la creación de una cuenta.
func TestUsuariosCrear(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{resultado: services.Resultado{Mensaje: "Usuario «jorge» creado."}}

	rec := ejecutarCaso(t, NuevoUsuarios(servicio).Coleccion, rutaUsuarios, casoPeticion{
		nombre: "crear", metodo: http.MethodPost, cuerpo: `{"nombre":"jorge","sudo":true,"llaves":true}`,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf(msgCodigo, rec.Code, http.StatusCreated)
	}

	if got := rec.Header().Get(cabeceraLocation); got != rutaUsuarioJorge {
		t.Errorf("Location = %q, want %q", got, rutaUsuarioJorge)
	}

	if servicio.creados != 1 || !servicio.sudo || !servicio.llaves {
		t.Errorf("Crear() = %+v, want jorge con sudo y llaves", servicio)
	}
}

// TestUsuariosCrearExistente cubre el conflicto de una cuenta que ya
// existe.
func TestUsuariosCrearExistente(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{err: services.ErrUsuarioExistente}

	rec := ejecutarCaso(t, NuevoUsuarios(servicio).Coleccion, rutaUsuarios, casoPeticion{
		nombre: "existente", metodo: http.MethodPost, cuerpo: `{"nombre":"jorge"}`,
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf(msgCodigo, rec.Code, http.StatusConflict)
	}
}

// TestUsuariosValidacion cubre las reglas de nombre y método.
func TestUsuariosValidacion(t *testing.T) {
	t.Parallel()

	handler := NuevoUsuarios(&stubUsuarios{}).Coleccion
	casos := []casoPeticion{
		{nombre: "con mayúsculas", metodo: http.MethodPost, cuerpo: `{"nombre":"José","sudo":false}`, codigo: http.StatusBadRequest, campo: campoNombre},
		{nombre: "root", metodo: http.MethodPost, cuerpo: `{"nombre":"root","sudo":true}`, codigo: http.StatusBadRequest, campo: campoNombre},
		{nombre: casoCuerpoInvalido, metodo: http.MethodPost, cuerpo: `{`, codigo: http.StatusBadRequest},
		{nombre: casoMetodoNoPermitido, metodo: http.MethodPut, codigo: http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, rutaUsuarios, caso)
		comprobarCaso(t, rec, caso)
	}
}

// TestUsuarioObtener cubre la consulta de una cuenta.
func TestUsuarioObtener(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{usuario: services.Usuario{
		Nombre: "jorge", Sudo: true, UID: 1000, Home: "/home/jorge",
	}}

	rec := ejecutarCasoConPath(t, NuevoUsuarios(servicio).Usuario, rutaUsuarioJorge, campoName, usuarioJorge, casoPeticion{
		nombre: casoConsultar, metodo: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	var cuerpo usuarioRespuesta
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if cuerpo.Nombre != usuarioJorge || !cuerpo.Sudo {
		t.Errorf("usuario = %+v, want jorge con sudo", cuerpo)
	}
}

// TestUsuarioNoEncontrado cubre la consulta de una cuenta inexistente.
func TestUsuarioNoEncontrado(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{err: services.ErrUsuarioNoEncontrado}

	rec := ejecutarCasoConPath(t, NuevoUsuarios(servicio).Usuario, rutaUsuarioJorge, campoName, usuarioJorge, casoPeticion{
		nombre: "inexistente", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNotFound)
	}
}

// TestUsuarioEliminar cubre la eliminación con archivos.
func TestUsuarioEliminar(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{}

	rec := ejecutarCasoConPath(t, NuevoUsuarios(servicio).Usuario, rutaUsuarioJorge+"?files=true", campoName, usuarioJorge, casoPeticion{
		nombre: casoEliminar, metodo: http.MethodDelete,
	})

	if rec.Code != http.StatusNoContent {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNoContent)
	}

	if servicio.eliminados != 1 || !servicio.archivos {
		t.Errorf("Eliminar() = %+v, want con archivos", servicio)
	}
}

// TestUsuarioArchivosInvalido cubre el parámetro files no booleano.
func TestUsuarioArchivosInvalido(t *testing.T) {
	t.Parallel()

	rec := ejecutarCasoConPath(t, NuevoUsuarios(&stubUsuarios{}).Usuario, rutaUsuarioJorge+"?files=quizá", campoName, usuarioJorge, casoPeticion{
		nombre: "archivos inválido", metodo: http.MethodDelete,
	})

	comprobarCaso(t, rec, casoPeticion{nombre: "archivos inválido", codigo: http.StatusBadRequest, campo: campoFiles})
}

// TestUsuarioLlave cubre la descarga de la llave privada.
func TestUsuarioLlave(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{archivo: "jorge_ed25519", contenido: []byte(llavePrivadaPrueba)}

	rec := ejecutarCasoConPath(t, NuevoUsuarios(servicio).Llave, rutaUsuarioJorge+"/key", campoName, usuarioJorge, casoPeticion{
		nombre: "descargar", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="jorge_ed25519"` {
		t.Errorf("Content-Disposition = %q", got)
	}

	if rec.Body.String() != llavePrivadaPrueba {
		t.Errorf("cuerpo = %q, want la llave privada", rec.Body.String())
	}
}

// TestUsuarioLlaveInexistente cubre la llave que no existe.
func TestUsuarioLlaveInexistente(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{descargaErr: services.ErrLlaveUsuarioNoEncontrada}

	rec := ejecutarCasoConPath(t, NuevoUsuarios(servicio).Llave, rutaUsuarioJorge+"/key", campoName, usuarioJorge, casoPeticion{
		nombre: "sin llave", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNotFound)
	}
}

// TestUsuarioLlaveError cubre un fallo inesperado al leer la llave.
func TestUsuarioLlaveError(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{descargaErr: errLectura}

	rec := ejecutarCasoConPath(t, NuevoUsuarios(servicio).Llave, rutaUsuarioJorge+"/key", campoName, usuarioJorge, casoPeticion{
		nombre: "error de lectura", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestUsuariosListarError cubre el fallo al listar cuentas.
func TestUsuariosListarError(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{err: errLectura}

	rec := ejecutarCaso(t, NuevoUsuarios(servicio).Coleccion, rutaUsuarios, casoPeticion{
		nombre: casoListar, metodo: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestUsuariosCrearError cubre el fallo de useradd.
func TestUsuariosCrearError(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{err: errLectura}

	rec := ejecutarCaso(t, NuevoUsuarios(servicio).Coleccion, rutaUsuarios, casoPeticion{
		nombre: "fallo de creación", metodo: http.MethodPost, cuerpo: `{"nombre":"jorge"}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestUsuarioErrorGenerico cubre un fallo inesperado al consultar.
func TestUsuarioErrorGenerico(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{err: errLectura}

	rec := ejecutarCasoConPath(t, NuevoUsuarios(servicio).Usuario, rutaUsuarioJorge, campoName, usuarioJorge, casoPeticion{
		nombre: "error de consulta", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestUsuarioEliminarNoEncontrado cubre la eliminación de una cuenta
// inexistente.
func TestUsuarioEliminarNoEncontrado(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{err: services.ErrUsuarioNoEncontrado}

	rec := ejecutarCasoConPath(t, NuevoUsuarios(servicio).Usuario, rutaUsuarioJorge, campoName, usuarioJorge, casoPeticion{
		nombre: "inexistente", metodo: http.MethodDelete,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNotFound)
	}
}

// TestUsuarioEliminarError cubre el fallo de userdel.
func TestUsuarioEliminarError(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{err: errLectura}

	rec := ejecutarCasoConPath(t, NuevoUsuarios(servicio).Usuario, rutaUsuarioJorge, campoName, usuarioJorge, casoPeticion{
		nombre: "fallo de eliminación", metodo: http.MethodDelete,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestUsuarioLlaveMetodo cubre el método no permitido al descargar.
func TestUsuarioLlaveMetodo(t *testing.T) {
	t.Parallel()

	rec := ejecutarCasoConPath(t, NuevoUsuarios(&stubUsuarios{}).Llave, rutaUsuarioJorge+"/key", campoName, usuarioJorge, casoPeticion{
		nombre: casoMetodoNoPermitido, metodo: http.MethodPost,
	})

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(msgCodigo, rec.Code, http.StatusMethodNotAllowed)
	}
}
