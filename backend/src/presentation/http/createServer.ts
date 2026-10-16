import Fastify, { type FastifyInstance, type FastifyServerOptions } from 'fastify';
import type { OpenPositionsOverview } from '@/application/position/GetOpenPositionsUseCase';
import { registerHostGuard } from '@/presentation/http/hostGuard';
import { registerHealthRoute } from '@/presentation/http/routes/health';
import { registerOpenPositionsRoute } from '@/presentation/position/http/openPositionsRoute';

/** Use Cases, die der Server braucht. Verdrahtet werden sie in `src/main.ts`. */
export type ServerDependencies = Readonly<{
  getOpenPositions: () => Promise<OpenPositionsOverview>;
}>;

/** Einstellungen des Servers. `allowedHosts` sind die erlaubten Host-Header (`host:port`). */
export type ServerSettings = Readonly<{
  allowedHosts: readonly string[];
}>;

/**
 * Baut die Fastify-Instanz und registriert alle Routen, startet sie aber nicht.
 * Der Host-Guard kommt zuerst, damit er für alle Pfade gilt (inkl. /health und 404).
 * Tests nutzen `inject()`, der Start mit `listen()` passiert in `src/main.ts`.
 */
export function createServer(
  dependencies: ServerDependencies,
  settings: ServerSettings,
  options: FastifyServerOptions = {},
): FastifyInstance {
  const server = Fastify(options);
  registerHostGuard(server, new Set(settings.allowedHosts));
  registerHealthRoute(server);
  registerOpenPositionsRoute(server, dependencies.getOpenPositions);
  return server;
}
