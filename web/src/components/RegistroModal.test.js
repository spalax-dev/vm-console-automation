// @vitest-environment happy-dom

import { afterEach, describe, expect, it, vi } from 'vitest';

import { abrirRegistro } from './RegistroModal.js';

afterEach(() => {
  vi.unstubAllGlobals();
  document.body.innerHTML = '';
});

describe('registration dialog', () => {
  it('opens without diagnostics or automatic requests', () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);

    abrirRegistro({ maquina: { nombre: 'debian-1' } });

    expect(document.querySelector('#registro-form')).not.toBeNull();
    expect(document.querySelector('#registro-diagnostico')).toBeNull();
    expect(document.querySelector('#diag-ga')).toBeNull();
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
