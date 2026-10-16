import { describe, expect, it } from 'vitest';
import { readImportConfig } from '@/infrastructure/config/importConfig';

const SECRET_KEY_VALUE = 'geheimer-key-123';
const SECRET_SECRET_VALUE = 'geheimes-secret-456';

function captureError(action: () => unknown): Error {
  try {
    action();
  } catch (error) {
    if (error instanceof Error) return error;
    throw new Error('Es wurde ein Error-Objekt erwartet.', { cause: error });
  }
  throw new Error('Es wurde ein Fehler erwartet, aber keiner geworfen.');
}

describe('readImportConfig', () => {
  it('nutzt die Defaults, wenn nur die Pflichtvariablen gesetzt sind', () => {
    expect(readImportConfig({ BX_KEY: SECRET_KEY_VALUE, BX_SECRET: SECRET_SECRET_VALUE })).toEqual({
      apiKey: SECRET_KEY_VALUE,
      apiSecret: SECRET_SECRET_VALUE,
      baseUrl: 'https://open-api.bingx.com',
      databasePath: 'data/journal.sqlite',
      fullImportDays: 365,
    });
  });

  it('übernimmt gesetzte Werte aus der Umgebung', () => {
    expect(
      readImportConfig({
        BX_KEY: SECRET_KEY_VALUE,
        BX_SECRET: SECRET_SECRET_VALUE,
        BX_BASE: 'https://example.test',
        JOURNAL_DB: '/tmp/journal-test.sqlite',
        IMPORT_FULL_DAYS: '90',
      }),
    ).toEqual({
      apiKey: SECRET_KEY_VALUE,
      apiSecret: SECRET_SECRET_VALUE,
      baseUrl: 'https://example.test',
      databasePath: '/tmp/journal-test.sqlite',
      fullImportDays: 90,
    });
  });

  it('behandelt leere optionale Werte wie nicht gesetzt', () => {
    const config = readImportConfig({
      BX_KEY: SECRET_KEY_VALUE,
      BX_SECRET: SECRET_SECRET_VALUE,
      BX_BASE: '',
      JOURNAL_DB: '  ',
      IMPORT_FULL_DAYS: '',
    });

    expect(config.baseUrl).toBe('https://open-api.bingx.com');
    expect(config.databasePath).toBe('data/journal.sqlite');
    expect(config.fullImportDays).toBe(365);
  });

  it('nennt BX_KEY, wenn der Key fehlt, ohne den Wert des Secrets zu verraten', () => {
    const error = captureError(() => readImportConfig({ BX_SECRET: SECRET_SECRET_VALUE }));

    expect(error.message).toContain('BX_KEY');
    expect(error.message).not.toContain(SECRET_SECRET_VALUE);
  });

  it('nennt BX_SECRET, wenn das Secret fehlt, ohne den Wert des Keys zu verraten', () => {
    const error = captureError(() => readImportConfig({ BX_KEY: SECRET_KEY_VALUE }));

    expect(error.message).toContain('BX_SECRET');
    expect(error.message).not.toContain(SECRET_KEY_VALUE);
  });

  it('behandelt leere Pflichtvariablen wie fehlende', () => {
    const error = captureError(() =>
      readImportConfig({ BX_KEY: '', BX_SECRET: SECRET_SECRET_VALUE }),
    );

    expect(error.message).toContain('BX_KEY');
  });

  it('wirft, wenn beide Pflichtvariablen fehlen', () => {
    expect(() => readImportConfig({})).toThrow(/BX_KEY|BX_SECRET/);
  });

  it.each(['abc', '0', '-5', '1.5', '30d'])(
    'lehnt ungültiges IMPORT_FULL_DAYS "%s" ab',
    (rawDays) => {
      const error = captureError(() =>
        readImportConfig({
          BX_KEY: SECRET_KEY_VALUE,
          BX_SECRET: SECRET_SECRET_VALUE,
          IMPORT_FULL_DAYS: rawDays,
        }),
      );

      expect(error.message).toContain('IMPORT_FULL_DAYS');
      expect(error.message).not.toContain(SECRET_KEY_VALUE);
      expect(error.message).not.toContain(SECRET_SECRET_VALUE);
    },
  );
});
