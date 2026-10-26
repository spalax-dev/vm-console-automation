// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

// Package handlers expone la API HTTP de la consola: la sesión y el
// registro de la máquina, los módulos de preparación del sistema,
// usuarios, systemd y HAProxy, el historial, la autenticación y los
// middlewares de observabilidad. Los handlers validan y traducen a JSON;
// los casos de uso viven en internal/services.
package handlers
