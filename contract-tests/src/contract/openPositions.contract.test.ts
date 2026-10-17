import { describe, expect, it } from 'vitest';
import { expectDocumentedResponse, expectProblem } from '@/contract/assertions';
import { createContractContext } from '@/contract/contractContext';
import { OPERATION_TEST_NAMES } from '@/contract/coverage';
import { hostHeaderFor } from '@/config';
import { isJsonObject } from '@/openapi';

const { config, validator, send } = createContractContext();

function openTimesOf(body: unknown): number[] {
  if (!isJsonObject(body) || !Array.isArray(body['positions'])) return [];
  return body['positions'].map((position: unknown) =>
    isJsonObject(position) && typeof position['openTime'] === 'number' ? position['openTime'] : NaN,
  );
}

describe(OPERATION_TEST_NAMES.getOpenPositions, () => {
  it('antwortet mit einem Status, Content-Type und Body laut Spec', async () => {
    const response = await send({ method: 'GET', path: '/positions/open' });

    expectDocumentedResponse(validator, 'getOpenPositions', response);
  });

  if (config.expectConfigured) {
    it('liefert mit Keys 200 und einen OpenPositionsSnapshot', async () => {
      const response = await send({ method: 'GET', path: '/positions/open' });

      expect(response.status).toBe(200);
      const body = expectDocumentedResponse(validator, 'getOpenPositions', response);
      expect(validator.validateSchema('OpenPositionsSnapshot', body).errors).toEqual([]);
    });

    it('sortiert die Positionen nach openTime absteigend (neueste zuerst)', async () => {
      const response = await send({ method: 'GET', path: '/positions/open' });
      const openTimes = openTimesOf(
        expectDocumentedResponse(validator, 'getOpenPositions', response),
      );

      expect(openTimes).toEqual([...openTimes].sort((left, right) => right - left));
    });
  } else {
    it('liefert ohne Keys 503 not-configured ohne Retry-After', async () => {
      const response = await send({ method: 'GET', path: '/positions/open' });

      expectDocumentedResponse(validator, 'getOpenPositions', response);
      expectProblem(validator, response, 503, 'not-configured');
    });
  }

  it('antwortet bei fremdem Host mit 403 forbidden-host laut Spec', async () => {
    const response = await send({
      method: 'GET',
      path: '/positions/open',
      headers: { host: hostHeaderFor(config.baseUrl, 'evil.example') },
    });

    expectDocumentedResponse(validator, 'getOpenPositions', response);
    expectProblem(validator, response, 403, 'forbidden-host');
  });
});
