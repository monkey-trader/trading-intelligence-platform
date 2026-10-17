import { afterEach, describe, expect, it } from 'vitest';
import type { FastifyInstance } from 'fastify';
import { createServer } from '@/presentation/http/createServer';
import type { OpenPositionsOverview } from '@/application/position/GetOpenPositionsUseCase';

/**
 * Allgemeine Regeln aus info.description in docs/openapi.yaml, die für alle Pfade gelten:
 * Host-Prüfung vor dem Routing (auch bei ungültigen Pfaden), 404 not-found für Unbekanntes
 * (auch HEAD, OPTIONS, DELETE, unabhängig vom Body), 400 bad-request für formal fehlerhafte
 * Anfragen und 500 internal ohne interne Details für unerwartete Fehler.
 */

/** inject() setzt standardmäßig den Host-Header localhost:80. */
const SERVER_SETTINGS = { allowedHosts: ['localhost:80'] } as const;

type OverviewPosition = OpenPositionsOverview['positions'][number];

const validPosition: OverviewPosition = {
  positionId: '9100000000000000001',
  symbol: 'SUI-USDT',
  side: 'LONG',
  isolated: true,
  positionAmount: '500',
  avgOpenPrice: '0.9600',
  markPrice: '1.0100',
  leverage: 10,
  initialMargin: '48.0000',
  margin: '60.5000',
  positionValue: '505',
  unrealizedProfit: '25.0000',
  realisedProfit: '-0.2400',
  pnlRatio: '0.5208',
  liquidationPrice: '0.865',
  liquidationDistanceRatio: '0.14356436',
  openTime: 1791500000000,
  updateTime: 1791600000000,
};

async function emptyOverview(): Promise<OpenPositionsOverview> {
  return { positions: [], totalUnrealizedProfit: '0', fetchedAt: 0 };
}

/**
 * Liefert eine Position ohne Pflichtfeld `symbol`. Die Rückgabe läuft ungeprüft durch den
 * try/catch der Route und scheitert erst bei der Serialisierung gegen das Response-Schema,
 * also in Fastify selbst. Der Cast ist nötig, weil der Typ das fehlende Feld zu Recht verbietet.
 */
async function overviewViolatingResponseSchema(): Promise<OpenPositionsOverview> {
  const brokenPosition = { ...validPosition, symbol: undefined } as unknown as OverviewPosition;
  return { positions: [brokenPosition], totalUnrealizedProfit: '25.0000', fetchedAt: 0 };
}

const SECRET_ERROR_MESSAGE = 'geheime interne Meldung /var/lib/journal.db';

async function throwingOverview(): Promise<OpenPositionsOverview> {
  throw new Error(SECRET_ERROR_MESSAGE);
}

describe('createServer – allgemeine Regeln', () => {
  let server: FastifyInstance | undefined;

  afterEach(async () => {
    await server?.close();
    server = undefined;
  });

  function createDefaultServer(
    getOpenPositions: () => Promise<OpenPositionsOverview> = emptyOverview,
  ): FastifyInstance {
    server = createServer({ getOpenPositions }, SERVER_SETTINGS);
    return server;
  }

  describe('404 not-found', () => {
    it('antwortet auf einen unbekannten Pfad mit 404 not-found als application/problem+json', async () => {
      const response = await createDefaultServer().inject({ method: 'GET', url: '/nope' });

      expect(response.statusCode).toBe(404);
      expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
      expect(response.json()).toEqual({
        type: 'not-found',
        title: expect.any(String),
        status: 404,
      });
    });

    it('liefert einen nicht leeren Titel und keine weiteren Felder (kein message/error/statusCode)', async () => {
      const response = await createDefaultServer().inject({ method: 'GET', url: '/nope' });

      const body: unknown = response.json();
      expect(body).toBeTypeOf('object');
      expect(Object.keys(body as object).sort()).toEqual(['status', 'title', 'type']);
      const { title } = body as { title: unknown };
      expect(typeof title).toBe('string');
      expect((title as string).trim().length).toBeGreaterThan(0);
    });

    it('gibt den angefragten Pfad nicht im Body zurück', async () => {
      const response = await createDefaultServer().inject({
        method: 'GET',
        url: '/geheimer-pfad-xyz',
      });

      expect(response.statusCode).toBe(404);
      expect(response.body).not.toContain('geheimer-pfad-xyz');
    });

    it('antwortet auf eine nicht beschriebene Methode (POST /health) mit 404 not-found', async () => {
      const response = await createDefaultServer().inject({ method: 'POST', url: '/health' });

      expect(response.statusCode).toBe(404);
      expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
      expect(response.json()).toEqual({
        type: 'not-found',
        title: expect.any(String),
        status: 404,
      });
    });

    it('antwortet auf HEAD /health mit 404 not-found (keine automatischen HEAD-Routen)', async () => {
      const response = await createDefaultServer().inject({ method: 'HEAD', url: '/health' });

      expect(response.statusCode).toBe(404);
      expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
    });

    it('antwortet auf HEAD /positions/open mit 404, ohne den Use Case aufzurufen', async () => {
      let useCaseCalls = 0;
      const countingOverview = async (): Promise<OpenPositionsOverview> => {
        useCaseCalls += 1;
        return emptyOverview();
      };

      const response = await createDefaultServer(countingOverview).inject({
        method: 'HEAD',
        url: '/positions/open',
      });

      expect(response.statusCode).toBe(404);
      expect(useCaseCalls).toBe(0);
    });
  });

  describe('Host-Prüfung vor dem Routing', () => {
    it('antwortet bei fremdem Host auf einem unbekannten Pfad mit 403 forbidden-host statt 404', async () => {
      const response = await createDefaultServer().inject({
        method: 'GET',
        url: '/nope',
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

    it('antwortet bei fremdem Host auf HEAD /health mit 403 forbidden-host', async () => {
      const response = await createDefaultServer().inject({
        method: 'HEAD',
        url: '/health',
        headers: { host: 'evil.example:3000' },
      });

      expect(response.statusCode).toBe(403);
    });
  });

  describe('404 not-found unabhängig von Methode und Body', () => {
    /** Gültiges JSON, größer als Fastifys Standard-bodyLimit von 1 MiB. */
    const OVERSIZED_JSON_BODY = JSON.stringify('x'.repeat(1024 * 1024 + 16));

    it.each([
      ['POST', '/nope'],
      ['POST', '/health'],
    ] as const)(
      'antwortet auf %s %s mit kaputtem JSON-Body mit 404 not-found statt 500',
      async (method, url) => {
        const response = await createDefaultServer().inject({
          method,
          url,
          headers: { 'content-type': 'application/json' },
          payload: '{',
        });

        expect(response.statusCode).toBe(404);
        expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
        expect(response.json()).toEqual({
          type: 'not-found',
          title: expect.any(String),
          status: 404,
        });
      },
    );

    it('antwortet auf POST /nope mit einem Body über 1 MiB mit 404 not-found', async () => {
      const response = await createDefaultServer().inject({
        method: 'POST',
        url: '/nope',
        headers: { 'content-type': 'application/json' },
        payload: OVERSIZED_JSON_BODY,
      });

      expect(response.statusCode).toBe(404);
      expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
      expect(response.json()).toEqual({
        type: 'not-found',
        title: expect.any(String),
        status: 404,
      });
    });

    it.each([
      ['OPTIONS', '/health'],
      ['DELETE', '/positions/open'],
    ] as const)('antwortet auf %s %s mit 404 not-found', async (method, url) => {
      const response = await createDefaultServer().inject({ method, url });

      expect(response.statusCode).toBe(404);
      expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
      expect(response.json()).toEqual({
        type: 'not-found',
        title: expect.any(String),
        status: 404,
      });
    });

    it('ruft bei DELETE /positions/open den Use Case nicht auf', async () => {
      let useCaseCalls = 0;
      const countingOverview = async (): Promise<OpenPositionsOverview> => {
        useCaseCalls += 1;
        return emptyOverview();
      };

      await createDefaultServer(countingOverview).inject({
        method: 'DELETE',
        url: '/positions/open',
      });

      expect(useCaseCalls).toBe(0);
    });
  });

  describe('400 bad-request bei ungültiger Prozent-Kodierung', () => {
    /** inject() reicht die URL roh durch; Fastify meldet sonst FST_ERR_BAD_URL. */
    const MALFORMED_PATH = '/%zz';

    it('antwortet auf GET /%zz mit erlaubtem Host mit 400 bad-request als application/problem+json', async () => {
      const response = await createDefaultServer().inject({ method: 'GET', url: MALFORMED_PATH });

      expect(response.statusCode).toBe(400);
      expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
      expect(response.json()).toEqual({
        type: 'bad-request',
        title: expect.any(String),
        status: 400,
      });
    });

    it('gibt weder den Pfad noch interne Felder (code/message/error/statusCode) zurück', async () => {
      const response = await createDefaultServer().inject({ method: 'GET', url: MALFORMED_PATH });

      expect(response.body).not.toContain('%zz');
      expect(response.body).not.toContain('FST_ERR');
      const body: unknown = response.json();
      expect(Object.keys(body as object).sort()).toEqual(['status', 'title', 'type']);
      const { title } = body as { title: unknown };
      expect(typeof title).toBe('string');
      expect((title as string).trim().length).toBeGreaterThan(0);
    });

    it('antwortet auf GET /%zz mit fremdem Host mit 403 forbidden-host (Host-Prüfung zuerst)', async () => {
      const response = await createDefaultServer().inject({
        method: 'GET',
        url: MALFORMED_PATH,
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
  });

  describe('500 internal', () => {
    it('antwortet bei einem Serialisierungsfehler mit 500 internal als application/problem+json', async () => {
      const response = await createDefaultServer(overviewViolatingResponseSchema).inject({
        method: 'GET',
        url: '/positions/open',
      });

      expect(response.statusCode).toBe(500);
      expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
      expect(response.json()).toEqual({
        type: 'internal',
        title: expect.any(String),
        status: 500,
      });
    });

    it('verrät beim Serialisierungsfehler keine internen Details (keine Meldung, kein Feldname, kein Stack)', async () => {
      const response = await createDefaultServer(overviewViolatingResponseSchema).inject({
        method: 'GET',
        url: '/positions/open',
      });

      expect(Object.keys(response.json() as object).sort()).toEqual(['status', 'title', 'type']);
      expect(response.body).not.toContain('symbol');
      expect(response.body).not.toContain('required');
      expect(response.body).not.toContain('statusCode');
      expect(response.body).not.toMatch(/\bat .+:\d+:\d+/);
    });

    it('antwortet bei einem normalen Error aus dem Use Case mit 500 internal ohne Fehlermeldung', async () => {
      const response = await createDefaultServer(throwingOverview).inject({
        method: 'GET',
        url: '/positions/open',
      });

      expect(response.statusCode).toBe(500);
      expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
      expect(response.json()).toEqual({
        type: 'internal',
        title: expect.any(String),
        status: 500,
      });
      expect(response.body).not.toContain(SECRET_ERROR_MESSAGE);
      expect(response.headers['retry-after']).toBeUndefined();
    });
  });
});
