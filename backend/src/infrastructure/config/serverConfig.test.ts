import { describe, expect, it } from 'vitest';
import { readServerConfig } from '@/infrastructure/config/serverConfig';

describe('readServerConfig', () => {
  it('nutzt 127.0.0.1:3000, wenn nichts gesetzt ist', () => {
    expect(readServerConfig({})).toEqual({ host: '127.0.0.1', port: 3000 });
  });

  it('behandelt leere Werte wie nicht gesetzt', () => {
    expect(readServerConfig({ HOST: '', PORT: '' })).toEqual({ host: '127.0.0.1', port: 3000 });
  });

  it('übernimmt gültige Werte aus der Umgebung', () => {
    expect(readServerConfig({ HOST: '0.0.0.0', PORT: '8080' })).toEqual({
      host: '0.0.0.0',
      port: 8080,
    });
  });

  it.each(['abc', '3000.5', '0', '65536', '-1'])('lehnt ungültigen PORT "%s" ab', (rawPort) => {
    expect(() => readServerConfig({ PORT: rawPort })).toThrow(`Ungültiger PORT: "${rawPort}"`);
  });
});
