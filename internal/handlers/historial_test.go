// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"consola-vm/internal/services"
)

// descripcionPaqueteNano es el resumen esperado al instalar nano.
const descripcionPaqueteNano = "Instalar el paquete «nano»"

// tipoJSONPrueba es el tipo de contenido de las peticiones de prueba.
const tipoJSONPrueba = "application/json"

// msgRegistrarLlamadas es el formato para comprobar las llamadas a Registrar.
const msgRegistrarLlamadas = "Registrar() se llamó %d veces, want 1"

// stubHistorial es un historial falso con resultados preprogramados.
type stubHistorial struct {
	lista       []services.Operacion
	err         error
	filtro      services.FiltroHistorial
	registradas []services.Operacion
}

// Listar devuelve la lista preprogramada.
func (s *stubHistorial) Listar(_ context.Context, filtro *services.FiltroHistorial) ([]services.Operacion, error) {
	s.filtro = *filtro

	return s.lista, s.err
}

// Registrar acumula las operaciones registradas.
func (s *stubHistorial) Registrar(_ context.Context, operacion *services.Operacion) error {
	s.registradas = append(s.registradas, *operacion)

	return s.err
}

// TestHistorialListar cubre la consulta y los filtros.
func TestHistorialListar(t *testing.T) {
	t.Parallel()

	servicio := &stubHistorial{lista: []services.Operacion{
		{ID: 1, FechaHora: "2026-10-25T10:00:00Z", VM: maquinaDebian, Modulo: "paquetes", Operacion: "POST /api/packages", Descripcion: descripcionPaqueteNano, Resultado: "ok"},
	}}

	ruta := "/api/historial?q=nano&vm=debian-1&modulo=paquetes&desde=2026-10-25&hasta=2026-10-26&hora_desde=08:00&hora_hasta=18:00&limite=50"

	rec := ejecutarCaso(t, NuevoHistorial(servicio).ServeHTTP, ruta, casoPeticion{
		nombre: casoListar, metodo: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	if servicio.filtro.Texto != "nano" || servicio.filtro.VM != maquinaDebian || servicio.filtro.Limite != 50 {
		t.Errorf("filtro = %+v, want los parámetros de la query", servicio.filtro)
	}

	var cuerpo listaHistorial
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if len(cuerpo.Operaciones) != 1 || cuerpo.Operaciones[0].VM != maquinaDebian {
		t.Errorf("operaciones = %+v, want debian-1", cuerpo.Operaciones)
	}

	if cuerpo.Operaciones[0].Descripcion != descripcionPaqueteNano {
		t.Errorf("descripcion = %q, want el resumen del historial", cuerpo.Operaciones[0].Descripcion)
	}
}

// TestHistorialValidacion cubre fechas, horas, límite y método.
func TestHistorialValidacion(t *testing.T) {
	t.Parallel()

	handler := NuevoHistorial(&stubHistorial{}).ServeHTTP

	casos := []struct {
		casoPeticion
		ruta string
	}{
		{casoPeticion{nombre: "fecha inválida", metodo: http.MethodGet, codigo: http.StatusBadRequest, campo: "desde"}, "/api/historial?desde=25-10-2026"},
		{casoPeticion{nombre: "hora inválida", metodo: http.MethodGet, codigo: http.StatusBadRequest, campo: "hora_hasta"}, "/api/historial?hora_hasta=25:00"},
		{casoPeticion{nombre: "límite inválido", metodo: http.MethodGet, codigo: http.StatusBadRequest, campo: "limite"}, "/api/historial?limite=0"},
		{casoPeticion{nombre: casoMetodoNoPermitido, metodo: http.MethodPost, codigo: http.StatusMethodNotAllowed}, "/api/historial"},
	}

	for _, caso := range casos {
		rec := ejecutarCaso(t, handler, caso.ruta, caso.casoPeticion)
		comprobarCaso(t, rec, caso.casoPeticion)
	}
}

// TestConHistorialRegistraMutaciones comprueba que solo se registran las
// operaciones que no son GET.
func TestConHistorialRegistraMutaciones(t *testing.T) {
	t.Parallel()

	servicio := &stubHistorial{}
	handler := ConHistorial(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
		servicio,
		func() string { return maquinaDebian },
	)

	post := httptest.NewRequestWithContext(t.Context(), http.MethodPost, rutaPaquetes, http.NoBody)
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, post)

	if len(servicio.registradas) != 1 {
		t.Fatalf(msgRegistrarLlamadas, len(servicio.registradas))
	}

	if servicio.registradas[0].Resultado != "ok" || servicio.registradas[0].VM != maquinaDebian {
		t.Errorf("operación = %+v, want ok en debian-1", servicio.registradas[0])
	}

	get := httptest.NewRequestWithContext(t.Context(), http.MethodGet, rutaPaquetes, http.NoBody)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, get)

	if len(servicio.registradas) != 1 {
		t.Errorf("una consulta GET se registró en el historial")
	}
}

// TestConHistorialDescribeYRestauraElCuerpo comprueba que el resumen usa
// el cuerpo sin consumirlo para el handler ni guardar secretos.
func TestConHistorialDescribeYRestauraElCuerpo(t *testing.T) {
	t.Parallel()

	servicio := &stubHistorial{}
	recibido := map[string]any{}

	handler := ConHistorial(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cuerpo, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("leer el cuerpo restaurado: %v", err)
			}

			if err := json.Unmarshal(cuerpo, &recibido); err != nil {
				t.Errorf("decodificar el cuerpo restaurado: %v", err)
			}

			w.WriteHeader(http.StatusCreated)
		}),
		servicio,
		func() string { return maquinaDebian },
	)

	peticion := httptest.NewRequestWithContext(t.Context(), http.MethodPost, rutaPaquetes,
		strings.NewReader(`{"nombre":"nano","password":"secreto"}`))
	peticion.Header.Set(cabeceraContentType, tipoJSONPrueba)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticion)

	if recibido["nombre"] != "nano" {
		t.Errorf("el handler recibió %+v, want el cuerpo restaurado", recibido)
	}

	if len(servicio.registradas) != 1 {
		t.Fatalf(msgRegistrarLlamadas, len(servicio.registradas))
	}

	operacion := servicio.registradas[0]
	if operacion.Descripcion != descripcionPaqueteNano {
		t.Errorf("Descripcion = %q, want el resumen del paquete", operacion.Descripcion)
	}

	if strings.Contains(operacion.Descripcion, "secreto") || strings.Contains(operacion.Operacion, "secreto") {
		t.Error("el historial guardó la contraseña")
	}
}

// TestConHistorialUsaLaMaquinaObjetivo comprueba que las operaciones de
// sesión y de encendido se atribuyen a la máquina solicitada.
func TestConHistorialUsaLaMaquinaObjetivo(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre   string
		metodo   string
		ruta     string
		cuerpo   string
		esperada string
	}{
		{
			nombre:   "registro",
			metodo:   http.MethodPost,
			ruta:     "/api/session/register",
			cuerpo:   `{"vm":"ubuntu-lab","usuario":"jorge"}`,
			esperada: "ubuntu-lab",
		},
		{
			nombre:   "encendido",
			metodo:   http.MethodPost,
			ruta:     "/api/vms/ubuntu-lab/poweron",
			esperada: "ubuntu-lab",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			servicio := &stubHistorial{}
			handler := ConHistorial(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusAccepted)
				}),
				servicio,
				func() string { return maquinaDebian },
			)

			var cuerpo io.Reader = http.NoBody
			if caso.cuerpo != "" {
				cuerpo = io.NopCloser(strings.NewReader(caso.cuerpo))
			}

			peticion := httptest.NewRequestWithContext(t.Context(), caso.metodo, caso.ruta, cuerpo)
			if caso.cuerpo != "" {
				peticion.Header.Set(cabeceraContentType, tipoJSONPrueba)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, peticion)

			if len(servicio.registradas) != 1 {
				t.Fatalf(msgRegistrarLlamadas, len(servicio.registradas))
			}

			if obtenida := servicio.registradas[0].VM; obtenida != caso.esperada {
				t.Errorf("VM = %q, want %q", obtenida, caso.esperada)
			}
		})
	}
}

// TestConHistorialCuerpoNulo comprueba que un JSON null no rompe el
// middleware ni el resumen.
func TestConHistorialCuerpoNulo(t *testing.T) {
	t.Parallel()

	servicio := &stubHistorial{}
	handler := ConHistorial(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
		servicio,
		func() string { return maquinaDebian },
	)

	peticion := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		rutaPaquetes+"?files=true", strings.NewReader("null"))
	peticion.Header.Set(cabeceraContentType, tipoJSONPrueba)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticion)

	if len(servicio.registradas) != 1 {
		t.Fatalf(msgRegistrarLlamadas, len(servicio.registradas))
	}

	if obtenido := servicio.registradas[0].Descripcion; obtenido != "Instalar un paquete" {
		t.Errorf("Descripcion = %q, want el resumen genérico", obtenido)
	}
}

// TestConHistorialDescribeElBorradoConArchivos comprueba que la opción
// files de la query se refleja en el resumen.
func TestConHistorialDescribeElBorradoConArchivos(t *testing.T) {
	t.Parallel()

	servicio := &stubHistorial{}
	handler := ConHistorial(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		servicio,
		func() string { return maquinaDebian },
	)

	peticion := httptest.NewRequestWithContext(t.Context(), http.MethodDelete,
		"/api/users/jorge?files=true", http.NoBody)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticion)

	if len(servicio.registradas) != 1 {
		t.Fatalf(msgRegistrarLlamadas, len(servicio.registradas))
	}

	if obtenido := servicio.registradas[0].Descripcion; obtenido != "Eliminar el usuario «jorge» y sus archivos" {
		t.Errorf("Descripcion = %q, want el borrado con archivos", obtenido)
	}
}
