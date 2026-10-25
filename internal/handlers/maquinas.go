// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Jorge Acosta y Andrey Tombe

package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"consola-vm/internal/services"
)

// ServicioMaquinas es el caso de uso del inventario de máquinas
// virtuales: buscar en el hipervisor y vincular una con la consola.
type ServicioMaquinas interface {
	// Listar devuelve las máquinas que coinciden con el filtro.
	Listar(ctx context.Context, filtro string) ([]services.MaquinaVirtual, error)
	// Vincular establece la máquina que administra la consola.
	Vincular(ctx context.Context, identificador string) (services.MaquinaVirtual, error)
}

// Maquinas expone GET /api/vms: el inventario que alimenta el selector
// de la barra superior.
type Maquinas struct {
	servicio ServicioMaquinas
}

// NuevoMaquinas construye el handler del inventario.
func NuevoMaquinas(servicio ServicioMaquinas) *Maquinas {
	return &Maquinas{servicio: servicio}
}

// maquinaRespuesta es una máquina del inventario en la respuesta JSON.
type maquinaRespuesta struct {
	Nombre string `json:"nombre"`
	UUID   string `json:"uuid"`
	Estado string `json:"estado"`
	Activa bool   `json:"activa"`
}

// listaMaquinas es el cuerpo de GET /api/vms.
type listaMaquinas struct {
	Maquinas []maquinaRespuesta `json:"maquinas"`
}

// nuevaMaquinaRespuesta traduce la máquina del servicio al contrato.
func nuevaMaquinaRespuesta(maquina services.MaquinaVirtual) maquinaRespuesta {
	return maquinaRespuesta{
		Nombre: maquina.Nombre,
		UUID:   maquina.UUID,
		Estado: string(maquina.Estado),
		Activa: maquina.Activa,
	}
}

// ServeHTTP implementa http.Handler: solo admite GET.
func (h *Maquinas) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !exigirMetodo(w, r, http.MethodGet) {
		return
	}

	maquinas, err := h.servicio.Listar(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		slog.Error("listar las máquinas virtuales", claveError, err)
		escribirJSON(w, http.StatusInternalServerError, respuestaError{
			Mensaje: "No fue posible consultar las máquinas virtuales del hipervisor.",
			Tecnico: err.Error(),
		})

		return
	}

	cuerpo := listaMaquinas{Maquinas: make([]maquinaRespuesta, 0, len(maquinas))}

	for _, maquina := range maquinas {
		cuerpo.Maquinas = append(cuerpo.Maquinas, nuevaMaquinaRespuesta(maquina))
	}

	escribirJSON(w, http.StatusOK, cuerpo)
}
