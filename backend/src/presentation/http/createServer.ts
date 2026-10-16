import Fastify, { type FastifyInstance, type FastifyServerOptions } from 'fastify';
import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import { registerHealthRoute } from '@/presentation/http/routes/health';
import { registerOpenPositionsRoute } from '@/presentation/position/http/openPositionsRoute';

/** Use Cases, die der Server braucht. Verdrahtet werden sie in `src/main.ts`. */
export type ServerDependencies = Readonly<{
  getOpenPositions: () => Promise<OpenPosition[]>;
}>;

/**
 * Baut die Fastify-Instanz und registriert alle Routen, startet sie aber nicht.
 * Tests nutzen `inject()`, der Start mit `listen()` passiert in `src/main.ts`.
 */
export function createServer(
  dependencies: ServerDependencies,
  options: FastifyServerOptions = {},
): FastifyInstance {
  const server = Fastify(options);
  registerHealthRoute(server);
  registerOpenPositionsRoute(server, dependencies.getOpenPositions);
  return server;
}
