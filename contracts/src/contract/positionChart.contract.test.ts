import { beforeAll, describe, expect, it } from 'vitest';
import { expectDocumentedResponse, expectProblem } from '@/contract/assertions';
import { createContractContext } from '@/contract/contractContext';
import { OPERATION_TEST_NAMES } from '@/contract/coverage';
import { hostHeaderFor } from '@/config';
import { isJsonObject } from '@/openapi';

const { config, validator, send } = createContractContext();

/** 19-stellig, gehört keiner echten Position (BingX-IDs beginnen hier mit 9 nur in Fixtures). */
const UNKNOWN_POSITION_ID = '9199999999999999999';

function chartPath(positionId: string, query = ''): string {
  return `/positions/${positionId}/chart${query}`;
}

function firstOpenPositionId(body: unknown): string | undefined {
  if (!isJsonObject(body) || !Array.isArray(body['positions'])) return undefined;
  const first: unknown = body['positions'][0];
  return isJsonObject(first) && typeof first['positionId'] === 'string'
    ? first['positionId']
    : undefined;
}

describe(OPERATION_TEST_NAMES.getPositionChart, () => {
  it('antwortet mit einem Status, Content-Type und Body laut Spec', async () => {
    const response = await send({ method: 'GET', path: chartPath(UNKNOWN_POSITION_ID) });

    expectDocumentedResponse(validator, 'getPositionChart', response);
  });

  // Die Parameter werden vor jedem BingX-Aufruf geprüft, also auch ohne Keys.
  describe.each([
    ['interval=2h', '?interval=2h'],
    ['interval=1m (BingX kennt es, die API nicht)', '?interval=1m'],
    ['interval=1H (Großschreibung)', '?interval=1H'],
    ['limit=0', '?limit=0'],
    ['limit=1001', '?limit=1001'],
    ['limit=-1', '?limit=-1'],
    ['limit=abc', '?limit=abc'],
  ])('ungültiger Parameter %s', (_label, query) => {
    it('liefert 400 bad-request', async () => {
      const response = await send({ method: 'GET', path: chartPath(UNKNOWN_POSITION_ID, query) });

      expectDocumentedResponse(validator, 'getPositionChart', response);
      expectProblem(validator, response, 400, 'bad-request');
    });
  });

  it('liefert 400 bad-request bei einer positionId ohne Ziffern', async () => {
    const response = await send({ method: 'GET', path: chartPath('abc') });

    expectDocumentedResponse(validator, 'getPositionChart', response);
    expectProblem(validator, response, 400, 'bad-request');
  });

  if (config.expectConfigured) {
    it('liefert 404 not-found bei unbekannter positionId', async () => {
      const response = await send({ method: 'GET', path: chartPath(UNKNOWN_POSITION_ID) });

      expectDocumentedResponse(validator, 'getPositionChart', response);
      expectProblem(validator, response, 404, 'not-found');
    });

    describe('mit einer offenen Position des Kontos', () => {
      let positionId: string | undefined;

      beforeAll(async () => {
        const response = await send({ method: 'GET', path: '/positions/open' });
        positionId = firstOpenPositionId(
          expectDocumentedResponse(validator, 'getOpenPositions', response),
        );
      });

      it('liefert 200 und einen PositionChart mit Defaults', async (testContext) => {
        if (positionId === undefined) return testContext.skip(); // hängt vom Live-Konto ab
        const response = await send({ method: 'GET', path: chartPath(positionId) });

        expect(response.status).toBe(200);
        const body = expectDocumentedResponse(validator, 'getPositionChart', response);
        expect(validator.validateSchema('PositionChart', body).errors).toEqual([]);
        expect(isJsonObject(body) ? body['interval'] : undefined).toBe('15m');
        expect(isJsonObject(body) ? body['positionId'] : undefined).toBe(positionId);
      });

      it('liefert die Kerzen aufsteigend und höchstens limit Stück', async (testContext) => {
        if (positionId === undefined) return testContext.skip();
        const response = await send({
          method: 'GET',
          path: chartPath(positionId, '?interval=5m&limit=10'),
        });

        const body = expectDocumentedResponse(validator, 'getPositionChart', response);
        const candles = isJsonObject(body) && Array.isArray(body['candles']) ? body['candles'] : [];
        const times = candles.map((candle: unknown) =>
          isJsonObject(candle) && typeof candle['time'] === 'number' ? candle['time'] : NaN,
        );
        expect(times.length).toBeLessThanOrEqual(10);
        expect(times).toEqual([...new Set(times)].sort((left, right) => left - right));
      });

      it('liefert genau einen ENTRY-Marker (Stufe 1)', async (testContext) => {
        if (positionId === undefined) return testContext.skip();
        const response = await send({ method: 'GET', path: chartPath(positionId) });

        const body = expectDocumentedResponse(validator, 'getPositionChart', response);
        const markers = isJsonObject(body) && Array.isArray(body['markers']) ? body['markers'] : [];
        expect(markers).toHaveLength(1);
        expect(isJsonObject(markers[0]) ? markers[0]['kind'] : undefined).toBe('ENTRY');
      });
    });
  } else {
    it('liefert ohne Keys 503 not-configured', async () => {
      const response = await send({ method: 'GET', path: chartPath(UNKNOWN_POSITION_ID) });

      expectDocumentedResponse(validator, 'getPositionChart', response);
      expectProblem(validator, response, 503, 'not-configured');
    });
  }

  it('antwortet bei fremdem Host mit 403 forbidden-host laut Spec', async () => {
    const response = await send({
      method: 'GET',
      path: chartPath(UNKNOWN_POSITION_ID),
      headers: { host: hostHeaderFor(config.baseUrl, 'evil.example') },
    });

    expectDocumentedResponse(validator, 'getPositionChart', response);
    expectProblem(validator, response, 403, 'forbidden-host');
  });
});
