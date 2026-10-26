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
	rutaHistorialPrueba = "/api/historial"
	rutaUnidadesPrueba  = "/api/units"
)

// escritorConFallo simula un cliente que corta la respuesta a mitad.
type escritorConFallo struct {
	cabecera http.Header
	estado   int
}

// Header devuelve las cabeceras de la respuesta.
func (e *escritorConFallo) Header() http.Header {
	if e.cabecera == nil {
		e.cabecera = make(http.Header)
	}

	return e.cabecera
}

// WriteHeader registra el primer estado y descarta la escritura.
func (e *escritorConFallo) WriteHeader(codigo int) {
	if e.estado == 0 {
		e.estado = codigo
	}
}

// Write siempre falla, como una conexión cerrada.
func (e *escritorConFallo) Write([]byte) (int, error) {
	return 0, errors.New("conexión cerrada")
}

// TestAccesoRegistrarErrorInterno devuelve 500 ante un fallo inesperado.
func TestAccesoRegistrarErrorInterno(t *testing.T) {
	t.Parallel()

	servicio := &stubAcceso{err: errors.New("base caída")}

	rec := ejecutarCaso(t, NuevoAcceso(servicio).ServeHTTP, rutaRegistro, casoPeticion{
		nombre: "error interno", metodo: http.MethodPost, cuerpo: `{"usuario":"root","password":"secreto"}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestAccesoOlvidar cubre el borrado del registro y sus errores.
func TestAccesoOlvidarRespuestas(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre   string
		err      error
		esperado int
	}{
		{"correcto", nil, http.StatusNoContent},
		{"sin máquina", services.ErrSinMaquina, http.StatusConflict},
		{"error interno", errors.New("base caída"), http.StatusInternalServerError},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			servicio := &stubAcceso{err: caso.err}

			rec := ejecutarCaso(t, NuevoAcceso(servicio).ServeHTTP, rutaRegistro, casoPeticion{
				nombre: caso.nombre, metodo: http.MethodDelete,
			})

			if rec.Code != caso.esperado {
				t.Errorf(msgCodigo, rec.Code, caso.esperado)
			}
		})
	}
}

// TestAuthErroresInternos cubre los fallos del servicio de autenticación.
func TestAuthErroresInternos(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	NuevoAuth(&stubConsola{configurarErr: errors.New("base caída")}).Estado(rec,
		peticionConCookie(http.MethodGet, rutaAuthEstado, "", ""))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Estado() = %d, want 500", rec.Code)
	}

	rec = httptest.NewRecorder()
	NuevoAuth(&stubConsola{configurarErr: errors.New("base caída")}).Configurar(rec,
		peticionConCookie(http.MethodPost, "/api/auth/setup", cuerpoCredenciales, ""))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Configurar() = %d, want 500", rec.Code)
	}

	rec = httptest.NewRecorder()
	NuevoAuth(&stubConsola{configurarErr: services.ErrDatosConsola}).Configurar(rec,
		peticionConCookie(http.MethodPost, "/api/auth/setup", cuerpoCredenciales, ""))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Configurar() con datos inválidos = %d, want 400", rec.Code)
	}

	rec = httptest.NewRecorder()
	NuevoAuth(&stubConsola{autenticarErr: errors.New("base caída")}).Entrar(rec,
		peticionConCookie(http.MethodPost, rutaAuthLogin, cuerpoCredenciales, ""))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Entrar() = %d, want 500", rec.Code)
	}

	rec = httptest.NewRecorder()
	NuevoAuth(&stubConsola{tokenErr: errors.New("sin secreto"), autenticado: true}).Entrar(rec,
		peticionConCookie(http.MethodPost, rutaAuthLogin, cuerpoCredenciales, ""))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Entrar() sin sesión = %d, want 500", rec.Code)
	}

	rec = httptest.NewRecorder()
	NuevoAuth(&stubConsola{}).Salir(rec,
		peticionConCookie(http.MethodGet, "/api/auth/logout", "", ""))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Salir() con GET = %d, want 405", rec.Code)
	}
}

// TestDiagnosticoErrorInterno devuelve 500 ante un fallo inesperado.
func TestDiagnosticoErrorInterno(t *testing.T) {
	t.Parallel()

	rec := ejecutarCaso(t, NuevoDiagnostico(&stubDiagnostico{err: errors.New("fallo")}).ServeHTTP,
		rutaDiagnostico, casoPeticion{nombre: "error", metodo: http.MethodGet})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestServidorInvalidoRechazaElFormato valida el identificador del
// back-end antes de tocar la máquina.
func TestServidorInvalidoRechazaElFormato(t *testing.T) {
	t.Parallel()

	rec := ejecutarCasoConPath(t, NuevoHaproxy(&stubHaproxy{}).Servidor, rutaServidores+"/x",
		campoServer, "192.168.1.61", casoPeticion{nombre: "sin puerto", metodo: http.MethodDelete})

	if rec.Code != http.StatusBadRequest {
		t.Errorf(msgCodigo, rec.Code, http.StatusBadRequest)
	}
}

// TestEncenderSinMaquinaRechazaLaRuta exige el identificador.
func TestEncenderSinMaquinaRechazaLaRuta(t *testing.T) {
	t.Parallel()

	rec := ejecutarCasoConPath(t, NuevoMaquinas(&stubMaquinas{}).Encender, "/api/vms//poweron",
		"vm", "", casoPeticion{nombre: "sin máquina", metodo: http.MethodPost})

	if rec.Code != http.StatusBadRequest {
		t.Errorf(msgCodigo, rec.Code, http.StatusBadRequest)
	}
}

// TestEncenderErrorInterno devuelve 500 ante un fallo inesperado.
func TestEncenderErrorInterno(t *testing.T) {
	t.Parallel()

	servicio := &stubMaquinas{errEncender: errors.New("hipervisor ocupado")}

	rec := ejecutarCasoConPath(t, NuevoMaquinas(servicio).Encender, "/api/vms/debian/poweron",
		"vm", maquinaDebian, casoPeticion{nombre: "error", metodo: http.MethodPost})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestVincularErrorInterno devuelve 500 ante un fallo inesperado.
func TestVincularErrorInterno(t *testing.T) {
	t.Parallel()

	vinculador := &stubVinculacion{err: errors.New("hipervisor ocupado")}
	handler := NuevoSesion(&stubDescubridor{}, ConVinculador(vinculador))

	rec := ejecutarCaso(t, handler.ServeHTTP, rutaSesion, casoPeticion{
		nombre: "vincular", metodo: http.MethodPut, cuerpo: `{"vm":"debian"}`,
	})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestSesionSinMaquinaVinculada devuelve 503 con un mensaje claro.
func TestSesionSinMaquinaVinculada(t *testing.T) {
	t.Parallel()

	handler := NuevoSesion(&stubDescubridor{err: services.ErrSinMaquina})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, rutaSesion, http.NoBody))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf(msgCodigo, rec.Code, http.StatusServiceUnavailable)
	}

	if !strings.Contains(rec.Body.String(), "vinculada") {
		t.Errorf("cuerpo = %q, want el mensaje de máquina sin vincular", rec.Body.String())
	}
}

// TestHistorialErrorInterno devuelve 500 cuando la consulta falla.
func TestHistorialErrorInterno(t *testing.T) {
	t.Parallel()

	rec := ejecutarCaso(t, NuevoHistorial(&stubHistorial{err: errors.New("base caída")}).ServeHTTP,
		rutaHistorialPrueba, casoPeticion{nombre: "error", metodo: http.MethodGet})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestHistorialConRegistroFallidoNoInterrumpe la operación.
func TestHistorialConRegistroFallidoNoInterrumpe(t *testing.T) {
	t.Parallel()

	servicio := &stubHistorial{err: errors.New("base caída")}
	manejador := ConHistorial(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), servicio, func() string { return maquinaDebian })

	rec := httptest.NewRecorder()
	manejador.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		rutaPaquetes, strings.NewReader(`{"nombre":"nano"}`)))

	if rec.Code != http.StatusNoContent {
		t.Errorf(msgCodigo, rec.Code, http.StatusNoContent)
	}
}

// TestRegistroWriterEscribeSinEstado fija 200 y propaga el fallo.
func TestRegistroWriterEscribeSinEstado(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	registro := &registroWriter{ResponseWriter: rec}

	if _, err := registro.Write([]byte("x")); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	if registro.estado != http.StatusOK {
		t.Errorf("estado = %d, want 200", registro.estado)
	}

	conFallo := &registroWriter{ResponseWriter: &escritorConFallo{}}
	if _, err := conFallo.Write([]byte("x")); err == nil {
		t.Error("Write() error = nil, want el fallo del cliente")
	}
}

// TestMotivoDeIgnoraCuerposAjenos deja vacío el motivo cuando no aplica.
func TestMotivoDeIgnoraCuerposAjenos(t *testing.T) {
	t.Parallel()

	if motivo := motivoDe(http.StatusOK, []byte(`{"mensaje":"x"}`)); motivo != "" {
		t.Errorf("motivoDe() = %q, want vacío para un 200", motivo)
	}

	if motivo := motivoDe(http.StatusBadRequest, nil); motivo != "" {
		t.Errorf("motivoDe() = %q, want vacío sin cuerpo", motivo)
	}

	if motivo := motivoDe(http.StatusBadRequest, []byte("no es json")); motivo != "" {
		t.Errorf("motivoDe() = %q, want vacío con JSON inválido", motivo)
	}
}

// TestRecortarLimitaElTexto sin partir caracteres.
func TestRecortarLimitaElTexto(t *testing.T) {
	t.Parallel()

	if texto := recortar("corto", 10); texto != "corto" {
		t.Errorf("recortar() = %q, want corto", texto)
	}

	if texto := recortar(strings.Repeat("á", 12), 5); texto != "ááááá…" {
		t.Errorf("recortar() = %q, want cinco caracteres y puntos suspensivos", texto)
	}
}

// TestNivelDeEstadoClasifica clasifica advertencias, errores e info.
func TestNivelDeEstadoClasifica(t *testing.T) {
	t.Parallel()

	casos := map[int]slog.Level{
		http.StatusOK:                  slog.LevelInfo,
		http.StatusNotFound:            slog.LevelWarn,
		http.StatusServiceUnavailable:  slog.LevelWarn,
		http.StatusInternalServerError: slog.LevelError,
	}

	for estado, esperado := range casos {
		if nivel := nivelDeEstado(estado); nivel != esperado {
			t.Errorf("nivelDeEstado(%d) = %v, want %v", estado, nivel, esperado)
		}
	}
}

// TestNuevaSolicitudConFalloDeAleatorio usa la hora como respaldo.
func TestNuevaSolicitudConFalloDeAleatorio(t *testing.T) {
	original := leerAleatorio

	t.Cleanup(func() { leerAleatorio = original })

	leerAleatorio = func([]byte) (int, error) { return 0, errors.New("sin entropía") }

	solicitud := nuevaSolicitud()
	if _, err := time.Parse("20060102T150405.000000", solicitud); err != nil {
		t.Errorf("nuevaSolicitud() = %q, want el respaldo con formato de hora: %v", solicitud, err)
	}
}

// TestConObservabilidadSinLogger usa el logger por omisión.
func TestConObservabilidadSinLogger(t *testing.T) {
	t.Parallel()

	manejador := ConObservabilidad(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), nil, nil)

	rec := httptest.NewRecorder()
	manejador.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		rutaSesion, http.NoBody))

	if rec.Code != http.StatusOK {
		t.Errorf(msgCodigo, rec.Code, http.StatusOK)
	}
}

// TestEscritorEstadoCapturaElError fija 200 y trunca el cuerpo de error.
func TestEscritorEstadoCapturaElError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	escritor := &escritorEstado{ResponseWriter: rec}

	cuerpo := strings.Repeat("x", maxCuerpoLog+10)

	if _, err := escritor.Write([]byte(cuerpo)); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	if escritor.estado != http.StatusOK {
		t.Errorf("estado = %d, want 200", escritor.estado)
	}

	conFallo := &escritorEstado{ResponseWriter: &escritorConFallo{}}
	if _, err := conFallo.Write([]byte("x")); err == nil {
		t.Error("Write() error = nil, want el fallo del cliente")
	}

	conError := &escritorEstado{ResponseWriter: httptest.NewRecorder()}
	conError.WriteHeader(http.StatusBadRequest)

	if _, err := conError.Write([]byte(cuerpo)); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	if conError.captura.Len() != maxCuerpoLog {
		t.Errorf("captura = %d bytes, want %d", conError.captura.Len(), maxCuerpoLog)
	}
}

// TestEscribirJSONConCuerpoNoSerializable solo deja constancia en el log.
func TestEscribirJSONConCuerpoNoSerializable(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	escribirJSON(rec, http.StatusOK, make(chan int))

	if rec.Code != http.StatusOK {
		t.Errorf(msgCodigo, rec.Code, http.StatusOK)
	}
}

// TestDescargarLlavesConClienteCaido comprueba que la descarga conserva
// las cabeceras aunque el cliente corte la conexión al escribir.
func TestDescargarLlavesConClienteCaido(t *testing.T) {
	t.Parallel()

	escritor := &escritorConFallo{}

	sistema := NuevoSistema(&stubPaquetes{}, &stubLlaves{nombre: "id_ed25519", contenido: []byte("k")},
		&stubIssue{}, &stubHostname{})
	sistema.descargarLlaveRoot(escritor)

	if escritor.estado != http.StatusOK {
		t.Errorf("descargarLlaveRoot() estado = %d, want 200", escritor.estado)
	}

	if tipo := escritor.Header().Get(cabeceraContentType); tipo != "application/octet-stream" {
		t.Errorf("%s = %q, want application/octet-stream", cabeceraContentType, tipo)
	}

	if disposicion := escritor.Header().Get("Content-Disposition"); !strings.Contains(disposicion, "id_ed25519") {
		t.Errorf("Content-Disposition = %q, want el nombre de la llave", disposicion)
	}

	usuarios := NuevoUsuarios(&stubUsuarios{archivo: "jorge_ed25519", contenido: []byte("k")})
	escritor = &escritorConFallo{}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, rutaUsuarioJorge+"/key", http.NoBody)
	req.SetPathValue(campoName, usuarioJorge)
	usuarios.Llave(escritor, req)

	if escritor.estado != http.StatusOK {
		t.Errorf("Llave() estado = %d, want 200", escritor.estado)
	}

	if disposicion := escritor.Header().Get("Content-Disposition"); !strings.Contains(disposicion, "jorge_ed25519") {
		t.Errorf("Content-Disposition = %q, want el nombre de la llave", disposicion)
	}
}

// TestUnidadesErrorInterno devuelve 500 cuando el listado falla.
func TestUnidadesErrorInterno(t *testing.T) {
	t.Parallel()

	rec := ejecutarCaso(t, NuevoSystemd(&stubSystemd{err: errors.New("systemd caído")}).Unidades,
		rutaUnidadesPrueba, casoPeticion{nombre: "error", metodo: http.MethodGet})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestCambiarEstadoSinObtener devuelve 500 si la relectura falla.
func TestCambiarEstadoSinObtener(t *testing.T) {
	t.Parallel()

	servicio := &stubSystemd{obtenerErr: errors.New("systemd caído")}

	rec := ejecutarCasoConPath(t, NuevoSystemd(servicio).Unidad, rutaUnidadesPrueba+"/"+unidadCatalogo,
		campoUnit, unidadCatalogo, casoPeticion{
			nombre: "cambiar estado", metodo: http.MethodPatch, cuerpo: `{"activo":true}`,
		})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestUsuarioErrorInterno devuelve 500 cuando la consulta falla.
func TestUsuarioErrorInterno(t *testing.T) {
	t.Parallel()

	servicio := &stubUsuarios{err: errors.New("máquina caída")}

	rec := ejecutarCasoConPath(t, NuevoUsuarios(servicio).Usuario, rutaUsuarioJorge,
		campoName, usuarioJorge, casoPeticion{nombre: "consultar", metodo: http.MethodGet})

	if rec.Code != http.StatusInternalServerError {
		t.Errorf(msgCodigo, rec.Code, http.StatusInternalServerError)
	}
}

// TestAyudantesDeHistorialCubrenLosValoresPorOmision comprueba las
// funciones que traducen el estado y el resultado.
func TestAyudantesDeHistorialCubrenLosValoresPorOmision(t *testing.T) {
	t.Parallel()

	if estado := estadoDe(&registroWriter{}); estado != http.StatusOK {
		t.Errorf("estadoDe() = %d, want 200", estado)
	}

	if resultado := resultadoDe(http.StatusNotFound); resultado != "error" {
		t.Errorf("resultadoDe() = %q, want error", resultado)
	}

	if nombre := nombreActivo(nil); nombre != "" {
		t.Errorf("nombreActivo(nil) = %q, want vacío", nombre)
	}

	if datos := leerCuerpo(&http.Request{}); len(datos) != 0 {
		t.Errorf("leerCuerpo() = %v, want vacío sin cuerpo", datos)
	}
}

// TestConObservabilidadConSolicitudLarga genera un identificador nuevo.
func TestConObservabilidadConSolicitudLarga(t *testing.T) {
	t.Parallel()

	manejador := ConObservabilidad(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), func() string { return maquinaDebian }, slog.New(slog.NewTextHandler(&strings.Builder{}, nil)))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, rutaSesion, http.NoBody)
	req.Header.Set(cabeceraSolicitud, strings.Repeat("x", 80))

	rec := httptest.NewRecorder()
	manejador.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf(msgCodigo, rec.Code, http.StatusNoContent)
	}

	if len(rec.Header().Get(cabeceraSolicitud)) != 16 {
		t.Errorf("solicitud = %q, want un identificador nuevo", rec.Header().Get(cabeceraSolicitud))
	}
}

// TestServidorEliminadoSinCuerpo responde 204 y registra la baja.
func TestServidorEliminadoSinCuerpo(t *testing.T) {
	t.Parallel()

	servicio := &stubHaproxy{}

	rec := ejecutarCasoConPath(t, NuevoHaproxy(servicio).Servidor, rutaServidores+"/"+servidorPrueba,
		campoServer, servidorPrueba, casoPeticion{
			nombre: "eliminar", metodo: http.MethodDelete,
		})

	if rec.Code != http.StatusNoContent {
		t.Errorf(msgCodigo, rec.Code, http.StatusNoContent)
	}

	if servicio.eliminados != 1 {
		t.Errorf("eliminados = %d, want 1", servicio.eliminados)
	}
}

// TestContextoConUsuario conserva el usuario autenticado.
func TestContextoConUsuario(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(t.Context(), claveUsuario{}, usuarioJorge)

	if usuario := UsuarioDe(ctx); usuario != usuarioJorge {
		t.Errorf("UsuarioDe() = %q, want %q", usuario, usuarioJorge)
	}
}

// TestNuevaSolicitudDevuelveHex comprueba el identificador normal.
func TestNuevaSolicitudDevuelveHex(t *testing.T) {
	t.Parallel()

	solicitud := nuevaSolicitud()
	if len(solicitud) != 16 {
		t.Fatalf("nuevaSolicitud() = %q, want 16 caracteres hex", solicitud)
	}

	if _, err := hex.DecodeString(solicitud); err != nil {
		t.Errorf("nuevaSolicitud() = %q, want hexadecimal válido: %v", solicitud, err)
	}
}

// lectorConFallo devuelve un error al leer el cuerpo.
type lectorConFallo struct{}

// Read siempre falla.
func (lectorConFallo) Read([]byte) (int, error) { return 0, errors.New("cuerpo roto") }

// Close no hace nada.
func (lectorConFallo) Close() error { return nil }

// TestMetodosNoPermitidosEnAcceso cubre el 405 de los recursos de la
// consola y de los módulos sin método alternativo.
func TestMetodosNoPermitidosEnAcceso(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nombre  string
		handler http.HandlerFunc
		ruta    string
	}{
		{"configurar", NuevoAuth(&stubConsola{}).Configurar, "/api/auth/setup"},
		{"entrar", NuevoAuth(&stubConsola{}).Entrar, rutaAuthLogin},
		{"unidades", NuevoSystemd(&stubSystemd{}).Unidades, rutaUnidadesPrueba},
		{"usuario", NuevoUsuarios(&stubUsuarios{}).Usuario, rutaUsuarioJorge},
		{"llave", NuevoUsuarios(&stubUsuarios{}).Llave, rutaUsuarioJorge + "/key"},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			rec := ejecutarCaso(t, caso.handler, caso.ruta, casoPeticion{
				nombre: caso.nombre, metodo: http.MethodPut,
			})

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf(msgCodigo, rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

// TestEntrarConCredencialesInvalidas rechaza el cuerpo antes de consultar.
func TestEntrarConCredencialesInvalidas(t *testing.T) {
	t.Parallel()

	rec := ejecutarCaso(t, NuevoAuth(&stubConsola{}).Entrar, rutaAuthLogin, casoPeticion{
		nombre: "usuario inválido", metodo: http.MethodPost,
		cuerpo: `{"usuario":"Jorge","password":"una-clave-larga"}`,
	})

	if rec.Code != http.StatusBadRequest {
		t.Errorf(msgCodigo, rec.Code, http.StatusBadRequest)
	}
}

// TestCambiarEstadoConCuerpoInvalido rechaza un JSON mal formado.
func TestCambiarEstadoConCuerpoInvalido(t *testing.T) {
	t.Parallel()

	rec := ejecutarCasoConPath(t, NuevoSystemd(&stubSystemd{}).Unidad,
		rutaUnidadesPrueba+"/"+unidadCatalogo, campoUnit, unidadCatalogo, casoPeticion{
			nombre: "cuerpo inválido", metodo: http.MethodPatch, cuerpo: `{`,
		})

	if rec.Code != http.StatusBadRequest {
		t.Errorf(msgCodigo, rec.Code, http.StatusBadRequest)
	}
}

// TestUsuarioConNombreInvalido rechaza el identificador de la ruta.
func TestUsuarioConNombreInvalido(t *testing.T) {
	t.Parallel()

	rec := ejecutarCasoConPath(t, NuevoUsuarios(&stubUsuarios{}).Usuario,
		rutaUsuarioJorge, campoName, usuarioJorgeMayus, casoPeticion{
			nombre: "nombre inválido", metodo: http.MethodGet,
		})

	if rec.Code != http.StatusBadRequest {
		t.Errorf(msgCodigo, rec.Code, http.StatusBadRequest)
	}

	rec = ejecutarCasoConPath(t, NuevoUsuarios(&stubUsuarios{}).Llave,
		rutaUsuarioJorge+"/key", campoName, usuarioJorgeMayus, casoPeticion{
			nombre: "llave con nombre inválido", metodo: http.MethodGet,
		})

	if rec.Code != http.StatusBadRequest {
		t.Errorf(msgCodigo, rec.Code, http.StatusBadRequest)
	}
}

// TestLeerCuerpoConFalloDeLectura devuelve un mapa vacío.
func TestLeerCuerpoConFalloDeLectura(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, rutaPaquetes, lectorConFallo{})
	req.Header.Set(cabeceraContentType, tipoJSONPrueba)

	if datos := leerCuerpo(req); len(datos) != 0 {
		t.Errorf("leerCuerpo() = %v, want vacío con un cuerpo ilegible", datos)
	}
}

// TestObservabilidadConRespuestaSinEstado supone 200 cuando el handler
// no escribe nada.
func TestObservabilidadConRespuestaSinEstado(t *testing.T) {
	t.Parallel()

	manejador := ConObservabilidad(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		nil, slog.New(slog.NewTextHandler(&strings.Builder{}, nil)))

	rec := httptest.NewRecorder()
	manejador.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		rutaSesion, http.NoBody))

	if rec.Code != http.StatusOK {
		t.Errorf(msgCodigo, rec.Code, http.StatusOK)
	}
}
