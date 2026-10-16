export type ServerConfig = { host: string; port: number };

type Environment = Readonly<Record<string, string | undefined>>;

const DEFAULT_HOST = '127.0.0.1';
const DEFAULT_PORT = 3000;

/**
 * Liest HOST und PORT aus der Umgebung. Leere Werte gelten als nicht gesetzt.
 * Die API hat in Phase 1 keine Authentifizierung und liefert Kontodaten. Ein Host außerhalb von
 * Loopback ist deshalb nur mit ausdrücklichem ALLOW_REMOTE=true erlaubt.
 */
export function readServerConfig(environment: Environment): ServerConfig {
  const host = nonEmpty(environment.HOST) ?? DEFAULT_HOST;
  if (!isLoopbackHost(host) && environment.ALLOW_REMOTE !== 'true') {
    throw new Error(
      `HOST "${host}" ist keine Loopback-Adresse. Die API hat noch keine Authentifizierung; ` +
        'nur mit ALLOW_REMOTE=true erlaubt.',
    );
  }
  return { host, port: parsePort(nonEmpty(environment.PORT)) };
}

function isLoopbackHost(host: string): boolean {
  return host === 'localhost' || host === '::1' || /^127(\.\d{1,3}){3}$/.test(host);
}

function nonEmpty(value: string | undefined): string | undefined {
  return value === undefined || value.trim() === '' ? undefined : value.trim();
}

function parsePort(rawPort: string | undefined): number {
  if (rawPort === undefined) return DEFAULT_PORT;
  const port = Number(rawPort);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error(`Ungültiger PORT: "${rawPort}"`);
  }
  return port;
}
