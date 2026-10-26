// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"

	"consola-vm/internal/core/domain"
)

// stubRegistroDiag informa si la máquina está registrada.
type stubRegistroDiag struct{ registrada bool }

// Registrada devuelve el estado configurado.
func (s *stubRegistroDiag) Registrada(string) bool { return s.registrada }

// nuevoDiagnostico arma el servicio con la máquina y el objetivo dados.
func nuevoDiagnostico(maquina *stubMachine, registrada bool, objetivo string) *DiagnosticoServicio {
	return NuevoDiagnostico(&ConfigDiagnostico{
		Maquina:  maquina,
		Registro: &stubRegistroDiag{registrada: registrada},
		Objetivo: func() domain.Target { return domain.Target{ID: objetivo} },
	})
}

// TestDiagnosticoConEstadoIlegible propaga el fallo de VirtualBox.
func TestDiagnosticoConEstadoIlegible(t *testing.T) {
	t.Parallel()

	maquina := &stubMachine{stateErr: errors.New("sin VBoxManage")}

	if _, err := nuevoDiagnostico(maquina, false, "otra-maquina").Obtener(context.Background()); err == nil {
		t.Error("Obtener() error = nil, want el fallo del estado")
	}
}

// TestDiagnosticoDeMaquinaApagada explica que está apagada.
func TestDiagnosticoDeMaquinaApagada(t *testing.T) {
	t.Parallel()

	maquina := &stubMachine{state: domain.StatePoweredOff}

	informe, err := nuevoDiagnostico(maquina, false, nombreDebian).Obtener(context.Background())
	if err != nil {
		t.Fatalf("Obtener() error = %v, want nil", err)
	}

	if len(informe.Problemas) == 0 || informe.Problemas[0] != "La máquina está apagada." {
		t.Errorf("problemas = %v, want el aviso de máquina apagada", informe.Problemas)
	}

	if informe.SSHAbierto {
		t.Error("SSHAbierto = true para una máquina apagada")
	}
}

// TestDiagnosticoDeMaquinaRegistrada avisa que no hace falta registrarla.
func TestDiagnosticoDeMaquinaRegistrada(t *testing.T) {
	t.Parallel()

	maquina := &stubMachine{state: domain.StateRunning, guest: domain.GuestAdditions{
		Version: "7.2.18", RunLevel: 2,
	}}

	informe, err := nuevoDiagnostico(maquina, true, nombreDebian).Obtener(context.Background())
	if err != nil {
		t.Fatalf("Obtener() error = %v, want nil", err)
	}

	if !informe.Registrada {
		t.Error("Registrada = false, want true")
	}
}

// TestDiagnosticoSinGuestAdditions avisa de la versión ausente.
func TestDiagnosticoSinGuestAdditions(t *testing.T) {
	t.Parallel()

	maquina := &stubMachine{state: domain.StateRunning}

	informe, err := nuevoDiagnostico(maquina, false, nombreDebian).Obtener(context.Background())
	if err != nil {
		t.Fatalf("Obtener() error = %v, want nil", err)
	}

	if len(informe.Problemas) == 0 {
		t.Error("problemas vacío, want el aviso de Guest Additions")
	}
}

// TestDiagnosticoConNivelBajo avisa de un nivel incompleto.
func TestDiagnosticoConNivelBajo(t *testing.T) {
	t.Parallel()

	maquina := &stubMachine{state: domain.StateRunning, guest: domain.GuestAdditions{
		Version: "7.2.18", RunLevel: 1,
	}}

	informe, err := nuevoDiagnostico(maquina, false, nombreDebian).Obtener(context.Background())
	if err != nil {
		t.Fatalf("Obtener() error = %v, want nil", err)
	}

	if len(informe.Problemas) == 0 {
		t.Error("problemas vacío, want el aviso del nivel de Guest Additions")
	}
}

// TestDiagnosticoConVersionesDistintas compara con el anfitrión.
func TestDiagnosticoConVersionesDistintas(t *testing.T) {
	t.Parallel()

	maquina := &stubMachine{state: domain.StateRunning, guest: domain.GuestAdditions{
		Version: "6.0.0", RunLevel: 2,
	}}

	servicio := NuevoDiagnostico(&ConfigDiagnostico{
		Maquina:        maquina,
		Objetivo:       func() domain.Target { return domain.Target{ID: nombreDebian} },
		VersionSistema: func(context.Context) (string, error) { return "7.2.18", nil },
	})

	informe, err := servicio.Obtener(context.Background())
	if err != nil {
		t.Fatalf("Obtener() error = %v, want nil", err)
	}

	if len(informe.Problemas) == 0 {
		t.Error("problemas vacío, want el aviso de versión distinta")
	}
}

// TestDiagnosticoConVersionIlegible no falla si el anfitrión no responde.
func TestDiagnosticoConVersionIlegible(t *testing.T) {
	t.Parallel()

	maquina := &stubMachine{state: domain.StateRunning, guest: domain.GuestAdditions{
		Version: "7.2.18", RunLevel: 2,
	}}

	servicio := NuevoDiagnostico(&ConfigDiagnostico{
		Maquina:        maquina,
		Objetivo:       func() domain.Target { return domain.Target{ID: nombreDebian} },
		VersionSistema: func(context.Context) (string, error) { return "", errors.New("sin VBoxManage") },
	})

	if _, err := servicio.Obtener(context.Background()); err != nil {
		t.Errorf("Obtener() error = %v, want nil", err)
	}
}

// conexionFalsa simula una conexión TCP ya establecida.
type conexionFalsa struct {
	closeErr error
}

// Read no se usa.
func (conexionFalsa) Read([]byte) (int, error) { return 0, nil }

// Write no se usa.
func (conexionFalsa) Write(datos []byte) (int, error) { return len(datos), nil }

// Close devuelve el error configurado.
func (c conexionFalsa) Close() error { return c.closeErr }

// LocalAddr no se usa.
func (conexionFalsa) LocalAddr() net.Addr { return &net.TCPAddr{} }

// RemoteAddr no se usa.
func (conexionFalsa) RemoteAddr() net.Addr { return &net.TCPAddr{} }

// SetDeadline no se usa.
func (conexionFalsa) SetDeadline(time.Time) error { return nil }

// SetReadDeadline no se usa.
func (conexionFalsa) SetReadDeadline(time.Time) error { return nil }

// SetWriteDeadline no se usa.
func (conexionFalsa) SetWriteDeadline(time.Time) error { return nil }

// TestProbarPuertoSSHConExito cubre la conexión correcta.
func TestProbarPuertoSSHConExito(t *testing.T) {
	original := abrirConexion

	t.Cleanup(func() { abrirConexion = original })

	abrirConexion = func(context.Context, string) (net.Conn, error) { return conexionFalsa{}, nil }

	if !probarPuertoSSH(context.Background(), netip.MustParseAddr("127.0.0.1")) {
		t.Error("probarPuertoSSH() = false, want true")
	}

	abrirConexion = func(context.Context, string) (net.Conn, error) {
		return conexionFalsa{closeErr: errors.New("ya cerrada")}, nil
	}

	if !probarPuertoSSH(context.Background(), netip.MustParseAddr("127.0.0.1")) {
		t.Error("probarPuertoSSH() = false, want true aunque falle el cierre")
	}

	abrirConexion = func(context.Context, string) (net.Conn, error) { return nil, errors.New("sin ruta") }

	if probarPuertoSSH(context.Background(), netip.MustParseAddr("127.0.0.1")) {
		t.Error("probarPuertoSSH() = true, want false sin conexión")
	}
}
