import Fastify, {
  type FastifyInstance,
  type FastifyReply,
  type FastifyServerOptions,
} from 'fastify';
import type { OpenPositionsOverview } from '@/application/position/GetOpenPositionsUseCase';
import { isAllowedHost, registerHostGuard, sendForbiddenHost } from '@/presentation/http/hostGuard';
import { createProblem, PROBLEM_CONTENT_TYPE } from '@/presentation/http/problem';
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
 * Automatische HEAD-Routen sind aus: HEAD ist in docs/openapi.yaml nicht beschrieben und ergibt 404.
 * Fehler vor dem Routing (z.B. ungültige Prozent-Kodierung) laufen über `frameworkErrors`:
 * dort gilt ebenfalls zuerst die Host-Prüfung, danach 400 bad-request.
 * Tests nutzen `inject()`, der Start mit `listen()` passiert in `src/main.ts`.
 */
export function createServer(
  dependencies: ServerDependencies,
  settings: ServerSettings,
  options: FastifyServerOptions = {},
): FastifyInstance {
  const allowedHosts: ReadonlySet<string> = new Set(settings.allowedHosts);
  const server = Fastify({
    ...options,
    exposeHeadRoutes: false,
    frameworkErrors: (error, request, reply) => {
      if (!isAllowedHost(request.headers.host, allowedHosts)) {
        sendForbiddenHost(reply);
        return;
      }
      request.log.info({ err: error }, 'Formal fehlerhafte Anfrage vor dem Routing abgelehnt.');
      sendBadRequest(reply);
    },
  });
  registerHostGuard(server, allowedHosts);
  registerProblemHandlers(server);
  registerHealthRoute(server);
  registerOpenPositionsRoute(server, dependencies.getOpenPositions);
  return server;
}

/**
 * Allgemeine Fehlerantworten laut info.description in docs/openapi.yaml:
 * unbekannte Pfade und Methoden als 404 not-found (auch wenn der Body nicht geparst werden kann),
 * Client-Fehler von Fastify (4xx) als 400 bad-request, unerwartete Fehler (auch
 * Serialisierungsfehler) als 500 internal. Der Body enthält nie Pfad, Meldung oder Stack; die
 * Ursache landet nur im Log. Der 404-Handler läuft nach dem onRequest-Hook, der Host-Guard greift
 * also vorher.
 */
function registerProblemHandlers(server: FastifyInstance): void {
  server.setNotFoundHandler(async (_request, reply) =>
    reply.code(404).type(PROBLEM_CONTENT_TYPE).send(createProblem('not-found')),
  );
  server.setErrorHandler(async (error: unknown, request, reply) => {
    if (request.is404) {
      request.log.info({ err: error }, 'Fehler beim Lesen der Anfrage an einen unbekannten Pfad.');
      return reply.code(404).type(PROBLEM_CONTENT_TYPE).send(createProblem('not-found'));
    }
    if (isClientErrorStatus(statusCodeOf(error))) {
      request.log.warn({ err: error }, 'Formal fehlerhafte Anfrage abgelehnt.');
      return sendBadRequest(reply);
    }
    request.log.error({ err: error }, 'Unerwarteter Fehler bei der Verarbeitung der Anfrage.');
    return reply.code(500).type(PROBLEM_CONTENT_TYPE).send(createProblem('internal'));
  });
}

/** Sendet 400 bad-request als application/problem+json, ohne Pfad oder Fehlermeldung. */
function sendBadRequest(reply: FastifyReply): FastifyReply {
  return reply.code(400).type(PROBLEM_CONTENT_TYPE).send(createProblem('bad-request'));
}

/** Liest `statusCode` aus einem Fehler, wie Fastify ihn bei Client-Fehlern setzt. */
function statusCodeOf(error: unknown): number | undefined {
  if (typeof error !== 'object' || error === null || !('statusCode' in error)) return undefined;
  const { statusCode } = error;
  return typeof statusCode === 'number' ? statusCode : undefined;
}

function isClientErrorStatus(statusCode: number | undefined): boolean {
  return statusCode !== undefined && statusCode >= 400 && statusCode < 500;
}
