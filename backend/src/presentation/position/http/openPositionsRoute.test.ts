import { afterEach, describe, expect, it } from 'vitest';
import type { FastifyInstance } from 'fastify';
import { createServer } from '@/presentation/http/createServer';
import type { OpenPositionsOverview } from '@/application/position/GetOpenPositionsUseCase';
import { OpenPositionsUnavailableError } from '@/domain/position/errors/OpenPositionsUnavailableError';
import type { OpenPositionsUnavailableReason } from '@/domain/position/errors/OpenPositionsUnavailableError';

const OPEN_POSITION_SCHEMA_FIELDS = [
  'positionId',
  'symbol',
  'side',
  'isolated',
  'positionAmount',
  'avgOpenPrice',
  'markPrice',
  'leverage',
  'initialMargin',
  'margin',
  'positionValue',
  'unrealizedProfit',
  'realisedProfit',
  'pnlRatio',
  'liquidationPrice',
  'liquidationDistanceRatio',
  'openTime',
  'updateTime',
];

type OverviewPosition = OpenPositionsOverview['positions'][number];

/** inject() setzt standardmäßig den Host-Header localhost:80. */
const SERVER_SETTINGS = { allowedHosts: ['localhost:80'] } as const;
const FETCHED_AT = 1791800000000;

const suiPosition: OverviewPosition = {
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

const goldPosition: OverviewPosition = {
  positionId: '9100000000000000002',
  symbol: 'NCCOGOLD2USD-USDT',
  side: 'SHORT',
  isolated: true,
  positionAmount: '0.25',
  avgOpenPrice: '4150.00',
  markPrice: '4175.00',
  leverage: 15,
  initialMargin: '69.1667',
  margin: '65.0000',
  positionValue: '1043.75',
  unrealizedProfit: '-6.2500',
  realisedProfit: '-0.5187',
  pnlRatio: '-0.0904',
  liquidationPrice: '4410.5',
  liquidationDistanceRatio: '0.05640719',
  openTime: 1791700000000,
  updateTime: 1791710000000,
};

function overviewOf(
  positions: readonly OverviewPosition[],
  totalUnrealizedProfit: string,
): () => Promise<OpenPositionsOverview> {
  return async () => ({ positions: [...positions], totalUnrealizedProfit, fetchedAt: FETCHED_AT });
}

function failingWith(error: unknown): () => Promise<OpenPositionsOverview> {
  return async () => {
    throw error;
  };
}

describe('GET /positions/open', () => {
  let server: FastifyInstance | undefined;

  afterEach(async () => {
    await server?.close();
    server = undefined;
  });

  it('antwortet mit 200 und der Hülle exakt laut Spec (Positionen in gelieferter Reihenfolge)', async () => {
    server = createServer(
      { getOpenPositions: overviewOf([goldPosition, suiPosition], '18.75') },
      SERVER_SETTINGS,
    );

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.statusCode).toBe(200);
    expect(response.headers['content-type']).toMatch(/^application\/json/);
    expect(response.json()).toEqual({
      positions: [goldPosition, suiPosition],
      totalUnrealizedProfit: '18.75',
      fetchedAt: FETCHED_AT,
    });
  });

  it('liefert liquidationDistanceRatio null ohne Liquidationspreis', async () => {
    const crossPosition: OverviewPosition = {
      ...suiPosition,
      isolated: false,
      liquidationPrice: '0',
      liquidationDistanceRatio: null,
    };
    server = createServer(
      { getOpenPositions: overviewOf([crossPosition], '25.0000') },
      SERVER_SETTINGS,
    );

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.statusCode).toBe(200);
    expect(response.json()).toEqual({
      positions: [crossPosition],
      totalUnrealizedProfit: '25.0000',
      fetchedAt: FETCHED_AT,
    });
  });

  it('antwortet mit 200 und leerer Liste, wenn keine Position offen ist', async () => {
    server = createServer({ getOpenPositions: overviewOf([], '0') }, SERVER_SETTINGS);

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.statusCode).toBe(200);
    expect(response.json()).toEqual({
      positions: [],
      totalUnrealizedProfit: '0',
      fetchedAt: FETCHED_AT,
    });
  });

  it('gibt zusätzliche Felder einer Entity oder der Hülle nicht aus', async () => {
    const positionWithExtras = {
      ...suiPosition,
      riskRate: '0.0071',
      rawPayload: '{"intern":true}',
    };
    const overviewWithExtras = {
      positions: [positionWithExtras],
      totalUnrealizedProfit: '25.0000',
      fetchedAt: FETCHED_AT,
      debugInfo: 'intern-abc',
    };
    server = createServer({ getOpenPositions: async () => overviewWithExtras }, SERVER_SETTINGS);

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.statusCode).toBe(200);
    const body: unknown = response.json();
    expect(Object.keys(body ?? {}).sort()).toEqual([
      'fetchedAt',
      'positions',
      'totalUnrealizedProfit',
    ]);
    const positions: unknown =
      typeof body === 'object' && body !== null && 'positions' in body ? body.positions : undefined;
    expect(Array.isArray(positions)).toBe(true);
    const firstEntry: unknown = Array.isArray(positions) ? positions[0] : undefined;
    expect(Object.keys(firstEntry ?? {}).sort()).toEqual([...OPEN_POSITION_SCHEMA_FIELDS].sort());
    expect(response.body).not.toContain('riskRate');
    expect(response.body).not.toContain('rawPayload');
    expect(response.body).not.toContain('intern-abc');
  });

  it('antwortet bei fremdem Host-Header mit 403 forbidden-host, ohne den Use Case aufzurufen', async () => {
    let useCaseCalls = 0;
    server = createServer(
      {
        getOpenPositions: async () => {
          useCaseCalls += 1;
          return { positions: [], totalUnrealizedProfit: '0', fetchedAt: FETCHED_AT };
        },
      },
      SERVER_SETTINGS,
    );

    const response = await server.inject({
      method: 'GET',
      url: '/positions/open',
      headers: { host: 'evil.example:3000' },
    });

    expect(response.statusCode).toBe(403);
    expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
    expect(response.json()).toEqual({
      type: 'forbidden-host',
      title: expect.any(String),
      status: 403,
    });
    expect(useCaseCalls).toBe(0);
  });

  it.each([
    ['not-configured', 503],
    ['rate-limited', 503],
    ['unreachable', 504],
    ['upstream-error', 502],
    ['invalid-response', 502],
  ] as const)(
    'antwortet bei %s mit %i als application/problem+json',
    async (reason: OpenPositionsUnavailableReason, expectedStatus) => {
      server = createServer(
        { getOpenPositions: failingWith(new OpenPositionsUnavailableError(reason)) },
        SERVER_SETTINGS,
      );

      const response = await server.inject({ method: 'GET', url: '/positions/open' });

      expect(response.statusCode).toBe(expectedStatus);
      expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
      expect(response.json()).toMatchObject({ type: reason, status: expectedStatus });
    },
  );

  it('setzt Retry-After: 30 bei rate-limited', async () => {
    server = createServer(
      { getOpenPositions: failingWith(new OpenPositionsUnavailableError('rate-limited')) },
      SERVER_SETTINGS,
    );

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.statusCode).toBe(503);
    expect(response.headers['retry-after']).toBe('30');
  });

  it('setzt kein Retry-After bei not-configured', async () => {
    server = createServer(
      { getOpenPositions: failingWith(new OpenPositionsUnavailableError('not-configured')) },
      SERVER_SETTINGS,
    );

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.headers['retry-after']).toBeUndefined();
  });

  it('antwortet bei einem unbekannten Fehler mit 500 internal', async () => {
    server = createServer(
      { getOpenPositions: failingWith(new Error('Programmierfehler')) },
      SERVER_SETTINGS,
    );

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.statusCode).toBe(500);
    expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
    expect(response.json()).toMatchObject({ type: 'internal', status: 500 });
    expect(response.body).not.toContain('Programmierfehler');
  });

  it('gibt den Text von cause nicht im Body preis', async () => {
    const cause = new Error('BingX-Detail geheim-xyz');
    server = createServer(
      { getOpenPositions: failingWith(new OpenPositionsUnavailableError('upstream-error', cause)) },
      SERVER_SETTINGS,
    );

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.statusCode).toBe(502);
    expect(response.body).not.toContain('geheim-xyz');
  });
});
