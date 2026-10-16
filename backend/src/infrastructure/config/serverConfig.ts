export type ServerConfig = { host: string; port: number };

type Environment = Readonly<Record<string, string | undefined>>;

const DEFAULT_HOST = '127.0.0.1';
const DEFAULT_PORT = 3000;

/** Liest HOST und PORT aus der Umgebung. Leere Werte gelten als nicht gesetzt. */
export function readServerConfig(environment: Environment): ServerConfig {
  return {
    host: nonEmpty(environment.HOST) ?? DEFAULT_HOST,
    port: parsePort(nonEmpty(environment.PORT)),
  };
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
