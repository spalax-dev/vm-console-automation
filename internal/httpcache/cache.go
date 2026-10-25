// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package httpcache guarda por unos segundos las respuestas de lectura de
// la API para no repetir consultas costosas a la máquina virtual: cada
// listado de paquetes, usuarios o unidades abre una sesión y ejecuta
// órdenes dentro de la VM.
//
// La caché es conservadora a propósito. Solo recuerda respuestas
// correctas de `GET` sobre la API, con una vigencia corta por recurso, y
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

// cabeceraCache anuncia si la respuesta salió de la caché. Es un
// encabezado de diagnóstico: no forma parte del contrato de la API.
const cabeceraCache = "X-Cache"

// maximoEntradas acota la memoria de la caché. Cuando se supera, se
// descarta todo: las entradas son de segundos y se reconstruyen solas.
const maximoEntradas = 256

// maximoCuerpo evita guardar respuestas desproporcionadas.
const maximoCuerpo = 1 << 20

// Politica declara la vigencia de las respuestas de un prefijo de ruta.
type Politica struct {
	// Prefijo es la ruta a la que se aplica, por ejemplo "/api/vms".
	Prefijo string
	// TTL es el tiempo durante el cual la respuesta se reutiliza.
	TTL time.Duration
}

// entrada es una respuesta guardada.
type entrada struct {
	cuerpo   []byte
	cabecera http.Header
	estado   int
	expira   time.Time
}

// Cache guarda respuestas de lectura de la API durante una vigencia
// corta y las invalida ante cualquier escritura.
type Cache struct {
	politicas []Politica

	mu       sync.Mutex
	entradas map[string]entrada

	// reloj permite inyectar el tiempo en las pruebas.
	reloj func() time.Time
}

// Nuevo construye la caché con las políticas dadas. Un prefijo más
// específico gana sobre uno más general.
func Nuevo(politicas []Politica) *Cache {
	return &Cache{
		politicas: politicas,
		entradas:  make(map[string]entrada),
		reloj:     time.Now,
	}
}

// Envolver devuelve un manejador que cachea las lecturas de la API y
// delega el resto en siguiente.
func (c *Cache) Envolver(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/api/") {
			siguiente.ServeHTTP(w, r)

			if r.Method != http.MethodGet {
				c.vaciar()
			}

			return
		}

		clave := r.URL.RequestURI()

		if r.Header.Get("Cache-Control") != "no-cache" {
			if guardada, encontrada := c.buscar(clave); encontrada {
				escribirEntrada(w, guardada, "HIT")

				return
			}
		}

		grabador := nuevoGrabador()
		siguiente.ServeHTTP(grabador, r)

		if grabador.estado == http.StatusOK {
			c.guardar(clave, grabador.entrada())
		}

		grabador.volcar(w, "MISS")
	})
}

// vaciar descarta todas las entradas. Se llama tras una escritura.
func (c *Cache) vaciar() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entradas = make(map[string]entrada)
}

// buscar devuelve la entrada vigente para la clave.
func (c *Cache) buscar(clave string) (entrada, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	guardada, encontrada := c.entradas[clave]
	if !encontrada {
		return entrada{}, false
	}

	if !c.reloj().Before(guardada.expira) {
		delete(c.entradas, clave)

		return entrada{}, false
	}

	return guardada, true
}

// guardar almacena la respuesta si el recurso tiene una política activa.
func (c *Cache) guardar(clave string, nueva entrada) {
	ttl := c.vigenciaDe(clave)
	if ttl <= 0 {
		return
	}

	if len(nueva.cuerpo) > maximoCuerpo {
		return
	}

	nueva.expira = c.reloj().Add(ttl)

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entradas) >= maximoEntradas {
		c.entradas = make(map[string]entrada)
	}

	c.entradas[clave] = nueva
}

// vigenciaDe resuelve la política del prefijo más específico que
// contiene la clave.
func (c *Cache) vigenciaDe(clave string) time.Duration {
	elegida := time.Duration(0)
	longitud := -1

	for _, politica := range c.politicas {
		if !strings.HasPrefix(clave, politica.Prefijo) {
			continue
		}

		if len(politica.Prefijo) > longitud {
			elegida = politica.TTL
			longitud = len(politica.Prefijo)
		}
	}

	return elegida
}

// escribirEntrada vuelca una respuesta guardada.
func escribirEntrada(w http.ResponseWriter, guardada entrada, cache string) {
	copiarCabecera(w, guardada.cabecera)
	w.Header().Set(cabeceraCache, cache)
	w.WriteHeader(guardada.estado)

	if _, err := w.Write(guardada.cuerpo); err != nil {
		return
	}
}

// copiarCabecera traslada las cabeceras guardadas al destino.
func copiarCabecera(w http.ResponseWriter, cabecera http.Header) {
	for nombre, valores := range cabecera {
		for _, valor := range valores {
			w.Header().Add(nombre, valor)
		}
	}
}

// grabador registra la respuesta del manejador siguiente para poder
// guardarla o reenviarla.
type grabador struct {
	cabecera http.Header
	cuerpo   []byte
	estado   int
}

// nuevoGrabador construye un grabador vacío.
func nuevoGrabador() *grabador {
	return &grabador{cabecera: make(http.Header), estado: http.StatusOK}
}

// Header expone las cabeceras que se están construyendo.
func (g *grabador) Header() http.Header { return g.cabecera }

// WriteHeader registra el código de estado.
func (g *grabador) WriteHeader(estado int) { g.estado = estado }

// Write acumula el cuerpo.
func (g *grabador) Write(datos []byte) (int, error) {
	g.cuerpo = append(g.cuerpo, datos...)

	return len(datos), nil
}

// entrada construye la entrada que se guardará.
func (g *grabador) entrada() entrada {
	return entrada{cuerpo: g.cuerpo, cabecera: g.cabecera.Clone(), estado: g.estado}
}

// volcar escribe la respuesta grabada en el destino real.
func (g *grabador) volcar(w http.ResponseWriter, cache string) {
	copiarCabecera(w, g.cabecera)
	w.Header().Set(cabeceraCache, cache)
	w.WriteHeader(g.estado)

	if _, err := w.Write(g.cuerpo); err != nil {
		return
	}
}
