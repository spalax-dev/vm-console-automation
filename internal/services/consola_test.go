// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// usuarioConsola y passwordConsola son las credenciales de las pruebas.
const (
	usuarioConsola  = "jorge"
	usuarioAndrey   = "andrey"
	passwordConsola = "una-clave-larga" // #nosec G101 -- contraseña de prueba.
	msgConfigurarOK = "Configurar() error = %v, want nil"
)

// TestConsolaSinCuenta comprueba el estado inicial.
func TestConsolaSinCuenta(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	consola := NuevoConsola(almacen)

	configurada, err := consola.Configurada()
	if err != nil {
		t.Fatalf("Configurada() error = %v, want nil", err)
	}

	if configurada {
		t.Error("Configurada() = true, want false en una base nueva")
	}

	if _, ok := consola.Validar(""); ok {
		t.Error("Validar() = true, want false sin cookie")
	}

	if _, ok := consola.Validar("jorge|9999999999|firma"); ok {
		t.Error("Validar() = true, want false con una firma inventada")
	}
}

// TestConsolaConfiguraYAutentica crea la cuenta y comprueba el acceso.
func TestConsolaConfiguraYAutentica(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	consola := NuevoConsola(almacen)
	ctx := context.Background()

	if err := consola.Configurar(ctx, usuarioConsola, passwordConsola); err != nil {
		t.Fatalf(msgConfigurarOK, err)
	}

	configurada, err := consola.Configurada()
	if err != nil || !configurada {
		t.Fatalf("Configurada() = %v, %v, want true", configurada, err)
	}

	casos := []struct {
		nombre   string
		usuario  string
		password string
		esperado bool
	}{
		{"correctas", usuarioConsola, passwordConsola, true},
		{"contraseña incorrecta", usuarioConsola, "otra-clave", false},
		{"usuario inexistente", usuarioAndrey, passwordConsola, false},
		{"usuario inválido", "Jorge", passwordConsola, false},
		{"contraseña vacía", usuarioConsola, "", false},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			ok, err := consola.Autenticar(ctx, caso.usuario, caso.password)
			if err != nil {
				t.Fatalf("Autenticar() error = %v, want nil", err)
			}

			if ok != caso.esperado {
				t.Errorf("Autenticar() = %v, want %v", ok, caso.esperado)
			}
		})
	}
}

// TestConsolaDatosInvalidos rechaza usuarios y contraseñas fuera de las
// reglas antes de tocar la base.
func TestConsolaDatosInvalidos(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	consola := NuevoConsola(almacen)

	casos := []struct {
		nombre   string
		usuario  string
		password string
	}{
		{"usuario con mayúscula", "Jorge", passwordConsola},
		{"usuario corto", "ab", passwordConsola},
		{"usuario con espacio", "jor ge", passwordConsola},
		{"contraseña corta", usuarioConsola, "corta"},
		{"contraseña con NUL", usuarioConsola, "una\x00clave"},
		{"contraseña larga", usuarioConsola, strings.Repeat("a", maxPasswordConsola+1)},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()

			err := consola.Configurar(context.Background(), caso.usuario, caso.password)
			if !errors.Is(err, ErrDatosConsola) {
				t.Errorf("Configurar() error = %v, want ErrDatosConsola", err)
			}
		})
	}
}

// TestConsolaSoloUnaCuenta impide crear una segunda cuenta.
func TestConsolaSoloUnaCuenta(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	consola := NuevoConsola(almacen)
	ctx := context.Background()

	if err := consola.Configurar(ctx, usuarioConsola, passwordConsola); err != nil {
		t.Fatalf(msgConfigurarOK, err)
	}

	err := consola.Configurar(ctx, usuarioAndrey, passwordConsola)
	if !errors.Is(err, ErrConsolaConfigurada) {
		t.Errorf("Configurar() error = %v, want ErrConsolaConfigurada", err)
	}
}

// TestConsolaSesionFirmada comprueba la emisión y la validación de la
// cookie, incluida su vigencia.
func TestConsolaSesionFirmada(t *testing.T) {
	t.Parallel()

	almacen, _, _ := nuevaBaseDePrueba(t)
	consola := NuevoConsola(almacen)

	if err := consola.Configurar(context.Background(), usuarioConsola, passwordConsola); err != nil {
		t.Fatalf(msgConfigurarOK, err)
	}

	token, err := consola.AbrirSesion(usuarioConsola)
	if err != nil {
		t.Fatalf("AbrirSesion() error = %v, want nil", err)
	}

	usuario, ok := consola.Validar(token)
	if !ok || usuario != usuarioConsola {
		t.Fatalf("Validar() = %q, %v, want %q, true", usuario, ok, usuarioConsola)
	}

	partes := strings.Split(token, "|")
	manipulado := partes[0] + "|" + partes[1] + "|" + strings.Repeat("0", len(partes[2]))

	if _, ok := consola.Validar(manipulado); ok {
		t.Error("Validar() = true, want false con la firma alterada")
	}

	if _, ok := consola.Validar("solo-dos|partes"); ok {
		t.Error("Validar() = true, want false con formato incompleto")
	}

	// Doce horas después la sesión vence.
	consola.ahora = func() time.Time { return time.Now().Add(VigenciaSesionConsola + time.Minute) }

	if _, ok := consola.Validar(token); ok {
		t.Error("Validar() = true, want false con la sesión vencida")
	}
}

// TestConsolaSecretoPersistente valida una cookie emitida por otra
// instancia de la consola sobre la misma base.
func TestConsolaSecretoPersistente(t *testing.T) {
	t.Parallel()

	almacen, rutaBD, rutaClave := nuevaBaseDePrueba(t)
	primera := NuevoConsola(almacen)

	if err := primera.Configurar(context.Background(), usuarioConsola, passwordConsola); err != nil {
		t.Fatalf(msgConfigurarOK, err)
	}

	token, err := primera.AbrirSesion(usuarioConsola)
	if err != nil {
		t.Fatalf("AbrirSesion() error = %v, want nil", err)
	}

	otro, err := NuevoAlmacen(rutaBD, rutaClave)
	if err != nil {
		t.Fatalf("NuevoAlmacen() error = %v, want nil", err)
	}

	usuario, ok := NuevoConsola(otro).Validar(token)
	if !ok || usuario != usuarioConsola {
		t.Errorf("Validar() tras reabrir la base = %q, %v, want %q, true", usuario, ok, usuarioConsola)
	}
}
