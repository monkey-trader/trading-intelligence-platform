import { describe, expect, it } from 'vitest';
import { readServerConfig } from '@/infrastructure/config/serverConfig';

const DEFAULT_ALLOWED_HOSTS = ['127.0.0.1:3000', 'localhost:3000', '[::1]:3000'];

describe('readServerConfig', () => {
  it('nutzt 127.0.0.1:3000, wenn nichts gesetzt ist', () => {
    expect(readServerConfig({})).toEqual({
      host: '127.0.0.1',
      port: 3000,
      allowedHosts: DEFAULT_ALLOWED_HOSTS,
    });
  });

  it('behandelt leere Werte wie nicht gesetzt', () => {
    expect(readServerConfig({ HOST: '', PORT: '' })).toEqual({
      host: '127.0.0.1',
      port: 3000,
      allowedHosts: DEFAULT_ALLOWED_HOSTS,
    });
  });

  it('übernimmt gültige Loopback-Werte aus der Umgebung', () => {
    expect(readServerConfig({ HOST: 'localhost', PORT: '8080' })).toEqual({
      host: 'localhost',
      port: 8080,
      allowedHosts: ['127.0.0.1:8080', 'localhost:8080', '[::1]:8080'],
    });
  });

  it('erlaubt für PORT=8080 die Loopback-Hosts mit Port 8080', () => {
    expect(readServerConfig({ PORT: '8080' }).allowedHosts).toEqual([
      '127.0.0.1:8080',
      'localhost:8080',
      '[::1]:8080',
    ]);
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
    expect(
      readServerConfig({
        HOST: '0.0.0.0',
        ALLOW_REMOTE: 'true',
        ALLOWED_HOSTS: 'journal.example:3000',
      }).host,
    ).toBe('0.0.0.0');
    expect(() =>
      readServerConfig({
        HOST: '0.0.0.0',
        ALLOW_REMOTE: 'yes',
        ALLOWED_HOSTS: 'journal.example:3000',
      }),
    ).toThrow(/ALLOW_REMOTE/);
  });

  it('ergänzt bei ALLOW_REMOTE=true die Einträge aus ALLOWED_HOSTS (getrimmt, kommagetrennt)', () => {
    const config = readServerConfig({
      HOST: '0.0.0.0',
      ALLOW_REMOTE: 'true',
      ALLOWED_HOSTS: ' journal.example:3000 ,192.168.1.10:3000 ',
    });

    expect(config.allowedHosts).toEqual([
      ...DEFAULT_ALLOWED_HOSTS,
      'journal.example:3000',
      '192.168.1.10:3000',
    ]);
  });

  it('ignoriert leere Einträge in ALLOWED_HOSTS', () => {
    const config = readServerConfig({
      HOST: '0.0.0.0',
      ALLOW_REMOTE: 'true',
      ALLOWED_HOSTS: 'journal.example:3000, ,',
    });

    expect(config.allowedHosts).toEqual([...DEFAULT_ALLOWED_HOSTS, 'journal.example:3000']);
  });

  it.each([undefined, '', '   ', ' , '])(
    'verlangt bei ALLOW_REMOTE=true ein gesetztes ALLOWED_HOSTS (Wert: %j)',
    (allowedHosts) => {
      expect(() =>
        readServerConfig({ HOST: '0.0.0.0', ALLOW_REMOTE: 'true', ALLOWED_HOSTS: allowedHosts }),
      ).toThrow(/ALLOWED_HOSTS/);
    },
  );

  it('ignoriert ALLOWED_HOSTS ohne ALLOW_REMOTE=true', () => {
    expect(readServerConfig({ ALLOWED_HOSTS: 'evil.example:3000' }).allowedHosts).toEqual(
      DEFAULT_ALLOWED_HOSTS,
    );
  });

  it.each(['abc', '3000.5', '0', '65536', '-1'])('lehnt ungültigen PORT "%s" ab', (rawPort) => {
    expect(() => readServerConfig({ PORT: rawPort })).toThrow(`Ungültiger PORT: "${rawPort}"`);
  });

  describe('Ergänzungen aus dem Review', () => {
    it('erlaubt zusätzlich den gebundenen Loopback-Host (z.B. 127.0.0.2)', () => {
      expect(readServerConfig({ HOST: '127.0.0.2', PORT: '3000' }).allowedHosts).toContain(
        '127.0.0.2:3000',
      );
    });

    it('nimmt den gebundenen Host nicht doppelt auf', () => {
      const { allowedHosts } = readServerConfig({ HOST: '127.0.0.1', PORT: '3000' });
      expect(allowedHosts.filter((allowedHost) => allowedHost === '127.0.0.1:3000')).toHaveLength(
        1,
      );
    });

    it('schreibt einen gebundenen IPv6-Loopback-Host mit Klammern', () => {
      expect(readServerConfig({ HOST: '::1', PORT: '3000' }).allowedHosts).toContain('[::1]:3000');
    });

    it.each(['meinrechner.local', 'meinrechner.local:', 'meinrechner.local:abc', 'a b:3000'])(
      'lehnt den ALLOWED_HOSTS-Eintrag "%s" ohne gültiges host:port ab',
      (entry) => {
        expect(() =>
          readServerConfig({ HOST: '0.0.0.0', ALLOW_REMOTE: 'true', ALLOWED_HOSTS: entry }),
        ).toThrow(/ALLOWED_HOSTS/);
      },
    );

    it('akzeptiert IPv6 mit Klammern in ALLOWED_HOSTS', () => {
      expect(
        readServerConfig({ HOST: '0.0.0.0', ALLOW_REMOTE: 'true', ALLOWED_HOSTS: '[fe80::1]:3000' })
          .allowedHosts,
      ).toContain('[fe80::1]:3000');
    });
  });
});
