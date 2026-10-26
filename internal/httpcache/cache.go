// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package httpcache guarda por unos segundos las respuestas de lectura de
// la API para no repetir consultas costosas a la máquina virtual: cada
// listado de paquetes, usuarios o unidades abre una sesión y ejecuta
// órdenes dentro de la VM.
//
// La caché es conservadora a propósito. Solo recuerda respuestas
// correctas de GET sobre la API, con una vigencia corta por recurso, y
// cualquier operación de escritura vacía la caché completa: una
// instalación de paquetes o un cambio de hostname invalida todo lo que
// el usuario podría estar viendo. Así se evita el riesgo clásico de
// mostrar datos viejos después de una operación.
package httpcache

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// headerCache anuncia si la respuesta salió de la caché. Es un
// encabezado de diagnóstico: no forma parte del contrato de la API.
const headerCache = "X-Cache"

// maxEntries acota la memoria de la caché. Cuando se supera, se
// descarta todo: las entradas son de segundos y se reconstruyen solas.
const maxEntries = 256

// maxBody evita guardar respuestas desproporcionadas.
const maxBody = 1 << 20

// Policy declara la vigencia de las respuestas de un prefijo de ruta.
type Policy struct {
	// Prefijo es la ruta a la que se aplica, por ejemplo "/api/vms".
	Prefijo string
	// TTL es el tiempo durante el cual la respuesta se reutiliza.
	TTL time.Duration
}

// entry es una respuesta guardada.
type entry struct {
	body    []byte
	header  http.Header
	state   int
	expires time.Time
}

// Cache guarda respuestas de lectura de la API durante una vigencia
// corta y las invalida ante cualquier escritura.
type Cache struct {
	policies []Policy

	mu      sync.Mutex
	entries map[string]entry

	// generation cuenta las invalidaciones. Una respuesta que empezó a
	// calcularse antes de una invalidación no se guarda al terminar: sus
	// datos son de la máquina o del estado anterior.
	generation uint64

	// clock permite inyectar el tiempo en las pruebas.
	clock func() time.Time
}

// New construye la caché con las políticas dadas. Un prefijo más
// específico gana sobre uno más general.
func New(policies []Policy) *Cache {
	return &Cache{
		policies: policies,
		entries:  make(map[string]entry),
		clock:    time.Now,
	}
}

// Wrap devuelve un manejador que cachea las lecturas de la API y
// delega el resto en siguiente.
func (c *Cache) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)

			if r.Method != http.MethodGet {
				c.clear()
			}

			return
		}

		key := r.URL.RequestURI()

		// El navegador no debe cachear la API: la caché es del servidor,
		// que sabe cuándo invalidarla.
		w.Header().Set("Cache-Control", "no-store")

		generation := c.currentGeneration()

		if r.Header.Get("Cache-Control") != "no-cache" {
			if saved, found := c.find(key); found {
				writeEntry(w, saved, "HIT")

				return
			}
		}

		recorder := newRecorder()
		next.ServeHTTP(recorder, r)

		if recorder.state == http.StatusOK {
			c.save(key, recorder.entry(), generation)
		}

		recorder.dump(w, "MISS")
	})
}

// clear descarta todas las entradas y avanza la generación. Se llama
// tras una escritura: lo que estuviera en vuelo ya no se guarda.
func (c *Cache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]entry)
	c.generation++
}

// currentGeneration devuelve la marca de la última invalidación.
func (c *Cache) currentGeneration() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.generation
}

// find devuelve la entrada vigente para la clave.
func (c *Cache) find(key string) (entry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	saved, found := c.entries[key]
	if !found {
		return entry{}, false
	}

	if !c.clock().Before(saved.expires) {
		delete(c.entries, key)

		return entry{}, false
	}

	return saved, true
}

// save almacena la respuesta si el recurso tiene una política activa
// y no hubo una invalidación desde que empezó la petición.
func (c *Cache) save(key string, fresh entry, generation uint64) {
	ttl := c.validityOf(key)
	if ttl <= 0 {
		return
	}

	if len(fresh.body) > maxBody {
		return
	}

	fresh.expires = c.clock().Add(ttl)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.generation != generation {
		return
	}

	if len(c.entries) >= maxEntries {
		c.entries = make(map[string]entry)
	}

	c.entries[key] = fresh
}

// validityOf resuelve la política del prefijo más específico que
// contiene la clave.
func (c *Cache) validityOf(key string) time.Duration {
	chosen := time.Duration(0)
	length := -1

	for _, policy := range c.policies {
		if !strings.HasPrefix(key, policy.Prefijo) {
			continue
		}

		if len(policy.Prefijo) > length {
			chosen = policy.TTL
			length = len(policy.Prefijo)
		}
	}

	return chosen
}

// writeEntry vuelca una respuesta guardada.
func writeEntry(w http.ResponseWriter, saved entry, cache string) {
	copyHeader(w, saved.header)
	w.Header().Set(headerCache, cache)
	w.WriteHeader(saved.state)

	if _, err := w.Write(saved.body); err != nil {
		return
	}
}

// copyHeader traslada las cabeceras guardadas al destino.
func copyHeader(w http.ResponseWriter, header http.Header) {
	for name, values := range header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
}

// recorder registra la respuesta del manejador siguiente para poder
// guardarla o reenviarla.
type recorder struct {
	header http.Header
	body   []byte
	state  int
}

// newRecorder construye un grabador vacío.
func newRecorder() *recorder {
	return &recorder{header: make(http.Header), state: http.StatusOK}
}

// Header expone las cabeceras que se están construyendo.
func (g *recorder) Header() http.Header { return g.header }

// WriteHeader registra el código de estado.
func (g *recorder) WriteHeader(state int) { g.state = state }

// Write acumula el cuerpo.
func (g *recorder) Write(data []byte) (int, error) {
	g.body = append(g.body, data...)

	return len(data), nil
}

// entry construye la entrada que se guardará.
func (g *recorder) entry() entry {
	return entry{body: g.body, header: g.header.Clone(), state: g.state}
}

// dump escribe la respuesta grabada en el destino real.
func (g *recorder) dump(w http.ResponseWriter, cache string) {
	copyHeader(w, g.header)
	w.Header().Set(headerCache, cache)
	w.WriteHeader(g.state)

	if _, err := w.Write(g.body); err != nil {
		return
	}
}
