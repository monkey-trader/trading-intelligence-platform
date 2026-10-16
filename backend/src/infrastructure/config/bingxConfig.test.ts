import { describe, expect, it } from 'vitest';
import { missingBingXVariables, readBingXConfig } from '@/infrastructure/config/bingxConfig';

const KEY_VALUE = 'test-key-123';
const SECRET_VALUE = 'test-secret-456';

describe('readBingXConfig', () => {
  it('liefert Key, Secret und die Default-Basis-URL, wenn beide Keys gesetzt sind', () => {
    expect(readBingXConfig({ BX_KEY: KEY_VALUE, BX_SECRET: SECRET_VALUE })).toEqual({
      apiKey: KEY_VALUE,
      apiSecret: SECRET_VALUE,
      baseUrl: 'https://open-api.bingx.com',
    });
  });

  it('übernimmt BX_BASE als Basis-URL', () => {
    expect(
      readBingXConfig({
        BX_KEY: KEY_VALUE,
        BX_SECRET: SECRET_VALUE,
        BX_BASE: 'https://example.test',
      }),
    ).toEqual({
      apiKey: KEY_VALUE,
      apiSecret: SECRET_VALUE,
      baseUrl: 'https://example.test',
    });
  });

  it('nutzt die Default-Basis-URL, wenn BX_BASE leer ist', () => {
    const config = readBingXConfig({ BX_KEY: KEY_VALUE, BX_SECRET: SECRET_VALUE, BX_BASE: '  ' });

    expect(config?.baseUrl).toBe('https://open-api.bingx.com');
  });

  it('enthält keine weiteren Felder', () => {
    const config = readBingXConfig({
      BX_KEY: KEY_VALUE,
      BX_SECRET: SECRET_VALUE,
      JOURNAL_DB: '/tmp/journal.sqlite',
      IMPORT_FULL_DAYS: '90',
    });

    expect(Object.keys(config ?? {}).sort()).toEqual(['apiKey', 'apiSecret', 'baseUrl']);
  });

  it.each([
    ['beide fehlen', {}],
    ['BX_KEY fehlt', { BX_SECRET: SECRET_VALUE }],
    ['BX_SECRET fehlt', { BX_KEY: KEY_VALUE }],
    ['BX_KEY leer ist', { BX_KEY: '', BX_SECRET: SECRET_VALUE }],
    ['BX_SECRET leer ist', { BX_KEY: KEY_VALUE, BX_SECRET: '' }],
    ['BX_KEY nur Whitespace ist', { BX_KEY: '   ', BX_SECRET: SECRET_VALUE }],
    ['BX_SECRET nur Whitespace ist', { BX_KEY: KEY_VALUE, BX_SECRET: '\t ' }],
    ['beide leer sind', { BX_KEY: '', BX_SECRET: '' }],
  ])('liefert undefined, wenn %s', (_description, environment) => {
    expect(readBingXConfig(environment)).toBeUndefined();
  });

  it('wirft nicht, wenn die Keys fehlen', () => {
    expect(() => readBingXConfig({})).not.toThrow();
  });
});

describe('missingBingXVariables', () => {
  it('nennt nur die fehlenden Variablennamen, nie Werte', () => {
    expect(missingBingXVariables({ BX_KEY: 'geheimer-key' })).toEqual(['BX_SECRET']);
    expect(missingBingXVariables({ BX_SECRET: '  ' })).toEqual(['BX_KEY', 'BX_SECRET']);
    expect(missingBingXVariables({ BX_KEY: 'a', BX_SECRET: 'b' })).toEqual([]);
  });
});
