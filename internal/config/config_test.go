// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestAddressFromEnv comprueba la dirección de escucha configurada.
func TestAddressFromEnv(t *testing.T) {
	t.Setenv(EnvAddress, "127.0.0.1:9090")

	if got := Address(); got != "127.0.0.1:9090" {
		t.Errorf("Address() = %q, want 127.0.0.1:9090", got)
	}

	t.Setenv(EnvAddress, "")

	if got := Address(); got != AddressProd {
		t.Errorf("Address() = %q, want %q", got, AddressProd)
	}
}

// TestDefaultPaths comprueba la base, la clave y la llave.
func TestDefaultPaths(t *testing.T) {
	t.Setenv(EnvDB, "")
	t.Setenv(EnvSSHKey, "")

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")

	want := filepath.Join(home, ".local", "share", "vmcons", "vmcons.db")
	if got := DatabasePath(); got != want {
		t.Errorf("DatabasePath() = %q, want %q", got, want)
	}

	if got := CipherKeyPath(); got != filepath.Join(filepath.Dir(want), "credenciales.key") {
		t.Errorf("CipherKeyPath() = %q, want junto a la base", got)
	}

	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	want = filepath.Join(dataHome, "vmcons", "vmcons.db")
	if got := DatabasePath(); got != want {
		t.Errorf("DatabasePath() con XDG_DATA_HOME = %q, want %q", got, want)
	}

	t.Setenv("XDG_DATA_HOME", "ruta/relativa")

	if got := DatabasePath(); got != filepath.Join(home, ".local", "share", "vmcons", "vmcons.db") {
		t.Errorf("DatabasePath() con XDG_DATA_HOME relativo = %q, want la ruta del home", got)
	}

	base := filepath.Join(t.TempDir(), "datos", "vmcons.db")
	t.Setenv(EnvDB, base)

	if got := CipherKeyPath(); got != filepath.Join(filepath.Dir(base), "credenciales.key") {
		t.Errorf("CipherKeyPath() = %q, want junto a la base", got)
	}

	t.Setenv(EnvSSHKey, "/tmp/llave")

	if got := SSHKeyPath(); got != "/tmp/llave" {
		t.Errorf("SSHKeyPath() = %q, want /tmp/llave", got)
	}

	t.Setenv(EnvSSHKey, "")

	if got := SSHKeyPath(); !strings.HasSuffix(got, filepath.Join("vmcons", "id_ed25519")) {
		t.Errorf("SSHKeyPath() = %q, want la llave en el directorio de configuración", got)
	}
}

// TestGuestUserByDefault devuelve root.
func TestGuestUserByDefault(t *testing.T) {
	t.Setenv(EnvGuestUser, "")

	if got := GuestUser(); got != "root" {
		t.Errorf("GuestUser() = %q, want root", got)
	}

	t.Setenv(EnvGuestUser, "jorge")

	if got := GuestUser(); got != "jorge" {
		t.Errorf("GuestUser() = %q, want jorge", got)
	}
}
