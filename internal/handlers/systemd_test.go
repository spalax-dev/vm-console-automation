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

// errUnidad simula un fallo inesperado del servicio de systemd.
var errUnidad = errors.New("fallo de systemd")

// stubSystemd es un servicio de systemd falso con resultados
// preprogramados.
type stubSystemd struct {
	lista      []services.EstadoUnidad
	estado     services.EstadoUnidad
	resultado  services.Resultado
	err        error
	obtenerErr error
	unidad     string
	archivos   []services.ArchivoDespliegue
	usuario    string
	grupo      string
	activo     *bool
	habilitado *bool
	reinicios  int
}

// Listar devuelve la lista preprogramada.
func (s *stubSystemd) Listar(context.Context) ([]services.EstadoUnidad, error) {
	return s.lista, s.err
}

// Obtener devuelve el estado preprogramado.
func (s *stubSystemd) Obtener(_ context.Context, unidad string) (services.EstadoUnidad, error) {
	s.unidad = unidad

	return s.estado, s.obtenerErr
}

// Desplegar registra los argumentos y devuelve el resultado.
func (s *stubSystemd) Desplegar(
	_ context.Context, unidad string, archivos []services.ArchivoDespliegue, usuario, grupo string,
) (services.Resultado, error) {
	s.unidad = unidad
	s.archivos = archivos
	s.usuario = usuario
	s.grupo = grupo

	return s.resultado, s.err
}

// CambiarEstado registra los argumentos y devuelve el resultado.
func (s *stubSystemd) CambiarEstado(
	_ context.Context, unidad string, activo, habilitado *bool,
) (services.Resultado, error) {
	s.unidad = unidad
	s.activo = activo
	s.habilitado = habilitado

	return s.resultado, s.err
}

// Reiniciar registra la llamada y devuelve el resultado.
func (s *stubSystemd) Reiniciar(_ context.Context, unidad string) (services.Resultado, error) {
	s.unidad = unidad
	s.reinicios++

	return s.resultado, s.err
}

// TestUnidadesListar cubre el listado de unidades.
func TestUnidadesListar(t *testing.T) {
	t.Parallel()

	servicio := &stubSystemd{lista: []services.EstadoUnidad{
		{Unidad: unidadCatalogo, Activo: true, Estado: estadoActivoPrueba, Detalle: "running"},
	}}

	rec := ejecutarCaso(t, NuevoSystemd(servicio).Unidades, "/api/units", casoPeticion{
		nombre: casoListar, metodo: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	var cuerpo listaUnidades
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if len(cuerpo.Unidades) != 1 || !cuerpo.Unidades[0].Activo {
		t.Errorf("unidades = %+v, want la unidad activa", cuerpo.Unidades)
	}
}

// TestUnidadObtener cubre la consulta del estado de una unidad.
func TestUnidadObtener(t *testing.T) {
	t.Parallel()

	servicio := &stubSystemd{estado: services.EstadoUnidad{
		Unidad: unidadCatalogo, Activo: true, Habilitado: true, Estado: estadoActivoPrueba, Detalle: "running",
	}}

	rec := ejecutarCasoConPath(t, NuevoSystemd(servicio).Unidad, rutaUnidad, campoUnit, unidadCatalogo, casoPeticion{
		nombre: casoConsultar, metodo: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	var cuerpo unidadRespuesta
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if !cuerpo.Activo || !cuerpo.Habilitado {
		t.Errorf("unidad = %+v, want activa y habilitada", cuerpo)
	}
}

// TestUnidadNoEncontrada cubre la consulta de una unidad inexistente.
func TestUnidadNoEncontrada(t *testing.T) {
	t.Parallel()

	servicio := &stubSystemd{obtenerErr: services.ErrUnidadNoEncontrada}

	rec := ejecutarCasoConPath(t, NuevoSystemd(servicio).Unidad, rutaUnidad, campoUnit, unidadCatalogo, casoPeticion{
		nombre: casoInexistente, metodo: http.MethodGet,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNotFound)
	}
}

// TestUnidadError cubre un fallo inesperado al consultar.
func TestUnidadError(t *testing.T) {
	t.Parallel()

	servicio := &stubSystemd{obtenerErr: errUnidad}

	rec := ejecutarCasoConPath(t, NuevoSystemd(servicio).Unidad, rutaUnidad, campoUnit, unidadCatalogo, casoPeticion{
		nombre: "error", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestUnidadDesplegar cubre la transferencia de archivos.
func TestUnidadDesplegar(t *testing.T) {
	t.Parallel()

	servicio := &stubSystemd{resultado: services.Resultado{Mensaje: "Unidad desplegada."}}

	cuerpo := `{"archivos":[{"origen":"./app/catalogo","destino":"/opt/catalogo/catalogo","modo":"755"}],"usuario":"catalogo","grupo":"catalogo"}`

	rec := ejecutarCasoConPath(t, NuevoSystemd(servicio).Unidad, rutaUnidad, campoUnit, unidadCatalogo, casoPeticion{
		nombre: "desplegar", metodo: http.MethodPut, cuerpo: cuerpo,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	if len(servicio.archivos) != 1 || servicio.archivos[0].Destino != "/opt/catalogo/catalogo" {
		t.Errorf("archivos = %+v, want el archivo transferido", servicio.archivos)
	}

	if servicio.usuario != propietarioCatalogo || servicio.grupo != propietarioCatalogo {
		t.Errorf("propietario = %s:%s, want catalogo:catalogo", servicio.usuario, servicio.grupo)
	}
}

// TestUnidadCambiarEstado cubre la activación de una unidad.
func TestUnidadCambiarEstado(t *testing.T) {
	t.Parallel()

	servicio := &stubSystemd{estado: services.EstadoUnidad{
		Unidad: unidadCatalogo, Activo: true, Estado: estadoActivoPrueba,
	}}

	rec := ejecutarCasoConPath(t, NuevoSystemd(servicio).Unidad, rutaUnidad, campoUnit, unidadCatalogo, casoPeticion{
		nombre: "activar", metodo: http.MethodPatch, cuerpo: cuerpoActivar,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	if servicio.activo == nil || !*servicio.activo {
		t.Errorf("activo = %v, want true", servicio.activo)
	}

	var cuerpo unidadRespuesta
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if !cuerpo.Activo {
		t.Errorf("estado = %+v, want activa", cuerpo)
	}
}

// TestUnidadValidacion cubre las reglas de unidad, archivos y estado.
func TestUnidadValidacion(t *testing.T) {
	t.Parallel()

	handler := NuevoSystemd(&stubSystemd{}).Unidad

	casos := []struct {
		casoPeticion
		unidad string
	}{
		{casoPeticion{nombre: casoUnidadInvalida, metodo: http.MethodGet, codigo: http.StatusBadRequest, campo: campoUnit}, "../etc/passwd"},
		{
			casoPeticion{
				nombre: "sin archivos",
				metodo: http.MethodPut,
				cuerpo: `{"archivos":[]}`,
				codigo: http.StatusBadRequest,
				campo:  campoArchivos,
			},
			unidadCatalogo,
		},
		{
			casoPeticion{
				nombre: "destino relativo",
				metodo: http.MethodPut,
				cuerpo: `{"archivos":[{"origen":"./app","destino":"opt/app","modo":"755"}]}`,
				codigo: http.StatusBadRequest,
				campo:  campoDestino,
			},
			unidadCatalogo,
		},
		{
			casoPeticion{
				nombre: "modo inválido",
				metodo: http.MethodPut,
				cuerpo: `{"archivos":[{"origen":"./app","destino":"/opt/app","modo":"rwx"}]}`,
				codigo: http.StatusBadRequest,
				campo:  campoModo,
			},
			unidadCatalogo,
		},
		{
			casoPeticion{
				nombre: casoUsuarioInvalido,
				metodo: http.MethodPut,
				cuerpo: `{"archivos":[{"origen":"./app","destino":"/opt/app","modo":"755"}],"usuario":"José"}`,
				codigo: http.StatusBadRequest,
				campo:  "usuario",
			},
			unidadCatalogo,
		},
		{
			casoPeticion{
				nombre: "sin estado",
				metodo: http.MethodPatch,
				cuerpo: `{}`,
				codigo: http.StatusBadRequest,
				campo:  campoActivo,
			},
			unidadCatalogo,
		},
		{
			casoPeticion{
				nombre: "dos estados",
				metodo: http.MethodPatch,
				cuerpo: `{"activo":true,"habilitado":true}`,
				codigo: http.StatusBadRequest,
				campo:  campoHabilitado,
			},
			unidadCatalogo,
		},
		{casoPeticion{nombre: casoCuerpoInvalido, metodo: http.MethodPut, cuerpo: `{`, codigo: http.StatusBadRequest}, unidadCatalogo},
		{casoPeticion{nombre: casoMetodoNoPermitido, metodo: http.MethodDelete, codigo: http.StatusMethodNotAllowed}, unidadCatalogo},
	}

	for _, caso := range casos {
		rec := ejecutarCasoConPath(t, handler, rutaUnidad, campoUnit, caso.unidad, caso.casoPeticion)
		comprobarCaso(t, rec, caso.casoPeticion)
	}
}

// TestReinicios cubre el reinicio de una unidad.
func TestReinicios(t *testing.T) {
	t.Parallel()

	servicio := &stubSystemd{resultado: services.Resultado{Mensaje: "Unidad reiniciada."}}

	rec := ejecutarCasoConPath(t, NuevoSystemd(servicio).Reinicios, rutaReinicios, campoUnit, unidadCatalogo, casoPeticion{
		nombre: "reiniciar", metodo: http.MethodPost,
	})

	if rec.Code != http.StatusAccepted {
		t.Fatalf(msgCodigo, rec.Code, http.StatusAccepted)
	}

	if servicio.reinicios != 1 {
		t.Errorf("Reiniciar() se llamó %d veces, want 1", servicio.reinicios)
	}
}

// TestReiniciosNoEncontrada cubre el reinicio de una unidad inexistente.
func TestReiniciosNoEncontrada(t *testing.T) {
	t.Parallel()

	servicio := &stubSystemd{err: services.ErrUnidadNoEncontrada}

	rec := ejecutarCasoConPath(t, NuevoSystemd(servicio).Reinicios, rutaReinicios, campoUnit, unidadCatalogo, casoPeticion{
		nombre: casoInexistente, metodo: http.MethodPost,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNotFound)
	}
}

// TestUnidadesListarError cubre el fallo al listar unidades.
func TestUnidadesListarError(t *testing.T) {
	t.Parallel()

	rec := ejecutarCaso(t, NuevoSystemd(&stubSystemd{err: errUnidad}).Unidades, "/api/units", casoPeticion{
		nombre: casoListar, metodo: http.MethodGet,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestUnidadDesplegarError cubre el fallo al transferir los archivos.
func TestUnidadDesplegarError(t *testing.T) {
	t.Parallel()

	servicio := &stubSystemd{err: errUnidad}
	cuerpo := `{"archivos":[{"origen":"./app","destino":"/opt/app","modo":"755"}]}`

	rec := ejecutarCasoConPath(t, NuevoSystemd(servicio).Unidad, rutaUnidad, campoUnit, unidadCatalogo, casoPeticion{
		nombre: "fallo de despliegue", metodo: http.MethodPut, cuerpo: cuerpo,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestUnidadCambiarEstadoError cubre el fallo de systemctl al cambiar el
// estado.
func TestUnidadCambiarEstadoError(t *testing.T) {
	t.Parallel()

	servicio := &stubSystemd{err: errUnidad}

	rec := ejecutarCasoConPath(t, NuevoSystemd(servicio).Unidad, rutaUnidad, campoUnit, unidadCatalogo, casoPeticion{
		nombre: "fallo de estado", metodo: http.MethodPatch, cuerpo: cuerpoActivar,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestUnidadCambiarEstadoSinConsulta cubre el fallo al releer el estado.
func TestUnidadCambiarEstadoSinConsulta(t *testing.T) {
	t.Parallel()

	servicio := &stubSystemd{obtenerErr: errUnidad}

	rec := ejecutarCasoConPath(t, NuevoSystemd(servicio).Unidad, rutaUnidad, campoUnit, unidadCatalogo, casoPeticion{
		nombre: "fallo de consulta", metodo: http.MethodPatch, cuerpo: cuerpoActivar,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestReiniciosError cubre el fallo al reiniciar una unidad.
func TestReiniciosError(t *testing.T) {
	t.Parallel()

	rec := ejecutarCasoConPath(t, NuevoSystemd(&stubSystemd{err: errUnidad}).Reinicios, rutaReinicios, campoUnit, unidadCatalogo, casoPeticion{
		nombre: "fallo de reinicio", metodo: http.MethodPost,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestUnidadCambiarEstadoNoEncontrada cubre el cambio de estado de una
// unidad inexistente.
func TestUnidadCambiarEstadoNoEncontrada(t *testing.T) {
	t.Parallel()

	servicio := &stubSystemd{err: services.ErrUnidadNoEncontrada}

	rec := ejecutarCasoConPath(t, NuevoSystemd(servicio).Unidad, rutaUnidad, campoUnit, unidadCatalogo, casoPeticion{
		nombre: "inexistente", metodo: http.MethodPatch, cuerpo: cuerpoActivar,
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf(msgCodigo, rec.Code, http.StatusNotFound)
	}
}

// TestUnidadDesplegarGrupoInvalido cubre el grupo no válido.
func TestUnidadDesplegarGrupoInvalido(t *testing.T) {
	t.Parallel()

	cuerpo := `{"archivos":[{"origen":"./app","destino":"/opt/app","modo":"755"}],"grupo":"José"}`

	rec := ejecutarCasoConPath(t, NuevoSystemd(&stubSystemd{}).Unidad, rutaUnidad, campoUnit, unidadCatalogo, casoPeticion{
		nombre: "grupo inválido", metodo: http.MethodPut, cuerpo: cuerpo,
	})

	comprobarCaso(t, rec, casoPeticion{nombre: "grupo inválido", codigo: http.StatusBadRequest, campo: "grupo"})
}

// TestUnidadDesplegarSinOrigen cubre el archivo sin ruta de origen.
func TestUnidadDesplegarSinOrigen(t *testing.T) {
	t.Parallel()

	cuerpo := `{"archivos":[{"origen":"","destino":"/opt/app","modo":"755"}]}`

	rec := ejecutarCasoConPath(t, NuevoSystemd(&stubSystemd{}).Unidad, rutaUnidad, campoUnit, unidadCatalogo, casoPeticion{
		nombre: "sin origen", metodo: http.MethodPut, cuerpo: cuerpo,
	})

	comprobarCaso(t, rec, casoPeticion{nombre: "sin origen", codigo: http.StatusBadRequest, campo: campoOrigen})
}

// TestReiniciosValidacion cubre el método y la unidad inválidos.
func TestReiniciosValidacion(t *testing.T) {
	t.Parallel()

	handler := NuevoSystemd(&stubSystemd{}).Reinicios

	metodo := ejecutarCasoConPath(t, handler, rutaReinicios, campoUnit, unidadCatalogo, casoPeticion{
		nombre: casoMetodoNoPermitido, metodo: http.MethodGet,
	})
	comprobarCaso(t, metodo, casoPeticion{nombre: casoMetodoNoPermitido, codigo: http.StatusMethodNotAllowed})

	invalida := ejecutarCasoConPath(t, handler, rutaReinicios, campoUnit, "catalogo", casoPeticion{
		nombre: casoUnidadInvalida, metodo: http.MethodPost,
	})
	comprobarCaso(t, invalida, casoPeticion{nombre: casoUnidadInvalida, codigo: http.StatusBadRequest, campo: campoUnit})
}
