import { afterEach, describe, expect, it } from 'vitest';
import type { FastifyInstance } from 'fastify';
import { createServer } from '@/presentation/http/createServer';
import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
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
  'openTime',
  'updateTime',
];

const suiPosition: OpenPosition = {
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
  openTime: 1791500000000,
  updateTime: 1791600000000,
};

const goldPosition: OpenPosition = {
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
  openTime: 1791700000000,
  updateTime: 1791710000000,
};

function failingWith(error: unknown): () => Promise<OpenPosition[]> {
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

  it('antwortet mit 200 und den DTOs in der gelieferten Reihenfolge', async () => {
    server = createServer({ getOpenPositions: async () => [goldPosition, suiPosition] });

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.statusCode).toBe(200);
    expect(response.headers['content-type']).toMatch(/^application\/json/);
    expect(response.json()).toEqual([goldPosition, suiPosition]);
  });

  it('antwortet mit 200 und [] , wenn keine Position offen ist', async () => {
    server = createServer({ getOpenPositions: async () => [] });

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.statusCode).toBe(200);
    expect(response.json()).toEqual([]);
  });

  it('gibt zusätzliche Felder einer Entity nicht aus', async () => {
    const positionWithExtras = {
      ...suiPosition,
      riskRate: '0.0071',
      rawPayload: '{"intern":true}',
    };
    server = createServer({ getOpenPositions: async () => [positionWithExtras] });

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.statusCode).toBe(200);
    const body: unknown = response.json();
    expect(Array.isArray(body)).toBe(true);
    const firstEntry: unknown = Array.isArray(body) ? body[0] : undefined;
    expect(Object.keys(firstEntry ?? {}).sort()).toEqual([...OPEN_POSITION_SCHEMA_FIELDS].sort());
    expect(response.body).not.toContain('riskRate');
    expect(response.body).not.toContain('rawPayload');
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
      server = createServer({
        getOpenPositions: failingWith(new OpenPositionsUnavailableError(reason)),
      });

      const response = await server.inject({ method: 'GET', url: '/positions/open' });

      expect(response.statusCode).toBe(expectedStatus);
      expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
      expect(response.json()).toMatchObject({ type: reason, status: expectedStatus });
    },
  );

  it('setzt Retry-After: 30 bei rate-limited', async () => {
    server = createServer({
      getOpenPositions: failingWith(new OpenPositionsUnavailableError('rate-limited')),
    });

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.statusCode).toBe(503);
    expect(response.headers['retry-after']).toBe('30');
  });

  it('setzt kein Retry-After bei not-configured', async () => {
    server = createServer({
      getOpenPositions: failingWith(new OpenPositionsUnavailableError('not-configured')),
    });

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.headers['retry-after']).toBeUndefined();
  });

  it('antwortet bei einem unbekannten Fehler mit 500 internal', async () => {
    server = createServer({ getOpenPositions: failingWith(new Error('Programmierfehler')) });

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.statusCode).toBe(500);
    expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
    expect(response.json()).toMatchObject({ type: 'internal', status: 500 });
    expect(response.body).not.toContain('Programmierfehler');
  });

  it('gibt den Text von cause nicht im Body preis', async () => {
    const cause = new Error('BingX-Detail geheim-xyz');
    server = createServer({
      getOpenPositions: failingWith(new OpenPositionsUnavailableError('upstream-error', cause)),
    });

    const response = await server.inject({ method: 'GET', url: '/positions/open' });

    expect(response.statusCode).toBe(502);
    expect(response.body).not.toContain('geheim-xyz');
  });
});
