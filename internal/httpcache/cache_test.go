// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package httpcache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

// Rutas y mensajes repetidos en las pruebas.
const (
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
}

// Manejador responde con un JSON que incluye el número de llamada.
func (c *contador) Manejador(estado int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		c.mu.Lock()
		c.llamadas++
		numero := c.llamadas
		c.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(estado)

		if _, err := w.Write([]byte(`{"llamada":` + strconv.Itoa(numero) + `}`)); err != nil {
			return
		}
	})
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
