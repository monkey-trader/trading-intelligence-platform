import { describe, expect, it } from 'vitest';
import { effectivePort, hostHeaderFor, readContractConfig } from '@/config';
import { InvalidBaseUrlError, InvalidExpectConfiguredError, MissingBaseUrlError } from '@/errors';

describe('readContractConfig', () => {
  it('liest die Basis-URL und erwartet ohne CONTRACT_EXPECT_CONFIGURED einen unkonfigurierten Server', () => {
    const config = readContractConfig({ CONTRACT_BASE_URL: 'http://127.0.0.1:3999' });

    expect(config.baseUrl.origin).toBe('http://127.0.0.1:3999');
    expect(config.expectConfigured).toBe(false);
  });

  it.each(['true', 'TRUE', ' true '])('setzt expectConfigured bei "%s"', (rawValue) => {
    const config = readContractConfig({
      CONTRACT_BASE_URL: 'http://127.0.0.1:3999',
      CONTRACT_EXPECT_CONFIGURED: rawValue,
    });

    expect(config.expectConfigured).toBe(true);
  });

  it.each(['false', ''])('lässt expectConfigured bei "%s" aus', (rawValue) => {
    const config = readContractConfig({
      CONTRACT_BASE_URL: 'http://127.0.0.1:3999',
      CONTRACT_EXPECT_CONFIGURED: rawValue,
    });

    expect(config.expectConfigured).toBe(false);
  });

  it('akzeptiert einen abschließenden Schrägstrich', () => {
    const config = readContractConfig({ CONTRACT_BASE_URL: 'http://localhost:3000/' });

    expect(config.baseUrl.origin).toBe('http://localhost:3000');
  });

  it.each([undefined, '', '   '])(
    'wirft MissingBaseUrlError, wenn CONTRACT_BASE_URL fehlt (%j)',
    (rawBaseUrl) => {
      const action = () => readContractConfig({ CONTRACT_BASE_URL: rawBaseUrl });

      expect(action).toThrow(MissingBaseUrlError);
      expect(action).toThrow(/CONTRACT_BASE_URL/);
    },
  );

  it.each([
    'keine-url',
    'ftp://127.0.0.1:3000',
    'http://127.0.0.1:3000/api',
    'http://127.0.0.1:3000/?x=1',
    'http://nutzer:passwort@127.0.0.1:3000',
  ])('wirft InvalidBaseUrlError für "%s"', (rawBaseUrl) => {
    expect(() => readContractConfig({ CONTRACT_BASE_URL: rawBaseUrl })).toThrow(
      InvalidBaseUrlError,
    );
  });

  it('wirft InvalidExpectConfiguredError für einen unbekannten Wert', () => {
    let caughtError: unknown;
    try {
      readContractConfig({
        CONTRACT_BASE_URL: 'http://127.0.0.1:3000',
        CONTRACT_EXPECT_CONFIGURED: 'ja',
      });
    } catch (error: unknown) {
      caughtError = error;
    }

    expect(caughtError).toBeInstanceOf(InvalidExpectConfiguredError);
  });
});

describe('effectivePort / hostHeaderFor', () => {
  it('nimmt den expliziten Port der Basis-URL', () => {
    const baseUrl = new URL('http://127.0.0.1:3999');

    expect(effectivePort(baseUrl)).toBe('3999');
    expect(hostHeaderFor(baseUrl, 'LOCALHOST')).toBe('LOCALHOST:3999');
  });

  it.each([
    ['http://localhost', '80'],
    ['https://localhost', '443'],
  ])('nimmt ohne Port den Standard-Port (%s → %s)', (rawBaseUrl, expectedPort) => {
    expect(effectivePort(new URL(rawBaseUrl))).toBe(expectedPort);
  });
});
