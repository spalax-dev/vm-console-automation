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
)

// accesoCon arma un servicio de registro con las piezas indicadas.
func accesoCon(
	t *testing.T, almacen *Almacen, maquina *stubMachine, guest *stubConnector,
	sshConector *stubConnector, sshPassword *stubPasswordConnector, rutaLlave string,
) *Acceso {
	t.Helper()

	return NuevoAcceso(&ConfigAcceso{
		Almacen:     almacen,
		Maquina:     maquina,
		Guest:       guest,
		SSH:         sshConector,
		SSHPassword: sshPassword,
		Ejecutor:    &stubExecutor{},
		RutaLlave:   rutaLlave,
		Objetivo:    func() domain.Target { return domain.Target{ID: nombreDebian} },
	})
}

// TestAccesoGuardarCredencialesFalla propaga el fallo de SQLite.
func TestAccesoGuardarCredencialesFalla(t *testing.T) {
	t.Parallel()

	almacen := conBaseCerrada(t)
	acceso := accesoCon(t, almacen, &stubMachine{addr: netip.MustParseAddr(ipPrueba), state: domain.StateRunning},
		&stubConnector{}, &stubConnector{}, nil, nuevaLlaveDePrueba(t))

	if _, err := acceso.Registrar(context.Background(), usuarioRoot, passwordPrueba); err == nil {
		t.Error("Registrar() error = nil, want el fallo al guardar")
	}
}

// TestAccesoGuardarRegistroFinalFalla propaga el fallo al marcar el
// registro.
func TestAccesoGuardarRegistroFinalFalla(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)

	if _, err := almacen.db.ExecContext(context.Background(),
		`CREATE TRIGGER bloquear_update BEFORE UPDATE ON credenciales
		 BEGIN SELECT RAISE(ABORT, 'sin escritura'); END;`); err != nil {
		t.Fatalf("crear el disparador: %v", err)
	}

	acceso := accesoCon(t, almacen, &stubMachine{addr: netip.MustParseAddr(ipPrueba), state: domain.StateRunning},
		&stubConnector{session: &stubSession{}}, &stubConnector{}, nil, nuevaLlaveDePrueba(t))

	if _, err := acceso.Registrar(context.Background(), usuarioRoot, passwordPrueba); err == nil {
		t.Error("Registrar() error = nil, want el fallo al marcar el registro")
	}

	if _, ok := almacen.Obtener(nombreDebian); ok {
		t.Error("Obtener() encontró credenciales de un registro fallido")
	}
}

// TestAccesoResolverErrores propaga los fallos de VirtualBox.
func TestAccesoResolverErrores(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)

	sinEstado := accesoCon(t, almacen, &stubMachine{stateErr: errors.New("sin VBoxManage")},
		&stubConnector{}, &stubConnector{}, nil, nuevaLlaveDePrueba(t))

	if _, err := sinEstado.Registrar(context.Background(), usuarioRoot, passwordPrueba); err == nil {
		t.Error("Registrar() error = nil, want el fallo del estado")
	}

	sinDireccion := accesoCon(t, almacen, &stubMachine{state: domain.StateRunning, addrErr: errors.New("sin IP")},
		&stubConnector{}, &stubConnector{}, nil, nuevaLlaveDePrueba(t))

	if _, err := sinDireccion.Registrar(context.Background(), usuarioRoot, passwordPrueba); err == nil {
		t.Error("Registrar() error = nil, want el fallo de la dirección")
	}
}

// TestAccesoGuestAdditionsNoConecta propaga el fallo del conector.
func TestAccesoGuestAdditionsNoConecta(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	guest := &stubConnector{connect: errors.New("sin Guest Additions")}

	acceso := accesoCon(t, almacen, &stubMachine{addr: netip.MustParseAddr(ipPrueba), state: domain.StateRunning},
		guest, &stubConnector{}, nil, nuevaLlaveDePrueba(t))

	if _, err := acceso.Registrar(context.Background(), usuarioRoot, passwordPrueba); err == nil {
		t.Error("Registrar() error = nil, want el fallo del conector")
	}
}

// TestAccesoRespaldoSSHFalla cubre el respaldo por contraseña cuando la
// llave no está disponible o el conector falla.
func TestAccesoRespaldoSSHFalla(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	guest := &stubConnector{session: &stubSession{err: &domain.ExitError{Code: 1, Command: "id"}}}

	// La llave vive bajo un archivo: comprobar su estado falla antes de
	// intentar el respaldo por contraseña.
	rutaImposible := filepath.Join(archivoConTexto(t), "id_ed25519")
	acceso := accesoCon(t, almacen, &stubMachine{addr: netip.MustParseAddr(ipPrueba), state: domain.StateRunning},
		guest, &stubConnector{}, &stubPasswordConnector{}, rutaImposible)

	if _, err := acceso.Registrar(context.Background(), usuarioRoot, passwordPrueba); err == nil {
		t.Error("Registrar() error = nil, want el fallo al comprobar la llave")
	}

	if _, err := acceso.prepararSSHPassword(context.Background(),
		domain.Target{ID: nombreDebian}, netip.MustParseAddr(ipPrueba), usuarioRoot, passwordPrueba,
	); err == nil {
		t.Error("prepararSSHPassword() error = nil, want el fallo al comprobar la llave")
	}

	// El conector de contraseña falla al abrir la sesión.
	otroAlmacen, _, _ := nuevaBaseDePrueba(t)
	conError := accesoCon(t, otroAlmacen, &stubMachine{addr: netip.MustParseAddr(ipPrueba), state: domain.StateRunning},
		guest, &stubConnector{}, &stubPasswordConnector{err: errors.New("sin sshpass")}, nuevaLlaveDePrueba(t))

	if _, err := conError.Registrar(context.Background(), usuarioRoot, passwordPrueba); !errors.Is(err, ErrCredencialesInvalidas) {
		t.Errorf("Registrar() error = %v, want ErrCredencialesInvalidas", err)
	}

	// La sesión por contraseña rechaza la orden de verificación.
	tercerAlmacen, _, _ := nuevaBaseDePrueba(t)
	sesionRota := &stubSession{err: &domain.ExitError{Code: 1, Command: "id"}}
	conSesionRota := accesoCon(t, tercerAlmacen, &stubMachine{addr: netip.MustParseAddr(ipPrueba), state: domain.StateRunning},
		guest, &stubConnector{}, &stubPasswordConnector{sesion: sesionRota}, nuevaLlaveDePrueba(t))

	if _, err := conSesionRota.Registrar(context.Background(), usuarioRoot, passwordPrueba); !errors.Is(err, ErrCredencialesInvalidas) {
		t.Errorf("Registrar() error = %v, want ErrCredencialesInvalidas", err)
	}
}

// TestAccesoInstalacionFalla propaga el fallo al preparar SSH.
func TestAccesoInstalacionFalla(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	sesion := &stubSession{errs: []error{nil, errors.New("sin espacio")}}

	acceso := accesoCon(t, almacen, &stubMachine{addr: netip.MustParseAddr(ipPrueba), state: domain.StateRunning},
		&stubConnector{session: sesion}, &stubConnector{}, nil, nuevaLlaveDePrueba(t))

	if _, err := acceso.Registrar(context.Background(), usuarioRoot, passwordPrueba); err == nil {
		t.Error("Registrar() error = nil, want el fallo de la instalación")
	}

	if _, ok := almacen.Obtener(nombreDebian); ok {
		t.Error("Obtener() encontró credenciales tras fallar la instalación")
	}
}

// TestAccesoRegistrarFallidoRestauraCredencialesAnteriores comprueba que
// un registro fallido no pisa las credenciales que ya funcionaban.
func TestAccesoRegistrarFallidoRestauraCredencialesAnteriores(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)

	anteriores := Credenciales{
		Usuario: "jorge", Password: "anterior", Puerto: 2222, Direccion: "10.0.0.9", Registrada: true,
	}
	if err := almacen.Guardar(nombreDebian, anteriores); err != nil {
		t.Fatalf("Guardar() error = %v, want nil", err)
	}

	guest := &stubConnector{session: &stubSession{err: &domain.ExitError{Code: 1, Command: "id"}}}
	acceso := accesoCon(t, almacen, &stubMachine{addr: netip.MustParseAddr(ipPrueba), state: domain.StateRunning},
		guest, &stubConnector{}, &stubPasswordConnector{err: errors.New("sin sshpass")}, nuevaLlaveDePrueba(t))

	if _, err := acceso.Registrar(context.Background(), usuarioRoot, passwordPrueba); !errors.Is(err, ErrCredencialesInvalidas) {
		t.Fatalf("Registrar() error = %v, want ErrCredencialesInvalidas", err)
	}

	guardadas, ok := almacen.Obtener(nombreDebian)
	if !ok {
		t.Fatal("Obtener() no encontró las credenciales anteriores")
	}

	if guardadas != anteriores {
		t.Errorf("Obtener() = %+v, want %+v", guardadas, anteriores)
	}
}

// TestAccesoOlvidarErrores cubre la ausencia de máquina y el fallo de
// SQLite.
func TestAccesoOlvidarErrores(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	sinMaquina := NuevoAcceso(&ConfigAcceso{
		Almacen:  almacen,
		Objetivo: func() domain.Target { return domain.Target{} },
	})

	if err := sinMaquina.Olvidar(); !errors.Is(err, ErrSinMaquina) {
		t.Errorf("Olvidar() error = %v, want ErrSinMaquina", err)
	}

	acceso := accesoCon(t, conBaseCerrada(t), &stubMachine{}, &stubConnector{}, &stubConnector{}, nil, "")

	if err := acceso.Olvidar(); err == nil {
		t.Error("Olvidar() error = nil, want el fallo de SQLite")
	}
}

// TestAsegurarLlaveErrores cubre los fallos al preparar la llave.
func TestAsegurarLlaveErrores(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)

	rutaImposible := filepath.Join(archivoConTexto(t), "id_ed25519")
	if _, err := accesoCon(t, almacen, &stubMachine{}, &stubConnector{}, &stubConnector{}, nil,
		rutaImposible).asegurarLlave(context.Background()); err == nil {
		t.Error("asegurarLlave() error = nil, want el fallo de comprobación")
	}

	rutaBajoArchivo := filepath.Join(archivoConTexto(t), "sub", "id_ed25519")
	if _, err := accesoCon(t, almacen, &stubMachine{}, &stubConnector{}, &stubConnector{}, nil,
		rutaBajoArchivo).asegurarLlave(context.Background()); err == nil {
		t.Error("asegurarLlave() error = nil, want el fallo del directorio")
	}

	acceso := accesoCon(t, almacen, &stubMachine{}, &stubConnector{}, &stubConnector{}, nil,
		filepath.Join(t.TempDir(), "id_ed25519"))
	acceso.ejecutor = &stubExecutor{err: errors.New("sin ssh-keygen")}

	if _, err := acceso.asegurarLlave(context.Background()); err == nil {
		t.Error("asegurarLlave() error = nil, want el fallo del generador")
	}

	// El generador crea la llave pública: el camino correcto la devuelve.
	ruta := filepath.Join(t.TempDir(), nombreLlave)
	generador := &stubExecutor{onExecute: func() {
		if err := os.WriteFile(ruta+".pub", []byte(clavePublica+"\n"), 0o600); err != nil {
			t.Errorf("escribir la pública: %v", err)
		}
	}}
	completo := accesoCon(t, almacen, &stubMachine{}, &stubConnector{}, &stubConnector{}, nil, ruta)
	completo.ejecutor = generador

	clave, err := completo.asegurarLlave(context.Background())
	if err != nil || clave != clavePublica {
		t.Errorf("asegurarLlave() = %q, %v, want la clave pública", clave, err)
	}
}
