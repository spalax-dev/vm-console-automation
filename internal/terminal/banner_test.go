// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package terminal

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestURL comprueba la dirección que se anuncia al arrancar.
func TestURL(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		":8080":           "http://localhost:8080",
		"0.0.0.0:8080":    "http://localhost:8080",
		"127.0.0.1:9000":  "http://127.0.0.1:9000",
		"192.168.1.10:80": "http://192.168.1.10:80",
		"[::1]:8080":      "http://[::1]:8080",
		"localhost:3000":  "http://localhost:3000",
	}

	for address, expected := range testCases {
		if got := URL(address); got != expected {
			t.Errorf("URL(%q) = %q, want %q", address, got, expected)
		}
	}
}

// TestPrintBannerWritesURL captura la salida del banner.
func TestPrintBannerWritesURL(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	var output bytes.Buffer
	if err := PrintBanner(&output, "127.0.0.1:8080"); err != nil {
		t.Fatalf("PrintBanner() error = %v, want nil", err)
	}

	if !strings.Contains(output.String(), "http://127.0.0.1:8080") {
		t.Errorf("banner = %q, want la URL de la consola", output.String())
	}
}

// TestColorEnabledRespectsNoColor comprueba la variable de entorno.
func TestColorEnabledRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	if ColorEnabled(os.Stdout) {
		t.Error("ColorEnabled() = true con NO_COLOR definido")
	}
}

// TestColorEnabledRejectsNonFile devuelve false para destinos que no son
// archivos, porque no se puede saber si son una terminal.
func TestColorEnabledRejectsNonFile(t *testing.T) {
	t.Parallel()

	if ColorEnabled(&bytes.Buffer{}) {
		t.Error("ColorEnabled(buffer) = true, want false")
	}
}
