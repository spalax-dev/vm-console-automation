// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/core/ports"
)

// nuevaLlaveDePrueba crea un par de llaves falso en el anfitrión.
func nuevaLlaveDePrueba(t *testing.T) string {
	t.Helper()

	ruta := filepath.Join(t.TempDir(), nombreLlave)
	if err := os.WriteFile(ruta, []byte(contenidoPrivado), 0o600); err != nil {
		t.Fatalf(msgCrearDir, err)
	}

	if err := os.WriteFile(ruta+".pub", []byte(clavePublica+"\n"), 0o600); err != nil {
		t.Fatalf(msgEscribirPub, err)
	}

	return ruta
}

// nuevoAccesoDePrueba arma el servicio con máquina, conectores y llave
// falsos.
func nuevoAccesoDePrueba(t *testing.T, estado domain.MachineState, sesion *stubSession, probeErr error) (*Acceso, *Almacen) {
	t.Helper()

	almacen, _, _ := nuevaBaseDePrueba(t)

	maquina := &stubMachine{addr: netip.MustParseAddr(ipPrueba), state: estado}
	guest := &stubConnector{name: conectorGuest, session: sesion}
	conectorSSH := &stubConnector{name: conectorSSH, probe: probeErr}

	acceso := NuevoAcceso(&ConfigAcceso{
		Almacen:   almacen,
		Maquina:   maquina,
		Guest:     guest,
		SSH:       conectorSSH,
		Ejecutor:  &stubExecutor{},
		RutaLlave: nuevaLlaveDePrueba(t),
		Objetivo:  func() domain.Target { return domain.Target{ID: nombreDebian} },
	})

	return acceso, almacen
}

// TestAccesoRegistrarMarcaLaMaquina comprueba el registro completo.
func TestAccesoRegistrarMarcaLaMaquina(t *testing.T) {
	t.Parallel()

	acceso, almacen := nuevoAccesoDePrueba(t, domain.StateRunning, &stubSession{}, nil)

	resultado, err := acceso.Registrar(context.Background(), usuarioRoot, passwordPrueba)
	if err != nil {
		t.Fatalf("Registrar() error = %v, want nil", err)
	}

	if resultado.Mensaje == "" {
		t.Error("Mensaje vacío, want confirmación")
	}

	credenciales, ok := almacen.Obtener(nombreDebian)
	if !ok || !credenciales.Registrada {
		t.Errorf("credenciales = %+v, want registrada", credenciales)
	}

	if !almacen.Registrada(nombreDebian) {
		t.Error("Registrada() = false, want true")
	}
}

// TestAccesoRegistrarCredencialesInvalidas restaura el estado si falla
// la verificación.
func TestAccesoRegistrarCredencialesInvalidas(t *testing.T) {
	t.Parallel()

	sesion := &stubSession{err: &domain.ExitError{Code: 1, Command: "id"}}
	acceso, almacen := nuevoAccesoDePrueba(t, domain.StateRunning, sesion, nil)

	if _, err := acceso.Registrar(context.Background(), usuarioRoot, passwordPrueba); !errors.Is(err, ErrCredencialesInvalidas) {
		t.Fatalf("Registrar() error = %v, want ErrCredencialesInvalidas", err)
	}

	if _, ok := almacen.Obtener(nombreDebian); ok {
		t.Error("se guardó una máquina sin registrar")
	}
}

// TestAccesoRegistrarSinSSH falla si la llave no queda operativa.
func TestAccesoRegistrarSinSSH(t *testing.T) {
	t.Parallel()

	acceso, almacen := nuevoAccesoDePrueba(t, domain.StateRunning, &stubSession{}, errors.New("ssh no responde"))

	if _, err := acceso.Registrar(context.Background(), usuarioRoot, passwordPrueba); err == nil {
		t.Fatal("Registrar() error = nil, want el fallo de verificación SSH")
	}

	if _, ok := almacen.Obtener(nombreDebian); ok {
		t.Error("se registró una máquina sin SSH")
	}
}

// TestAccesoRegistrarSinMaquina comprueba la consola sin vínculo.
func TestAccesoRegistrarSinMaquina(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	acceso := NuevoAcceso(&ConfigAcceso{
		Almacen:  almacen,
		Objetivo: func() domain.Target { return domain.Target{} },
	})

	if _, err := acceso.Registrar(context.Background(), usuarioRoot, passwordPrueba); !errors.Is(err, ErrSinMaquina) {
		t.Fatalf("Registrar() error = %v, want ErrSinMaquina", err)
	}
}

// TestAccesoRegistrarMaquinaApagada comprueba la VM apagada.
func TestAccesoRegistrarMaquinaApagada(t *testing.T) {
	t.Parallel()

	acceso, _ := nuevoAccesoDePrueba(t, domain.StatePoweredOff, &stubSession{}, nil)

	if _, err := acceso.Registrar(context.Background(), usuarioRoot, passwordPrueba); !errors.Is(err, ErrSesionNoDisponible) {
		t.Fatalf("Registrar() error = %v, want ErrSesionNoDisponible", err)
	}
}

// TestAccesoOlvidar elimina las credenciales guardadas.
func TestAccesoOlvidar(t *testing.T) {
	t.Parallel()

	acceso, almacen := nuevoAccesoDePrueba(t, domain.StateRunning, &stubSession{}, nil)

	if err := almacen.Guardar(nombreDebian, Credenciales{Usuario: usuarioRoot, Registrada: true}); err != nil {
		t.Fatalf("Guardar() error = %v", err)
	}

	if err := acceso.Olvidar(); err != nil {
		t.Fatalf("Olvidar() error = %v, want nil", err)
	}

	if _, ok := almacen.Obtener(nombreDebian); ok {
		t.Error("Obtener() encontró credenciales olvidadas")
	}
}

// stubPasswordConnector abre una sesión falsa con contraseña.
type stubPasswordConnector struct {
	sesion   ports.Session
	err      error
	usuario  string
	password string
	llamadas int
}

// ConnectPassword registra las credenciales y devuelve la sesión.
func (s *stubPasswordConnector) ConnectPassword(
	_ context.Context, _ domain.Target, usuario, password string,
) (ports.Session, error) {
	s.usuario = usuario
	s.password = password
	s.llamadas++

	return s.sesion, s.err
}

// TestAccesoRegistrarConSSHPassword comprueba el respaldo cuando Guest
// Additions rechaza las credenciales.
func TestAccesoRegistrarConSSHPassword(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	maquina := &stubMachine{addr: netip.MustParseAddr(ipPrueba), state: domain.StateRunning}
	guest := &stubConnector{name: conectorGuest, session: &stubSession{err: &domain.ExitError{Code: 1, Command: "id"}}}
	sshSesion := &stubSession{}
	sshPassword := &stubPasswordConnector{sesion: sshSesion}

	acceso := NuevoAcceso(&ConfigAcceso{
		Almacen:     almacen,
		Maquina:     maquina,
		Guest:       guest,
		SSH:         &stubConnector{name: conectorSSH},
		SSHPassword: sshPassword,
		Ejecutor:    &stubExecutor{},
		RutaLlave:   nuevaLlaveDePrueba(t),
		Objetivo:    func() domain.Target { return domain.Target{ID: nombreDebian} },
	})

	if _, err := acceso.Registrar(context.Background(), usuarioRoot, passwordPrueba); err != nil {
		t.Fatalf("Registrar() error = %v, want nil", err)
	}

	if sshPassword.llamadas != 1 {
		t.Errorf("ConnectPassword() se llamó %d veces, want 1", sshPassword.llamadas)
	}

	if !almacen.Registrada(nombreDebian) {
		t.Error("la máquina no quedó registrada con el respaldo SSH")
	}
}
