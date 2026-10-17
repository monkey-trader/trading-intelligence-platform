import type { FastifyInstance, FastifyReply } from 'fastify';
import { createProblem, PROBLEM_CONTENT_TYPE } from '@/presentation/http/problem';

/**
 * Prüft den rohen Host-Header gegen die erlaubten Einträge (`host:port`), ohne Groß-/Kleinschreibung.
 * Ein fehlender Header ist nie erlaubt.
 */
export function isAllowedHost(
  hostHeader: string | undefined,
  allowedHosts: ReadonlySet<string>,
): boolean {
  if (hostHeader === undefined) return false;
  const normalizedHost = hostHeader.toLowerCase();
  for (const allowedHost of allowedHosts) {
    if (allowedHost.toLowerCase() === normalizedHost) return true;
  }
  return false;
}

/**
 * Schutz vor DNS-Rebinding: weist jede Anfrage mit fremdem Host-Header mit 403 forbidden-host ab,
 * bevor Routing oder Handler laufen. Der abgelehnte Host landet nicht im Body.
 */
export function registerHostGuard(
  server: FastifyInstance,
  allowedHosts: ReadonlySet<string>,
): void {
  server.addHook('onRequest', async (request, reply) => {
    if (isAllowedHost(request.headers.host, allowedHosts)) return;
    return sendForbiddenHost(reply);
  });
}

/**
 * Sendet 403 forbidden-host als application/problem+json. Gemeinsam genutzt vom onRequest-Hook
 * und von Fastifys `frameworkErrors` (Fehler vor dem Routing, z.B. ungültige Prozent-Kodierung).
 */
export function sendForbiddenHost(reply: FastifyReply): FastifyReply {
  return reply.code(403).type(PROBLEM_CONTENT_TYPE).send(createProblem('forbidden-host'));
}
