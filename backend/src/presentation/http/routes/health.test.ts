import { afterEach, describe, expect, it } from 'vitest';
import type { FastifyInstance } from 'fastify';
// Import bewusst über den @/-Alias: der Test prüft so auch die Alias-Konfiguration.
import { createServer } from '@/presentation/http/createServer';
import type { OpenPositionsOverview } from '@/application/position/GetOpenPositionsUseCase';

/** inject() setzt standardmäßig den Host-Header localhost:80. */
const SERVER_SETTINGS = { allowedHosts: ['localhost:80'] } as const;

/** Stub: /health braucht keine offenen Positionen. */
async function getOpenPositionsStub(): Promise<OpenPositionsOverview> {
  return { positions: [], totalUnrealizedProfit: '0', fetchedAt: 0 };
}

describe('GET /health', () => {
  let server: FastifyInstance | undefined;

  afterEach(async () => {
    await server?.close();
    server = undefined;
  });

  it('antwortet mit 200 und status "ok"', async () => {
    server = createServer({ getOpenPositions: getOpenPositionsStub }, SERVER_SETTINGS);

    const response = await server.inject({ method: 'GET', url: '/health' });

    expect(response.statusCode).toBe(200);
    expect(response.headers['content-type']).toMatch(/^application\/json/);
    expect(response.json()).toEqual({ status: 'ok' });
  });

  it('antwortet bei fremdem Host-Header mit 403 forbidden-host', async () => {
    server = createServer({ getOpenPositions: getOpenPositionsStub }, SERVER_SETTINGS);

    const response = await server.inject({
      method: 'GET',
      url: '/health',
      headers: { host: 'evil.example:3000' },
    });

    expect(response.statusCode).toBe(403);
    expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
    expect(response.json()).toEqual({
      type: 'forbidden-host',
      title: expect.any(String),
      status: 403,
    });
  });

  it('weist den Standard-Host von inject ab, wenn nur andere Hosts erlaubt sind', async () => {
    server = createServer(
      { getOpenPositions: getOpenPositionsStub },
      { allowedHosts: ['127.0.0.1:3000'] },
    );

    const response = await server.inject({ method: 'GET', url: '/health' });

    expect(response.statusCode).toBe(403);
  });
});
