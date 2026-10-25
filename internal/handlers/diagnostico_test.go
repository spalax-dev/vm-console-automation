// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/netip"
	"testing"

	"consola-vm/internal/core/domain"
	"consola-vm/internal/services"
)

// rutaDiagnostico es la ruta del recurso de diagnóstico.
const rutaDiagnostico = "/api/session/diagnostico"

// stubDiagnostico devuelve un informe preprogramado.
type stubDiagnostico struct {
	informe services.Diagnostico
	err     error
}

// Obtener devuelve el informe o el error configurados.
func (s *stubDiagnostico) Obtener(context.Context) (services.Diagnostico, error) {
	return s.informe, s.err
}

// TestDiagnosticoHandlerRespondeInforme cubre la respuesta del informe.
func TestDiagnosticoHandlerRespondeInforme(t *testing.T) {
	t.Parallel()

	servicio := &stubDiagnostico{informe: services.Diagnostico{
		VM:                     maquinaDebian,
		Estado:                 domain.StateRunning,
		Red:                    domain.NetworkBridged,
		IP:                     netipMustParse(t, ipPrueba),
		GuestAdditionsVersion:  "6.0.0 r127566",
		GuestAdditionsRunLevel: 2,
		UsuarioSugerido:        usuarioJorge,
		Problemas:              []string{"SSH no responde"},
		Sugerencias:            []string{"Instala openssh-server"},
	}}

	rec := ejecutarCaso(t, NuevoDiagnostico(servicio).ServeHTTP, rutaDiagnostico, casoPeticion{
		nombre: "diagnóstico", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf(msgCodigo, rec.Code, http.StatusOK)
	}

	var cuerpo diagnosticoRespuesta
	if err := json.Unmarshal(rec.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf(msgJSONInvalido, err)
	}

	if cuerpo.VM != maquinaDebian || cuerpo.UsuarioSugerido != usuarioJorge || cuerpo.SSHAbierto {
		t.Errorf("cuerpo = %+v, want debian/jorge/SSH cerrado", cuerpo)
	}
}

// TestDiagnosticoHandlerSinMaquina cubre la consola sin vínculo.
func TestDiagnosticoHandlerSinMaquina(t *testing.T) {
	t.Parallel()

	rec := ejecutarCaso(t, NuevoDiagnostico(&stubDiagnostico{err: services.ErrSinMaquina}).ServeHTTP, rutaDiagnostico, casoPeticion{
		nombre: "sin máquina", metodo: http.MethodGet,
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf(msgCodigo, rec.Code, http.StatusConflict)
	}
}

// TestDiagnosticoHandlerMetodo cubre el método no permitido.
func TestDiagnosticoHandlerMetodo(t *testing.T) {
	t.Parallel()

	rec := ejecutarCaso(t, NuevoDiagnostico(&stubDiagnostico{}).ServeHTTP, rutaDiagnostico, casoPeticion{
		nombre: casoMetodoNoPermitido, metodo: http.MethodPost,
	})

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(msgCodigo, rec.Code, http.StatusMethodNotAllowed)
	}
}

// netipMustParse evita importar net/netip en varias aserciones.
func netipMustParse(t *testing.T, valor string) (direccion netip.Addr) {
	t.Helper()

	direccion, err := netip.ParseAddr(valor)
	if err != nil {
		t.Fatalf("ParseAddr(%q): %v", valor, err)
	}

	return direccion
}
