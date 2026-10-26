// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"consola-vm/internal/core/domain"
)

// versionHostTests es la versión simulada de VirtualBox.
const versionHostTests = "7.2.18"

// newDiagnosticsOfTests arma el servicio con la máquina y el sondeo
// de SSH preprogramados.
func newDiagnosticsOfTests(
	t *testing.T, machine *stubMachine, sshOpen bool, version string,
) *DiagnosticsService {
	t.Helper()

	return NewDiagnostics(&DiagnosticsConfig{
		Machine:       machine,
		Registration:  &stubRegistration{registradas: map[string]bool{}},
		CurrentTarget: func() domain.Target { return domain.Target{ID: testMachineName} },
		VersionSystem: func(context.Context) (string, error) {
			return version, nil
		},
		TestSSH: func(context.Context, netip.Addr) bool {
			return sshOpen
		},
	})
}

// TestDiagnosticsIndicatesSSHYCredentials comprueba los problemas y las
// sugerencias de una VM accesible por red pero sin SSH.
func TestDiagnosticsIndicatesSSHYCredentials(t *testing.T) {
	t.Parallel()

	machine := &stubMachine{
		addr:    netip.MustParseAddr(ipTests),
		state:   domain.StateRunning,
		network: domain.NetworkBridged,
		guest: domain.GuestAdditions{
			Version:     "6.0.0 r127566",
			RunLevel:    2,
			LoggedUsers: []string{nameJorge},
		},
	}

	report, err := newDiagnosticsOfTests(t, machine, false, versionHostTests).Get(context.Background())
	if err != nil {
		t.Fatalf("Obtener() error = %v", err)
	}

	if report.VM != testMachineName || report.IP.String() != ipTests {
		t.Errorf("informe = %+v, want %s en %s", report, testMachineName, ipTests)
	}

	if report.SSHOpen {
		t.Error("SSHAbierto = true, want false")
	}

	if report.UserSugerido != nameJorge {
		t.Errorf("UsuarioSugerido = %q, want %q", report.UserSugerido, nameJorge)
	}

	if len(report.Problemas) < 2 {
		t.Errorf("Problemas = %v, want SSH y versión", report.Problemas)
	}

	if len(report.Sugerencias) == 0 {
		t.Error("Sugerencias vacías, want al menos una")
	}
}

// TestDiagnosticsIndicatesNAT comprueba el aviso de red NAT.
func TestDiagnosticsIndicatesNAT(t *testing.T) {
	t.Parallel()

	machine := &stubMachine{
		addr:    netip.MustParseAddr(ipTests),
		state:   domain.StateRunning,
		network: domain.NetworkNAT,
		guest:   domain.GuestAdditions{Version: versionHostTests, RunLevel: 2},
	}

	report, err := newDiagnosticsOfTests(t, machine, true, versionHostTests).Get(context.Background())
	if err != nil {
		t.Fatalf("Obtener() error = %v", err)
	}

	found := false

	for _, problema := range report.Problemas {
		if problema == "La interfaz está en modo nat; el anfitrión no alcanza esa red." {
			found = true
		}
	}

	if !found {
		t.Errorf("Problemas = %v, want el aviso de NAT", report.Problemas)
	}
}

// TestDiagnosticsWithoutMachine comprueba la consola sin vínculo.
func TestDiagnosticsWithoutMachine(t *testing.T) {
	t.Parallel()

	service := NewDiagnostics(&DiagnosticsConfig{
		Machine:       &stubMachine{},
		CurrentTarget: func() domain.Target { return domain.Target{} },
	})

	if _, err := service.Get(context.Background()); !errors.Is(err, ErrNoMachine) {
		t.Fatalf("Obtener() error = %v, want ErrSinMaquina", err)
	}
}
