// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package httpcache

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

// Rutas y mensajes repetidos en las pruebas.
const (
	cabeceraTipo = "Content-Type"
	tipoJSON     = "application/json"
	rutaSesion   = "/api/session"
	rutaPaquetes = "/api/packages"
	rutaVMs      = "/api/vms"
	msgUnaVez    = "llamadas = %d, want 1"
	msgDosVeces  = "llamadas = %d, want 2"
)

// relojFalso permite avanzar el tiempo sin dormir.
type relojFalso struct {
	mu    sync.Mutex
	ahora time.Time
}

// Ahora devuelve la hora simulada.
func (r *relojFalso) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.ahora
}

// Avanzar mueve el reloj simulado.
func (r *relojFalso) Avanzar(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ahora = r.ahora.Add(d)
}

// contador cuenta las veces que el manejador de prueba fue invocado.
type contador struct {
	mu       sync.Mutex
	llamadas int
	lecturas int
}

// Manejador responde con un JSON que incluye el número de llamada.
func (c *contador) Manejador(estado int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.mu.Lock()
		c.llamadas++

		if r.Method == http.MethodGet {
			c.lecturas++
		}

		numero := c.llamadas
		c.mu.Unlock()

		w.Header().Set(cabeceraTipo, tipoJSON)
		w.WriteHeader(estado)

		if _, err := w.Write([]byte(`{"llamada":` + strconv.Itoa(numero) + `}`)); err != nil {
			return
		}
	})
}

// Lecturas devuelve cuántas lecturas llegaron al manejador.
func (c *contador) Lecturas() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.lecturas
}

// Total devuelve cuántas veces se invocó el manejador.
func (c *contador) Total() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.llamadas
}

// nuevoCache arma una caché con reloj falso y una política para /api.
func nuevoCache(ttl time.Duration) (*Cache, *relojFalso) {
	reloj := &relojFalso{ahora: time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)}
	cache := Nuevo([]Politica{{Prefijo: "/api", TTL: ttl}})
	cache.reloj = reloj.Ahora

	return cache, reloj
}

// TestCacheSirveLaSegundaLecturaSinLlamarAlManejador comprueba el acierto.
func TestCacheSirveLaSegundaLecturaSinLlamarAlManejador(t *testing.T) {
	t.Parallel()

	cache, _ := nuevoCache(10 * time.Second)
	contador := &contador{}
	manejador := cache.Envolver(contador.Manejador(http.StatusOK))

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		manejador.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaSesion, http.NoBody))

		if rec.Code != http.StatusOK {
			t.Fatalf("código = %d, want %d", rec.Code, http.StatusOK)
		}

		if got := rec.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
	}

	if contador.Total() != 1 {
		t.Errorf(msgUnaVez, contador.Total())
	}
}

// TestCacheDistingueLaCadenaDeConsulta comprueba que cada búsqueda es
// una entrada distinta.
func TestCacheDistingueLaCadenaDeConsulta(t *testing.T) {
	t.Parallel()

	cache, _ := nuevoCache(10 * time.Second)
	contador := &contador{}
	manejador := cache.Envolver(contador.Manejador(http.StatusOK))

	for _, ruta := range []string{"/api/vms?q=deb", "/api/vms?q=ubuntu", "/api/vms?q=deb"} {
		manejador.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, ruta, http.NoBody))
	}

	if contador.Total() != 2 {
		t.Errorf(msgDosVeces, contador.Total())
	}
}

// TestCacheExpira comprueba que una entrada vieja se vuelve a pedir.
func TestCacheExpira(t *testing.T) {
	t.Parallel()

	cache, reloj := nuevoCache(5 * time.Second)
	contador := &contador{}
	manejador := cache.Envolver(contador.Manejador(http.StatusOK))
	peticion := func() {
		manejador.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaPaquetes, http.NoBody))
	}

	peticion()
	peticion()

	if contador.Total() != 1 {
		t.Fatalf(msgUnaVez, contador.Total())
	}

	reloj.Avanzar(6 * time.Second)
	peticion()

	if contador.Total() != 2 {
		t.Errorf("llamadas = %d, want 2 tras expirar", contador.Total())
	}
}

// TestCacheSeInvalidaAlEscribir comprueba que una mutación vacía la caché.
func TestCacheSeInvalidaAlEscribir(t *testing.T) {
	t.Parallel()

	cache, _ := nuevoCache(time.Minute)
	contador := &contador{}
	manejador := cache.Envolver(contador.Manejador(http.StatusOK))

	manejador.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaPaquetes, http.NoBody))
	manejador.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaPaquetes, http.NoBody))

	if contador.Total() != 1 {
		t.Fatalf(msgUnaVez, contador.Total())
	}

	manejador.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodPost, rutaPaquetes, http.NoBody))
	manejador.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaPaquetes, http.NoBody))

	if contador.Total() != 3 {
		t.Errorf("llamadas = %d, want 3 (lectura y escritura)", contador.Total())
	}
}

// TestCacheNoGuardaErrores evita recordar un fallo transitorio.
func TestCacheNoGuardaErrores(t *testing.T) {
	t.Parallel()

	cache, _ := nuevoCache(time.Minute)
	contador := &contador{}
	manejador := cache.Envolver(contador.Manejador(http.StatusServiceUnavailable))

	manejador.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaSesion, http.NoBody))
	manejador.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaSesion, http.NoBody))

	if contador.Total() != 2 {
		t.Errorf(msgDosVeces, contador.Total())
	}
}

// TestCacheIgnoraLoQueNoEsDeLaAPI deja intactos los archivos estáticos.
func TestCacheIgnoraLoQueNoEsDeLaAPI(t *testing.T) {
	t.Parallel()

	cache, _ := nuevoCache(time.Minute)
	contador := &contador{}
	manejador := cache.Envolver(contador.Manejador(http.StatusOK))

	manejador.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/assets/main.js", http.NoBody))
	manejador.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/assets/main.js", http.NoBody))

	if contador.Total() != 2 {
		t.Errorf(msgDosVeces, contador.Total())
	}
}

// TestCacheRespetaNoCache permite forzar una lectura fresca.
func TestCacheRespetaNoCache(t *testing.T) {
	t.Parallel()

	cache, _ := nuevoCache(time.Minute)
	contador := &contador{}
	manejador := cache.Envolver(contador.Manejador(http.StatusOK))

	manejador.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaVMs, http.NoBody))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaVMs, http.NoBody)
	req.Header.Set("Cache-Control", "no-cache")
	manejador.ServeHTTP(httptest.NewRecorder(), req)

	if contador.Total() != 2 {
		t.Errorf(msgDosVeces, contador.Total())
	}
}

// TestCacheAnunciaElAcierto comprueba el encabezado de diagnóstico.
func TestCacheAnunciaElAcierto(t *testing.T) {
	t.Parallel()

	cache, _ := nuevoCache(time.Minute)
	manejador := cache.Envolver((&contador{}).Manejador(http.StatusOK))

	primera := httptest.NewRecorder()
	manejador.ServeHTTP(primera, httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaVMs, http.NoBody))

	segunda := httptest.NewRecorder()
	manejador.ServeHTTP(segunda, httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaVMs, http.NoBody))

	if got := primera.Header().Get(cabeceraCache); got != "MISS" {
		t.Errorf("%s = %q, want MISS", cabeceraCache, got)
	}

	if got := segunda.Header().Get(cabeceraCache); got != "HIT" {
		t.Errorf("%s = %q, want HIT", cabeceraCache, got)
	}
}

// TestCambiarDeMaquinaInvalidaLasLecturas reproduce el caso real: al
// vincular otra máquina se descarta todo lo que se había leído de la
// anterior.
func TestCambiarDeMaquinaInvalidaLasLecturas(t *testing.T) {
	t.Parallel()

	cache, _ := nuevoCache(time.Minute)
	contador := &contador{}
	manejador := cache.Envolver(contador.Manejador(http.StatusOK))

	leer := func() {
		manejador.ServeHTTP(httptest.NewRecorder(),
			httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaVMs, http.NoBody))
	}

	leer()
	leer()

	if contador.Lecturas() != 1 {
		t.Fatalf(msgUnaVez, contador.Lecturas())
	}

	manejador.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequestWithContext(context.Background(), http.MethodPut, rutaSesion, http.NoBody))

	leer()

	if contador.Lecturas() != 2 {
		t.Errorf("lecturas = %d, want 2 tras cambiar de máquina", contador.Lecturas())
	}
}

// TestRespuestaEnVueloNoContaminaLaCache reproduce la carrera que deja
// datos de la máquina anterior: una lectura lenta que empezó antes de
// cambiar de máquina no debe guardarse al terminar.
func TestRespuestaEnVueloNoContaminaLaCache(t *testing.T) {
	t.Parallel()

	cache, _ := nuevoCache(time.Minute)

	var (
		mu       sync.Mutex
		llamadas int
	)

	enVuelo := make(chan struct{})
	liberar := make(chan struct{})

	manejador := cache.Envolver(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		llamadas++
		numero := llamadas
		mu.Unlock()

		if numero == 1 {
			close(enVuelo)
			<-liberar
		}

		w.Header().Set(cabeceraTipo, tipoJSON)

		if _, err := w.Write([]byte(`{"llamada":` + strconv.Itoa(numero) + `}`)); err != nil {
			return
		}
	}))

	var espera sync.WaitGroup

	espera.Add(1)

	go func() {
		defer espera.Done()

		manejador.ServeHTTP(httptest.NewRecorder(),
			httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaPaquetes, http.NoBody))
	}()

	<-enVuelo

	// Cambiar de máquina mientras la lectura anterior sigue en curso.
	manejador.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequestWithContext(context.Background(), http.MethodPut, rutaSesion, http.NoBody))

	close(liberar)
	espera.Wait()

	// La siguiente lectura debe pedir los datos a la máquina nueva.
	manejador.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, rutaPaquetes, http.NoBody))

	mu.Lock()
	total := llamadas
	mu.Unlock()

	if total != 3 {
		t.Errorf("llamadas = %d, want 3 (lectura, cambio y lectura nueva)", total)
	}
}

// escritorQueFalla simula un cliente que corta la respuesta.
type escritorQueFalla struct {
	cabecera http.Header
	estado   int
}

// Header devuelve las cabeceras de la respuesta.
func (e *escritorQueFalla) Header() http.Header {
	if e.cabecera == nil {
		e.cabecera = make(http.Header)
	}

	return e.cabecera
}

// WriteHeader registra el estado y descarta la escritura.
func (e *escritorQueFalla) WriteHeader(codigo int) {
	if e.estado == 0 {
		e.estado = codigo
	}
}

// Write siempre falla, como una conexión cerrada.
func (e *escritorQueFalla) Write([]byte) (int, error) {
	return 0, errors.New("conexión cerrada")
}

// TestCacheGuardarSinPolitica comprueba que un recurso sin vigencia no se
// guarda.
func TestCacheGuardarSinPolitica(t *testing.T) {
	t.Parallel()

	cache := Nuevo(nil)
	cache.guardar("/otra/ruta", entrada{cuerpo: []byte("x")}, cache.generacionActual())

	if _, encontrada := cache.buscar("/otra/ruta"); encontrada {
		t.Error("buscar() = true, want false para un recurso sin política")
	}
}

// TestCacheGuardarIgnoraCuerpoGrande descarta respuestas desmedidas.
func TestCacheGuardarIgnoraCuerpoGrande(t *testing.T) {
	t.Parallel()

	cache, _ := nuevoCache(time.Minute)
	cache.guardar(rutaPaquetes, entrada{cuerpo: make([]byte, maximoCuerpo+1)}, cache.generacionActual())

	if _, encontrada := cache.buscar(rutaPaquetes); encontrada {
		t.Error("buscar() = true, want false para un cuerpo demasiado grande")
	}
}

// TestCacheGuardarDescartaGeneracionVieja evita que una lectura en vuelo
// repueble la caché después de una invalidación.
func TestCacheGuardarDescartaGeneracionVieja(t *testing.T) {
	t.Parallel()

	cache, _ := nuevoCache(time.Minute)
	generacion := cache.generacionActual()

	cache.vaciar()
	cache.guardar(rutaPaquetes, entrada{cuerpo: []byte("{")}, generacion)

	if _, encontrada := cache.buscar(rutaPaquetes); encontrada {
		t.Error("buscar() = true, want false para una generación anterior")
	}
}

// TestCacheReciclaEntradas vacía la caché cuando alcanza el límite.
func TestCacheReciclaEntradas(t *testing.T) {
	t.Parallel()

	cache, _ := nuevoCache(time.Minute)
	cache.generacion++
	generacion := cache.generacionActual()

	for i := 0; i < maximoEntradas; i++ {
		cache.entradas[rutaPaquetes+strconv.Itoa(i)] = entrada{cuerpo: []byte("x")}
	}

	cache.guardar(rutaPaquetes, entrada{cuerpo: []byte("y")}, generacion)

	if len(cache.entradas) != 1 {
		t.Errorf("entradas = %d, want 1 tras reciclar", len(cache.entradas))
	}
}

// TestVigenciaDePrefiereElPrefijoMasEspecifico resuelve la política más
// concreta.
func TestVigenciaDePrefiereElPrefijoMasEspecifico(t *testing.T) {
	t.Parallel()

	cache := Nuevo([]Politica{
		{Prefijo: "/api", TTL: time.Minute},
		{Prefijo: rutaSesion, TTL: 3 * time.Second},
	})

	if ttl := cache.vigenciaDe(rutaSesion + "/extra"); ttl != 3*time.Second {
		t.Errorf("vigenciaDe() = %v, want 3s", ttl)
	}

	if ttl := cache.vigenciaDe("/otra"); ttl != 0 {
		t.Errorf("vigenciaDe() = %v, want 0", ttl)
	}
}

// TestEscribirEntradaConClienteCaido comprueba que el acierto conserva
// sus cabeceras aunque el cliente corte la conexión.
func TestEscribirEntradaConClienteCaido(t *testing.T) {
	t.Parallel()

	escritor := &escritorQueFalla{}
	entrada := entrada{
		cuerpo:   []byte("x"),
		cabecera: http.Header{cabeceraTipo: []string{tipoJSON}},
		estado:   http.StatusOK,
	}

	escribirEntrada(escritor, entrada, "HIT")

	if escritor.estado != http.StatusOK {
		t.Errorf("estado = %d, want 200", escritor.estado)
	}

	if cabecera := escritor.Header().Get(cabeceraCache); cabecera != "HIT" {
		t.Errorf("%s = %q, want HIT", cabeceraCache, cabecera)
	}

	if tipo := escritor.Header().Get(cabeceraTipo); tipo != tipoJSON {
		t.Errorf("%s = %q, want %q", cabeceraTipo, tipo, tipoJSON)
	}
}

// TestGrabadorVolcarConClienteCaido comprueba que el fallo de escritura no
// pierde el estado ni las cabeceras de la respuesta.
func TestGrabadorVolcarConClienteCaido(t *testing.T) {
	t.Parallel()

	grabador := nuevoGrabador()
	grabador.Header().Set(cabeceraTipo, tipoJSON)

	if _, err := grabador.Write([]byte("x")); err != nil {
		t.Fatalf("grabador.Write() error = %v", err)
	}

	escritor := &escritorQueFalla{}
	grabador.volcar(escritor, "MISS")

	if escritor.estado != http.StatusOK {
		t.Errorf("estado = %d, want 200", escritor.estado)
	}

	if cabecera := escritor.Header().Get(cabeceraCache); cabecera != "MISS" {
		t.Errorf("%s = %q, want MISS", cabeceraCache, cabecera)
	}

	if tipo := escritor.Header().Get(cabeceraTipo); tipo != tipoJSON {
		t.Errorf("%s = %q, want %q", cabeceraTipo, tipo, tipoJSON)
	}
}
