export type ServerConfig = {
  host: string;
  port: number;
  /** Erlaubte Host-Header (`host:port`) für den Schutz vor DNS-Rebinding. */
  allowedHosts: string[];
};

type Environment = Readonly<Record<string, string | undefined>>;

const DEFAULT_HOST = '127.0.0.1';
const DEFAULT_PORT = 3000;

/**
 * Liest HOST und PORT aus der Umgebung. Leere Werte gelten als nicht gesetzt.
 * Die API hat in Phase 1 keine Authentifizierung und liefert Kontodaten. Ein Host außerhalb von
 * Loopback ist deshalb nur mit ausdrücklichem ALLOW_REMOTE=true erlaubt.
 *
 * `allowedHosts` enthält immer die Loopback-Namen mit dem Port. Bei ALLOW_REMOTE=true kommen die
 * kommagetrennten Einträge aus ALLOWED_HOSTS dazu; ALLOWED_HOSTS ist dann Pflicht.
 */
export function readServerConfig(environment: Environment): ServerConfig {
  const host = nonEmpty(environment.HOST) ?? DEFAULT_HOST;
  if (!isLoopbackHost(host) && environment.ALLOW_REMOTE !== 'true') {
    throw new Error(
      `HOST "${host}" ist keine Loopback-Adresse. Die API hat noch keine Authentifizierung; ` +
        'nur mit ALLOW_REMOTE=true erlaubt.',
    );
  }
  const port = parsePort(nonEmpty(environment.PORT));
  const loopbackHosts = withBoundHost(
    [`127.0.0.1:${port}`, `localhost:${port}`, `[::1]:${port}`],
    host,
    port,
  );
  if (environment.ALLOW_REMOTE !== 'true') return { host, port, allowedHosts: loopbackHosts };
  return { host, port, allowedHosts: [...loopbackHosts, ...parseRemoteAllowedHosts(environment)] };
}

function parseRemoteAllowedHosts(environment: Environment): string[] {
  const remoteHosts = (environment.ALLOWED_HOSTS ?? '')
    .split(',')
    .map((entry) => entry.trim())
    .filter((entry) => entry !== '');
  const invalidEntry = remoteHosts.find((entry) => !HOST_WITH_PORT_PATTERN.test(entry));
  if (invalidEntry !== undefined) {
    throw new Error(
      `ALLOWED_HOSTS enthält "${invalidEntry}", erwartet wird host:port (z.B. meinrechner.local:3000).`,
    );
  }
  if (remoteHosts.length === 0) {
    throw new Error(
      'Mit ALLOW_REMOTE=true muss ALLOWED_HOSTS gesetzt sein ' +
        '(kommagetrennte Host-Header mit Port, z.B. meinrechner.local:3000).',
    );
  }
  return remoteHosts;
}

/** host:port, IPv6 in eckigen Klammern, Port 1–5 Ziffern. */
const HOST_WITH_PORT_PATTERN = /^(\[[0-9a-fA-F:.]+\]|[A-Za-z0-9.-]+):\d{1,5}$/;

/**
 * Ergänzt den gebundenen Loopback-Host (z.B. 127.0.0.2), damit Aufrufe über genau diese Adresse
 * nicht vom Host-Check abgelehnt werden. IPv6 wird in eckigen Klammern geschrieben.
 */
function withBoundHost(loopbackHosts: string[], host: string, port: number): string[] {
  if (!isLoopbackHost(host)) return loopbackHosts;
  const boundHost = host.includes(':') ? `[${host}]:${port}` : `${host}:${port}`;
  return loopbackHosts.includes(boundHost) ? loopbackHosts : [...loopbackHosts, boundHost];
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
