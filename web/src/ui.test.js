// @vitest-environment happy-dom

import { describe, expect, it, vi } from 'vitest';

import { ApiError } from './api.js';
import {
  escapar,
  mensajeDeError,
  presentarEstado,
  presentarEstadoServidor,
  presentarEstadoUnidad,
  resultadoDeError,
} from './ui.js';

describe('utilidades de presentación', () => {
  it('escapa los caracteres peligrosos de una plantilla', () => {
    expect(escapar('<script>alert("x")</script>')).toBe('&lt;script&gt;alert(&quot;x&quot;)&lt;/script&gt;');
    expect(escapar(null)).toBe('');
    expect(escapar(0)).toBe('0');
  });

  it('traduce los estados de encendido del contrato', () => {
    expect(presentarEstado('running').texto).toBe('Encendida');
    expect(presentarEstado('poweroff').texto).toBe('Apagada');
    expect(presentarEstado('paused').texto).toBe('Pausada');
    expect(presentarEstado('saved').texto).toBe('Apagada');
    expect(presentarEstado('desconocido').texto).toBe('Desconocido');
  });

  it('traduce los estados de una unidad systemd', () => {
    expect(presentarEstadoUnidad('active').texto).toBe('activa');
    expect(presentarEstadoUnidad('failed').texto).toBe('falló');
    expect(presentarEstadoUnidad('inactive').texto).toBe('inactiva');
  });

  it('traduce los estados de un servidor del back-end', () => {
    expect(presentarEstadoServidor('up').texto).toBe('UP');
    expect(presentarEstadoServidor('down').texto).toBe('DOWN');
    expect(presentarEstadoServidor('otro').texto).toBe('DESCONOCIDO');
  });

  it('explica un módulo todavía no conectado', () => {
    const error = new ApiError(501, { mensaje: 'La operación aún no está conectada.' });

    expect(mensajeDeError(error)).toBe('Este módulo aún no está conectado con la máquina.');
  });

  it('conserva el mensaje del servidor para el resto de errores', () => {
    const error = new ApiError(400, { mensaje: 'el campo "nombre" no es válido', campo: 'nombre' });

    expect(mensajeDeError(error)).toBe('el campo "nombre" no es válido');
    expect(resultadoDeError(error, 'POST /api/packages')).toEqual({
      ok: false,
      mensaje: 'el campo "nombre" no es válido',
      operacion: 'POST /api/packages',
      codigo: undefined,
      stderr: '',
      tecnico: 'nombre',
      solicitud: '',
    });
  });

  it('explica un fallo de red', () => {
    expect(mensajeDeError(new TypeError('Failed to fetch'))).toContain('No fue posible contactar');
  });
});

describe('ejecución de operaciones', () => {
  it('entrega el resultado correcto y recarga la lista', async () => {
    const informar = vi.fn();
    const recargar = vi.fn();
    const { ejecutar } = await import('./ui.js');

    const exito = await ejecutar({
      accion: async () => ({ mensaje: 'Hecho', operacion: 'apt-get install -y nano', codigo: 0 }),
      operacion: 'POST /api/packages',
      informar,
      recargar,
    });

    expect(exito).toBe(true);
    expect(informar).toHaveBeenCalledWith(
      expect.objectContaining({ ok: true, mensaje: 'Hecho', operacion: 'apt-get install -y nano' }),
    );
    expect(recargar).toHaveBeenCalledOnce();
  });

  it('informa el error sin recargar la lista', async () => {
    const informar = vi.fn();
    const recargar = vi.fn();
    const { ejecutar } = await import('./ui.js');

    const exito = await ejecutar({
      accion: async () => {
        throw new ApiError(503, { mensaje: 'Sin sesión', tecnico: 'sin conector' });
      },
      operacion: 'GET /api/packages',
      informar,
      recargar,
    });

    expect(exito).toBe(false);
    expect(informar).toHaveBeenCalledWith(
      expect.objectContaining({ ok: false, mensaje: 'Sin sesión', tecnico: 'sin conector' }),
    );
    expect(recargar).not.toHaveBeenCalled();
  });

  it('muestra y actualiza una alerta flotante en el documento', async () => {
    const { mostrarAlerta } = await import('./ui.js');

    const alerta = mostrarAlerta({
      tipo: 'progreso',
      titulo: 'Operación iniciada',
      mensaje: 'Instalando paquete...',
    });

    const contenedor = document.getElementById('toast-container');
    expect(contenedor).not.toBeNull();
    expect(contenedor.textContent).toContain('Operación iniciada');
    expect(contenedor.textContent).toContain('Instalando paquete...');

    alerta.actualizar({
      tipo: 'exito',
      titulo: 'Operación completada',
      mensaje: 'Instalación exitosa',
    });

    expect(contenedor.textContent).toContain('Operación completada');
    expect(contenedor.textContent).toContain('Instalación exitosa');

    alerta.cerrar();
  });

  it('inhabilita los controles y muestra indicador en el botón principal', async () => {
    const { inhabilitar } = await import('./ui.js');
    const form = document.createElement('form');
    form.innerHTML = `
      <input type="text" name="paquete" value="curl" />
      <button type="submit" class="btn btn-primary">Ejecutar</button>
    `;
    document.body.appendChild(form);

    const input = form.querySelector('input');
    const boton = form.querySelector('button');

    inhabilitar(form, true);
    expect(input.disabled).toBe(true);
    expect(boton.disabled).toBe(true);
    expect(boton.textContent).toContain('Ejecutando…');

    inhabilitar(form, false);
    expect(input.disabled).toBe(false);
    expect(boton.disabled).toBe(false);
    expect(boton.textContent).toContain('Ejecutar');

    form.remove();
  });

  it('no muestra alertas flotantes ni durante ni al completar una operación', async () => {
    document.getElementById('toast-container')?.remove();

    const { ejecutar } = await import('./ui.js');

    let complete;
    const pending = new Promise((resolve) => {
      complete = resolve;
    });

    const execution = ejecutar({
      accion: () => pending,
      operacion: 'POST /api/packages',
      informar: vi.fn(),
    });

    expect(document.getElementById('toast-container')).toBeNull();

    complete({ mensaje: 'Hecho' });
    await execution;

    expect(document.getElementById('toast-container')).toBeNull();
  });

  it('muestra una alerta flotante cuando la operación falla', async () => {
    document.getElementById('toast-container')?.remove();

    const { ejecutar } = await import('./ui.js');

    const exito = await ejecutar({
      accion: async () => {
        throw new ApiError(503, { mensaje: 'Sin sesión' });
      },
      operacion: 'GET /api/packages',
      informar: vi.fn(),
    });

    expect(exito).toBe(false);

    const container = document.getElementById('toast-container');
    expect(container.textContent).toContain('Error en la operación');

    container.remove();
  });
});

