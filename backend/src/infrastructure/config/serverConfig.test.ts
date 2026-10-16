import { describe, expect, it } from 'vitest';
import { readServerConfig } from '@/infrastructure/config/serverConfig';

describe('readServerConfig', () => {
  it('nutzt 127.0.0.1:3000, wenn nichts gesetzt ist', () => {
    expect(readServerConfig({})).toEqual({ host: '127.0.0.1', port: 3000 });
  });

  it('behandelt leere Werte wie nicht gesetzt', () => {
    expect(readServerConfig({ HOST: '', PORT: '' })).toEqual({ host: '127.0.0.1', port: 3000 });
  });

  it('übernimmt gültige Loopback-Werte aus der Umgebung', () => {
    expect(readServerConfig({ HOST: 'localhost', PORT: '8080' })).toEqual({
      host: 'localhost',
      port: 8080,
    });
  });

  it.each(['127.0.0.1', '127.0.0.2', 'localhost', '::1'])(
    'erlaubt den Loopback-Host %s',
    (host) => {
      expect(readServerConfig({ HOST: host }).host).toBe(host);
    },
  );

  it.each(['0.0.0.0', '192.168.1.10', '::', 'example.org'])(
    'lehnt den Nicht-Loopback-Host %s ohne ALLOW_REMOTE ab (keine Authentifizierung in Phase 1)',
    (host) => {
      expect(() => readServerConfig({ HOST: host })).toThrow(/ALLOW_REMOTE/);
    },
  );

  it('erlaubt einen Nicht-Loopback-Host nur mit ALLOW_REMOTE=true', () => {
    expect(readServerConfig({ HOST: '0.0.0.0', ALLOW_REMOTE: 'true' }).host).toBe('0.0.0.0');
    expect(() => readServerConfig({ HOST: '0.0.0.0', ALLOW_REMOTE: 'yes' })).toThrow(
      /ALLOW_REMOTE/,
    );
  });

  it.each(['abc', '3000.5', '0', '65536', '-1'])('lehnt ungültigen PORT "%s" ab', (rawPort) => {
    expect(() => readServerConfig({ PORT: rawPort })).toThrow(`Ungültiger PORT: "${rawPort}"`);
  });
});
