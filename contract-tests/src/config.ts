import { InvalidBaseUrlError, InvalidExpectConfiguredError, MissingBaseUrlError } from '@/errors';

/** Konfiguration des Vertragstests, ausschließlich aus Umgebungsvariablen. */
export type ContractConfig = Readonly<{
  /** Basis-URL der laufenden API, nur Origin (z.B. http://127.0.0.1:3000). */
  baseUrl: URL;
  /** true: BingX-Keys sind gesetzt, GET /positions/open muss 200 liefern; sonst 503 not-configured. */
  expectConfigured: boolean;
}>;

export type ContractEnvironment = Readonly<Record<string, string | undefined>>;

/** Liest CONTRACT_BASE_URL (Pflicht) und CONTRACT_EXPECT_CONFIGURED (optional). */
export function readContractConfig(environment: ContractEnvironment): ContractConfig {
  return {
    baseUrl: parseBaseUrl(environment['CONTRACT_BASE_URL']),
    expectConfigured: parseExpectConfigured(environment['CONTRACT_EXPECT_CONFIGURED']),
  };
}

function parseBaseUrl(rawBaseUrl: string | undefined): URL {
  if (rawBaseUrl === undefined || rawBaseUrl.trim() === '') throw new MissingBaseUrlError();
  let baseUrl: URL;
  try {
    baseUrl = new URL(rawBaseUrl.trim());
  } catch {
    throw new InvalidBaseUrlError(rawBaseUrl, 'keine gültige URL');
  }
  if (baseUrl.protocol !== 'http:' && baseUrl.protocol !== 'https:') {
    throw new InvalidBaseUrlError(rawBaseUrl, 'nur http oder https');
  }
  if (baseUrl.pathname !== '/' || baseUrl.search !== '' || baseUrl.hash !== '') {
    throw new InvalidBaseUrlError(rawBaseUrl, 'nur Schema, Host und Port, kein Pfad');
  }
  if (baseUrl.username !== '' || baseUrl.password !== '') {
    throw new InvalidBaseUrlError(rawBaseUrl, 'keine Zugangsdaten in der URL');
  }
  return baseUrl;
}

function parseExpectConfigured(rawValue: string | undefined): boolean {
  if (rawValue === undefined || rawValue === '') return false;
  const normalizedValue = rawValue.trim().toLowerCase();
  if (normalizedValue === 'true') return true;
  if (normalizedValue === 'false') return false;
  throw new InvalidExpectConfiguredError(rawValue);
}

/** Port der Basis-URL; ohne expliziten Port der Standard-Port des Schemas. */
export function effectivePort(baseUrl: URL): string {
  if (baseUrl.port !== '') return baseUrl.port;
  return baseUrl.protocol === 'https:' ? '443' : '80';
}

/** Host-Header `<hostname>:<port>` mit dem Port der Basis-URL, z.B. für LOCALHOST oder fremde Hosts. */
export function hostHeaderFor(baseUrl: URL, hostname: string): string {
  return `${hostname}:${effectivePort(baseUrl)}`;
}
