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
	headerKind   = "Content-Type"
	kindJSON     = "application/json"
	pathSession  = "/api/session"
	pathPackages = "/api/packages"
	pathVMs      = "/api/vms"
	msgTime      = "llamadas = %d, want 1"
	msgDosTimes  = "llamadas = %d, want 2"
)

// fakeClock permite avanzar el tiempo sin dormir.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// Now devuelve la hora simulada.
func (r *fakeClock) Now() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.now
}

// Avanzar mueve el reloj simulado.
func (r *fakeClock) Avanzar(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.now = r.now.Add(d)
}

// contador cuenta las veces que el manejador de prueba fue invocado.
type contador struct {
	mu       sync.Mutex
	llamadas int
	reads    int
}

// Handler responde con un JSON que incluye el número de llamada.
func (c *contador) Handler(state int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.mu.Lock()
		c.llamadas++

		if r.Method == http.MethodGet {
			c.reads++
		}

		number := c.llamadas
		c.mu.Unlock()

		w.Header().Set(headerKind, kindJSON)
		w.WriteHeader(state)

		if _, err := w.Write([]byte(`{"llamada":` + strconv.Itoa(number) + `}`)); err != nil {
			return
		}
	})
}

// Reads devuelve cuántas lecturas llegaron al manejador.
func (c *contador) Reads() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.reads
}

// Total devuelve cuántas veces se invocó el manejador.
func (c *contador) Total() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.llamadas
}

// newCache arma una caché con reloj falso y una política para /api.
func newCache(ttl time.Duration) (*Cache, *fakeClock) {
	clock := &fakeClock{now: time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)}
	cache := New([]Policy{{Prefijo: "/api", TTL: ttl}})
	cache.clock = clock.Now

	return cache, clock
}

// TestCacheServesSecondReadWithoutLlamarToHandler comprueba el acierto.
func TestCacheServesSecondReadWithoutLlamarToHandler(t *testing.T) {
	t.Parallel()

	cache, _ := newCache(10 * time.Second)
	contador := &contador{}
	handler := cache.Wrap(contador.Handler(http.StatusOK))

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathSession, http.NoBody))

		if rec.Code != http.StatusOK {
			t.Fatalf("código = %d, want %d", rec.Code, http.StatusOK)
		}

		if got := rec.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
	}

	if contador.Total() != 1 {
		t.Errorf(msgTime, contador.Total())
	}
}

// TestCacheDistingueStringOfConsulta comprueba que cada búsqueda es
// una entrada distinta.
func TestCacheDistingueStringOfConsulta(t *testing.T) {
	t.Parallel()

	cache, _ := newCache(10 * time.Second)
	contador := &contador{}
	handler := cache.Wrap(contador.Handler(http.StatusOK))

	for _, path := range []string{"/api/vms?q=deb", "/api/vms?q=ubuntu", "/api/vms?q=deb"} {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, http.NoBody))
	}

	if contador.Total() != 2 {
		t.Errorf(msgDosTimes, contador.Total())
	}
}

// TestCacheExpires comprueba que una entrada vieja se vuelve a pedir.
func TestCacheExpires(t *testing.T) {
	t.Parallel()

	cache, clock := newCache(5 * time.Second)
	contador := &contador{}
	handler := cache.Wrap(contador.Handler(http.StatusOK))
	request := func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathPackages, http.NoBody))
	}

	request()
	request()

	if contador.Total() != 1 {
		t.Fatalf(msgTime, contador.Total())
	}

	clock.Avanzar(6 * time.Second)
	request()

	if contador.Total() != 2 {
		t.Errorf("llamadas = %d, want 2 tras expirar", contador.Total())
	}
}

// TestCacheSeInvalidaToWrite comprueba que una mutación vacía la caché.
func TestCacheSeInvalidaToWrite(t *testing.T) {
	t.Parallel()

	cache, _ := newCache(time.Minute)
	contador := &contador{}
	handler := cache.Wrap(contador.Handler(http.StatusOK))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathPackages, http.NoBody))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathPackages, http.NoBody))

	if contador.Total() != 1 {
		t.Fatalf(msgTime, contador.Total())
	}

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodPost, pathPackages, http.NoBody))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathPackages, http.NoBody))

	if contador.Total() != 3 {
		t.Errorf("llamadas = %d, want 3 (lectura y escritura)", contador.Total())
	}
}

// TestCacheNotSavesErrors evita recordar un fallo transitorio.
func TestCacheNotSavesErrors(t *testing.T) {
	t.Parallel()

	cache, _ := newCache(time.Minute)
	contador := &contador{}
	handler := cache.Wrap(contador.Handler(http.StatusServiceUnavailable))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathSession, http.NoBody))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathSession, http.NoBody))

	if contador.Total() != 2 {
		t.Errorf(msgDosTimes, contador.Total())
	}
}

// TestCacheIgnoresThatNotIsOfAPI deja intactos los archivos estáticos.
func TestCacheIgnoresThatNotIsOfAPI(t *testing.T) {
	t.Parallel()

	cache, _ := newCache(time.Minute)
	contador := &contador{}
	handler := cache.Wrap(contador.Handler(http.StatusOK))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/assets/main.js", http.NoBody))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/assets/main.js", http.NoBody))

	if contador.Total() != 2 {
		t.Errorf(msgDosTimes, contador.Total())
	}
}

// TestCacheRespetaNotCache permite forzar una lectura fresca.
func TestCacheRespetaNotCache(t *testing.T) {
	t.Parallel()

	cache, _ := newCache(time.Minute)
	contador := &contador{}
	handler := cache.Wrap(contador.Handler(http.StatusOK))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathVMs, http.NoBody))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathVMs, http.NoBody)
	req.Header.Set("Cache-Control", "no-cache")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if contador.Total() != 2 {
		t.Errorf(msgDosTimes, contador.Total())
	}
}

// TestCacheAnunciaAcierto comprueba el encabezado de diagnóstico.
func TestCacheAnunciaAcierto(t *testing.T) {
	t.Parallel()

	cache, _ := newCache(time.Minute)
	handler := cache.Wrap((&contador{}).Handler(http.StatusOK))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathVMs, http.NoBody))

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathVMs, http.NoBody))

	if got := first.Header().Get(headerCache); got != "MISS" {
		t.Errorf("%s = %q, want MISS", headerCache, got)
	}

	if got := second.Header().Get(headerCache); got != "HIT" {
		t.Errorf("%s = %q, want HIT", headerCache, got)
	}
}

// TestChangeOfMachineInvalidaReads reproduce el caso real: al
// vincular otra máquina se descarta todo lo que se había leído de la
// anterior.
func TestChangeOfMachineInvalidaReads(t *testing.T) {
	t.Parallel()

	cache, _ := newCache(time.Minute)
	contador := &contador{}
	handler := cache.Wrap(contador.Handler(http.StatusOK))

	read := func() {
		handler.ServeHTTP(httptest.NewRecorder(),
			httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathVMs, http.NoBody))
	}

	read()
	read()

	if contador.Reads() != 1 {
		t.Fatalf(msgTime, contador.Reads())
	}

	handler.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequestWithContext(context.Background(), http.MethodPut, pathSession, http.NoBody))

	read()

	if contador.Reads() != 2 {
		t.Errorf("lecturas = %d, want 2 tras cambiar de máquina", contador.Reads())
	}
}

// TestResponseInVueloNotContaminaCache reproduce la carrera que deja
// datos de la máquina anterior: una lectura lenta que empezó antes de
// cambiar de máquina no debe guardarse al terminar.
func TestResponseInVueloNotContaminaCache(t *testing.T) {
	t.Parallel()

	cache, _ := newCache(time.Minute)

	var (
		mu       sync.Mutex
		llamadas int
	)

	inVuelo := make(chan struct{})
	liberar := make(chan struct{})

	handler := cache.Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		llamadas++
		number := llamadas
		mu.Unlock()

		if number == 1 {
			close(inVuelo)
			<-liberar
		}

		w.Header().Set(headerKind, kindJSON)

		if _, err := w.Write([]byte(`{"llamada":` + strconv.Itoa(number) + `}`)); err != nil {
			return
		}
	}))

	var espera sync.WaitGroup

	espera.Add(1)

	go func() {
		defer espera.Done()

		handler.ServeHTTP(httptest.NewRecorder(),
			httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathPackages, http.NoBody))
	}()

	<-inVuelo

	// Cambiar de máquina mientras la lectura anterior sigue en curso.
	handler.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequestWithContext(context.Background(), http.MethodPut, pathSession, http.NoBody))

	close(liberar)
	espera.Wait()

	// La siguiente lectura debe pedir los datos a la máquina nueva.
	handler.ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, pathPackages, http.NoBody))

	mu.Lock()
	total := llamadas
	mu.Unlock()

	if total != 3 {
		t.Errorf("llamadas = %d, want 3 (lectura, cambio y lectura nueva)", total)
	}
}

// failingWriter2 simula un cliente que corta la respuesta.
type failingWriter2 struct {
	header http.Header
	state  int
}

// Header devuelve las cabeceras de la respuesta.
func (e *failingWriter2) Header() http.Header {
	if e.header == nil {
		e.header = make(http.Header)
	}

	return e.header
}

// WriteHeader registra el estado y descarta la escritura.
func (e *failingWriter2) WriteHeader(code int) {
	if e.state == 0 {
		e.state = code
	}
}

// Write siempre falla, como una conexión cerrada.
func (e *failingWriter2) Write([]byte) (int, error) {
	return 0, errors.New("conexión cerrada")
}

// TestCacheSaveWithoutPolicy comprueba que un recurso sin vigencia no se
// guarda.
func TestCacheSaveWithoutPolicy(t *testing.T) {
	t.Parallel()

	cache := New(nil)
	cache.save("/otra/ruta", entry{body: []byte("x")}, cache.currentGeneration())

	if _, found := cache.find("/otra/ruta"); found {
		t.Error("buscar() = true, want false para un recurso sin política")
	}
}

// TestCacheSaveIgnoresBodyLarge descarta respuestas desmedidas.
func TestCacheSaveIgnoresBodyLarge(t *testing.T) {
	t.Parallel()

	cache, _ := newCache(time.Minute)
	cache.save(pathPackages, entry{body: make([]byte, maxBody+1)}, cache.currentGeneration())

	if _, found := cache.find(pathPackages); found {
		t.Error("buscar() = true, want false para un cuerpo demasiado grande")
	}
}

// TestCacheSaveDiscardsGenerationOld evita que una lectura en vuelo
// repueble la caché después de una invalidación.
func TestCacheSaveDiscardsGenerationOld(t *testing.T) {
	t.Parallel()

	cache, _ := newCache(time.Minute)
	generation := cache.currentGeneration()

	cache.clear()
	cache.save(pathPackages, entry{body: []byte("{")}, generation)

	if _, found := cache.find(pathPackages); found {
		t.Error("buscar() = true, want false para una generación anterior")
	}
}

// TestCacheReciclaEntries vacía la caché cuando alcanza el límite.
func TestCacheReciclaEntries(t *testing.T) {
	t.Parallel()

	cache, _ := newCache(time.Minute)
	cache.generation++
	generation := cache.currentGeneration()

	for i := 0; i < maxEntries; i++ {
		cache.entries[pathPackages+strconv.Itoa(i)] = entry{body: []byte("x")}
	}

	cache.save(pathPackages, entry{body: []byte("y")}, generation)

	if len(cache.entries) != 1 {
		t.Errorf("entradas = %d, want 1 tras reciclar", len(cache.entries))
	}
}

// TestValidityOfPrefierePrefijoMasEspecifico resuelve la política más
// concreta.
func TestValidityOfPrefierePrefijoMasEspecifico(t *testing.T) {
	t.Parallel()

	cache := New([]Policy{
		{Prefijo: "/api", TTL: time.Minute},
		{Prefijo: pathSession, TTL: 3 * time.Second},
	})

	if ttl := cache.validityOf(pathSession + "/extra"); ttl != 3*time.Second {
		t.Errorf("vigenciaDe() = %v, want 3s", ttl)
	}

	if ttl := cache.validityOf("/otra"); ttl != 0 {
		t.Errorf("vigenciaDe() = %v, want 0", ttl)
	}
}

// TestWriteEntryWithClienteCaido comprueba que el acierto conserva
// sus cabeceras aunque el cliente corte la conexión.
func TestWriteEntryWithClienteCaido(t *testing.T) {
	t.Parallel()

	writer := &failingWriter2{}
	entry := entry{
		body:   []byte("x"),
		header: http.Header{headerKind: []string{kindJSON}},
		state:  http.StatusOK,
	}

	writeEntry(writer, entry, "HIT")

	if writer.state != http.StatusOK {
		t.Errorf("estado = %d, want 200", writer.state)
	}

	if header := writer.Header().Get(headerCache); header != "HIT" {
		t.Errorf("%s = %q, want HIT", headerCache, header)
	}

	if kind := writer.Header().Get(headerKind); kind != kindJSON {
		t.Errorf("%s = %q, want %q", headerKind, kind, kindJSON)
	}
}

// TestRecorderDumpWithClienteCaido comprueba que el fallo de escritura no
// pierde el estado ni las cabeceras de la respuesta.
func TestRecorderDumpWithClienteCaido(t *testing.T) {
	t.Parallel()

	recorder := newRecorder()
	recorder.Header().Set(headerKind, kindJSON)

	if _, err := recorder.Write([]byte("x")); err != nil {
		t.Fatalf("grabador.Write() error = %v", err)
	}

	writer := &failingWriter2{}
	recorder.dump(writer, "MISS")

	if writer.state != http.StatusOK {
		t.Errorf("estado = %d, want 200", writer.state)
	}

	if header := writer.Header().Get(headerCache); header != "MISS" {
		t.Errorf("%s = %q, want MISS", headerCache, header)
	}

	if kind := writer.Header().Get(headerKind); kind != kindJSON {
		t.Errorf("%s = %q, want %q", headerKind, kind, kindJSON)
	}
}
