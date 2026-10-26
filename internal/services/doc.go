// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package services implementa los casos de uso de la consola: descubre el
// conector disponible, ejecuta las operaciones de administración sobre la
// máquina, guarda credenciales, historial y configuración en SQLite y
// autentica a los usuarios de la consola. Depende solo de los puertos del
// núcleo (internal/core/ports) y no conoce los adaptadores concretos: la
// inyección de dependencias se hace en main.go.
package services
