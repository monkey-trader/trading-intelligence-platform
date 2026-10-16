import { afterEach, describe, expect, it } from 'vitest';
import type { FastifyInstance } from 'fastify';
// Import bewusst über den @/-Alias: der Test prüft so auch die Alias-Konfiguration.
import { createServer } from '@/presentation/http/createServer';
import type { OpenPosition } from '@/domain/position/entities/OpenPosition';

/** Stub: /health braucht keine offenen Positionen. */
async function getOpenPositionsStub(): Promise<OpenPosition[]> {
  return [];
}

describe('GET /health', () => {
  let server: FastifyInstance | undefined;

  afterEach(async () => {
    await server?.close();
    server = undefined;
  });

  it('antwortet mit 200 und status "ok"', async () => {
    server = createServer({ getOpenPositions: getOpenPositionsStub });

    const response = await server.inject({ method: 'GET', url: '/health' });

    expect(response.statusCode).toBe(200);
    expect(response.headers['content-type']).toMatch(/^application\/json/);
    expect(response.json()).toEqual({ status: 'ok' });
  });
});
