// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package services implementa los casos de uso de la consola: descubre
// el conector disponible con la máquina virtual y ejecuta las
// operaciones de administración sobre ella. Depende solo de los puertos
// del núcleo (internal/core/ports) y no conoce los adaptadores
// concretos: la inyección de dependencias se hace en main.go.
package services
