import { describe, expect, it } from 'vitest';
import { expectDocumentedResponse, expectProblem } from '@/contract/assertions';
import { createContractContext } from '@/contract/contractContext';
import { OPERATION_TEST_NAMES } from '@/contract/coverage';
import { isJsonObject } from '@/openapi';

const { config, validator, send } = createContractContext();

function closedPositionsOf(body: unknown): unknown[] {
  return isJsonObject(body) && Array.isArray(body['positions']) ? body['positions'] : [];
}

function closeTimesOf(body: unknown): number[] {
  return closedPositionsOf(body).map((position) =>
    isJsonObject(position) && typeof position['closeTime'] === 'number'
      ? position['closeTime']
      : NaN,
  );
}

describe(OPERATION_TEST_NAMES.getClosedPositions, () => {
  if (config.expectConfigured) {
    it('liefert 200 und eine ClosedPositionsResponse laut Spec (auch leer als [])', async () => {
      const response = await send({ method: 'GET', path: '/positions/closed' });

      expect(response.status).toBe(200);
      const body = expectDocumentedResponse(validator, 'getClosedPositions', response);
      expect(validator.validateSchema('ClosedPositionsResponse', body).errors).toEqual([]);
      expect(Array.isArray(isJsonObject(body) ? body['positions'] : undefined)).toBe(true);
    });

    it('jede Position hat alle Pflichtfelder, Nullfelder als null und leverage als Zahl', async () => {
      const response = await send({ method: 'GET', path: '/positions/closed?limit=500' });

      const body = expectDocumentedResponse(validator, 'getClosedPositions', response);
      for (const position of closedPositionsOf(body)) {
        expect(validator.validateSchema('ClosedPosition', position).errors).toEqual([]);
        expect(isJsonObject(position) ? typeof position['leverage'] : '').toBe('number');
      }
    });

    it('sortiert nach closeTime absteigend', async () => {
      const response = await send({ method: 'GET', path: '/positions/closed?limit=500' });

      const times = closeTimesOf(
        expectDocumentedResponse(validator, 'getClosedPositions', response),
      );
      expect(times).toEqual([...times].sort((first, second) => second - first));
    });

    it('begrenzt die Anzahl auf limit', async () => {
      const response = await send({ method: 'GET', path: '/positions/closed?limit=1' });

      const body = expectDocumentedResponse(validator, 'getClosedPositions', response);
      expect(closedPositionsOf(body).length).toBeLessThanOrEqual(1);
    });

    it('liefert eine leere Liste für einen Zeitraum ohne Positionen', async () => {
      const response = await send({ method: 'GET', path: '/positions/closed?from=0&to=0' });

      const body = expectDocumentedResponse(validator, 'getClosedPositions', response);
      expect(closedPositionsOf(body)).toEqual([]);
    });
  } else {
    it('liefert ohne Datenbank 503 not-configured', async () => {
      const response = await send({ method: 'GET', path: '/positions/closed' });

      expectDocumentedResponse(validator, 'getClosedPositions', response);
      expectProblem(validator, response, 503, 'not-configured');
    });
  }

  it('liefert 400 bad-request bei limit=501', async () => {
    const response = await send({ method: 'GET', path: '/positions/closed?limit=501' });

    expectDocumentedResponse(validator, 'getClosedPositions', response);
    expectProblem(validator, response, 400, 'bad-request');
  });

  it('liefert 400 bad-request bei limit=0', async () => {
    const response = await send({ method: 'GET', path: '/positions/closed?limit=0' });

    expectProblem(validator, response, 400, 'bad-request');
  });

  it('liefert 400 bad-request, wenn from nach to liegt', async () => {
    const response = await send({ method: 'GET', path: '/positions/closed?from=2000&to=1000' });

    expectDocumentedResponse(validator, 'getClosedPositions', response);
    expectProblem(validator, response, 400, 'bad-request');
  });
});
