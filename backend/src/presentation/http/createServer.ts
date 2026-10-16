import Fastify, { type FastifyInstance, type FastifyServerOptions } from 'fastify';
import { registerHealthRoute } from '@/presentation/http/routes/health';

/**
 * Baut die Fastify-Instanz und registriert alle Routen, startet sie aber nicht.
 * Tests nutzen `inject()`, der Start mit `listen()` passiert in `src/main.ts`.
 */
export function createServer(options: FastifyServerOptions = {}): FastifyInstance {
  const server = Fastify(options);
  registerHealthRoute(server);
  return server;
}
