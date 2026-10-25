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

// versionAnfitrionPrueba es la versión simulada de VirtualBox.
const versionAnfitrionPrueba = "7.2.18"

// nuevoDiagnosticoDePrueba arma el servicio con la máquina y el sondeo
// de SSH preprogramados.
func nuevoDiagnosticoDePrueba(
	t *testing.T, maquina *stubMachine, sshAbierto bool, version string,
) *DiagnosticoServicio {
	t.Helper()

	return NuevoDiagnostico(&ConfigDiagnostico{
		Maquina:  maquina,
		Registro: &stubRegistro{registradas: map[string]bool{}},
		Objetivo: func() domain.Target { return domain.Target{ID: nombreDebian} },
		VersionSistema: func(context.Context) (string, error) {
			return version, nil
		},
		ProbarSSH: func(context.Context, netip.Addr) bool {
			return sshAbierto
		},
	})
}

// TestDiagnosticoSenalaSSHYCredenciales comprueba los problemas y las
// sugerencias de una VM accesible por red pero sin SSH.
func TestDiagnosticoSenalaSSHYCredenciales(t *testing.T) {
	t.Parallel()

	maquina := &stubMachine{
		addr:    netip.MustParseAddr(ipPrueba),
		state:   domain.StateRunning,
		network: domain.NetworkBridged,
		guest: domain.GuestAdditions{
			Version:     "6.0.0 r127566",
			RunLevel:    2,
			LoggedUsers: []string{nombreJorge},
		},
	}

	informe, err := nuevoDiagnosticoDePrueba(t, maquina, false, versionAnfitrionPrueba).Obtener(context.Background())
	if err != nil {
		t.Fatalf("Obtener() error = %v", err)
	}

	if informe.VM != nombreDebian || informe.IP.String() != ipPrueba {
		t.Errorf("informe = %+v, want %s en %s", informe, nombreDebian, ipPrueba)
	}

	if informe.SSHAbierto {
		t.Error("SSHAbierto = true, want false")
	}

	if informe.UsuarioSugerido != nombreJorge {
		t.Errorf("UsuarioSugerido = %q, want %q", informe.UsuarioSugerido, nombreJorge)
	}

	if len(informe.Problemas) < 2 {
		t.Errorf("Problemas = %v, want SSH y versión", informe.Problemas)
	}

	if len(informe.Sugerencias) == 0 {
		t.Error("Sugerencias vacías, want al menos una")
	}
}

// TestDiagnosticoSenalaNAT comprueba el aviso de red NAT.
func TestDiagnosticoSenalaNAT(t *testing.T) {
	t.Parallel()

	maquina := &stubMachine{
		addr:    netip.MustParseAddr(ipPrueba),
		state:   domain.StateRunning,
		network: domain.NetworkNAT,
		guest:   domain.GuestAdditions{Version: versionAnfitrionPrueba, RunLevel: 2},
	}

	informe, err := nuevoDiagnosticoDePrueba(t, maquina, true, versionAnfitrionPrueba).Obtener(context.Background())
	if err != nil {
		t.Fatalf("Obtener() error = %v", err)
	}

	encontrado := false

	for _, problema := range informe.Problemas {
		if problema == "La interfaz está en modo nat; el anfitrión no alcanza esa red." {
			encontrado = true
		}
	}

	if !encontrado {
		t.Errorf("Problemas = %v, want el aviso de NAT", informe.Problemas)
	}
}

// TestDiagnosticoSinMaquina comprueba la consola sin vínculo.
func TestDiagnosticoSinMaquina(t *testing.T) {
	t.Parallel()

	servicio := NuevoDiagnostico(&ConfigDiagnostico{
		Maquina:  &stubMachine{},
		Objetivo: func() domain.Target { return domain.Target{} },
	})

	if _, err := servicio.Obtener(context.Background()); !errors.Is(err, ErrSinMaquina) {
		t.Fatalf("Obtener() error = %v, want ErrSinMaquina", err)
	}
}
