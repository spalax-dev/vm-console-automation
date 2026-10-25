import { describe, expect, it } from 'vitest';

import {
  esIpv4,
  esModo,
  esPuerto,
  esRutaAbsoluta,
  identificadorServidor,
  validarNombreHost,
  validarNombrePaquete,
  validarNombreUnidad,
  validarNombreUsuario,
} from './validacion.js';

describe('validación del contrato', () => {
  it('acepta los nombres de paquete del contrato', () => {
    for (const nombre of ['nano', 'nginx', 'python3-pip', 'libstdc++6']) {
      expect(validarNombrePaquete(nombre)).toBeNull();
    }
  });

  it('rechaza nombres de paquete con mayúsculas o espacios', () => {
    for (const nombre of ['Nano', 'na no', '-nano', '']) {
      expect(validarNombrePaquete(nombre)).toContain('paquete');
    }
  });

  it('acepta los nombres de usuario POSIX del contrato', () => {
    for (const nombre of ['jorge', 'andrey_t', 'soporte-1', '_svc']) {
      expect(validarNombreUsuario(nombre)).toBeNull();
    }
  });

  it('rechaza nombres de usuario que empiezan con dígito o exceden 32', () => {
    expect(validarNombreUsuario('1jorge')).toContain('usuario');
    expect(validarNombreUsuario('Jorge')).toContain('usuario');
    expect(validarNombreUsuario('a'.repeat(33))).toContain('usuario');
  });

  it('acepta nombres de host RFC 1123', () => {
    for (const nombre of ['debian13', 'nodo-web-01', 'a']) {
      expect(validarNombreHost(nombre)).toBeNull();
    }
  });

  it('rechaza nombres de host con guion en los extremos', () => {
    for (const nombre of ['-nodo', 'nodo-', 'no_do', '']) {
      expect(validarNombreHost(nombre)).toContain('host');
    }
  });

  it('acepta unidades con extensión .service', () => {
    for (const unidad of ['catalogo.service', 'vmcons-tarea.service', 'app@1.service']) {
      expect(validarNombreUnidad(unidad)).toBeNull();
    }
  });

  it('rechaza unidades sin extensión .service', () => {
    expect(validarNombreUnidad('catalogo')).toContain('.service');
    expect(validarNombreUnidad('catalogo.timer')).toContain('.service');
  });

  it('valida direcciones IPv4', () => {
    expect(esIpv4('192.168.1.60')).toBe(true);
    expect(esIpv4('0.0.0.0')).toBe(true);
    expect(esIpv4('255.255.255.255')).toBe(true);
    expect(esIpv4('256.1.1.1')).toBe(false);
    expect(esIpv4('192.168.1')).toBe(false);
    expect(esIpv4('192.168.1.1.1')).toBe(false);
  });

  it('valida el rango de puertos', () => {
    expect(esPuerto(1)).toBe(true);
    expect(esPuerto(65535)).toBe(true);
    expect(esPuerto(0)).toBe(false);
    expect(esPuerto(65536)).toBe(false);
    expect(esPuerto('8080')).toBe(false);
  });

  it('valida el modo octal', () => {
    expect(esModo('644')).toBe(true);
    expect(esModo('0755')).toBe(true);
    expect(esModo('999')).toBe(false);
    expect(esModo('64')).toBe(false);
  });

  it('valida rutas absolutas', () => {
    expect(esRutaAbsoluta('/opt/catalogo/app')).toBe(true);
    expect(esRutaAbsoluta('opt/catalogo')).toBe(false);
    expect(esRutaAbsoluta('')).toBe(false);
  });

  it('compone el identificador del servidor del back-end', () => {
    expect(identificadorServidor('192.168.1.61', 8080)).toBe('192.168.1.61:8080');
  });
});
