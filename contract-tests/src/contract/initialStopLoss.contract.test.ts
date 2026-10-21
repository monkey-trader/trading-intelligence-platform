import { beforeAll, describe, expect, it } from 'vitest';
import { expectDocumentedResponse, expectProblem } from '@/contract/assertions';
import { createContractContext } from '@/contract/contractContext';
import { OPERATION_TEST_NAMES } from '@/contract/coverage';
import { isJsonObject } from '@/openapi';

const { config, validator, send } = createContractContext();

/** 19-stellig, gehört keiner echten Position. */
const UNKNOWN_POSITION_ID = '9199999999999999999';
const JSON_HEADERS = { 'content-type': 'application/json' } as const;

function stopPath(positionId: string): string {
  return `/positions/${positionId}/initial-stop-loss`;
}

function putStop(positionId: string, body: string, headers: Record<string, string> = JSON_HEADERS) {
  return send({ method: 'PUT', path: stopPath(positionId), headers, body });
}

type OpenPositionRef = Readonly<{ positionId: string; side: string; avgOpenPrice: string }>;

function firstOpenPosition(body: unknown): OpenPositionRef | undefined {
  if (!isJsonObject(body) || !Array.isArray(body['positions'])) return undefined;
  const first: unknown = body['positions'][0];
  if (!isJsonObject(first)) return undefined;
  const { positionId, side, avgOpenPrice } = first;
  return typeof positionId === 'string' &&
    typeof side === 'string' &&
    typeof avgOpenPrice === 'string'
    ? { positionId, side, avgOpenPrice }
    : undefined;
}

describe(OPERATION_TEST_NAMES.setInitialStopLoss, () => {
  it('antwortet auf eine unbekannte Position laut Spec', async () => {
    const response = await putStop(UNKNOWN_POSITION_ID, '{"initialStopLoss":"0.9"}');

    expectDocumentedResponse(validator, 'setInitialStopLoss', response);
  });

  // Diese Prüfungen laufen vor jedem BingX-Aufruf, also auch ohne Keys.
  it('liefert 415 unsupported-media-type bei Content-Type text/plain', async () => {
    const response = await putStop(UNKNOWN_POSITION_ID, '{"initialStopLoss":"0.9"}', {
      'content-type': 'text/plain',
    });

    expectDocumentedResponse(validator, 'setInitialStopLoss', response);
    expectProblem(validator, response, 415, 'unsupported-media-type');
  });

  it.each([
    ['kein JSON', 'nope'],
    ['Zahl statt String', '{"initialStopLoss":0.9}'],
    ['unbekanntes Feld', '{"initialStopLoss":"0.9","extra":1}'],
    ['Feld fehlt', '{}'],
  ])('liefert 400 bad-request bei Body: %s', async (_label, body) => {
    const response = await putStop(UNKNOWN_POSITION_ID, body);

    expectDocumentedResponse(validator, 'setInitialStopLoss', response);
    expectProblem(validator, response, 400, 'bad-request');
  });

  it('liefert 400 bad-request bei einer positionId ohne Ziffern', async () => {
    const response = await putStop('abc', '{"initialStopLoss":"0.9"}');

    expectDocumentedResponse(validator, 'setInitialStopLoss', response);
    expectProblem(validator, response, 400, 'bad-request');
  });

  it('liefert 413 payload-too-large bei mehr als 1 KiB', async () => {
    const body = JSON.stringify({ initialStopLoss: '0.9', padding: 'a'.repeat(1100) });
    const response = await putStop(UNKNOWN_POSITION_ID, body);

    expectDocumentedResponse(validator, 'setInitialStopLoss', response);
    expectProblem(validator, response, 413, 'payload-too-large');
  });

  it('liefert 403 forbidden-origin bei fremdem Origin', async () => {
    const response = await putStop(UNKNOWN_POSITION_ID, '{"initialStopLoss":"0.9"}', {
      ...JSON_HEADERS,
      origin: 'http://evil.example',
    });

    expectDocumentedResponse(validator, 'setInitialStopLoss', response);
    expectProblem(validator, response, 403, 'forbidden-origin');
  });

  if (config.expectConfigured) {
    it('liefert 404 not-found bei unbekannter positionId', async () => {
      const response = await putStop(UNKNOWN_POSITION_ID, '{"initialStopLoss":"0.9"}');

      expectDocumentedResponse(validator, 'setInitialStopLoss', response);
      expectProblem(validator, response, 404, 'not-found');
    });

    describe('mit einer offenen Position des Kontos', () => {
      let position: OpenPositionRef | undefined;

      beforeAll(async () => {
        const response = await send({ method: 'GET', path: '/positions/open' });
        position = firstOpenPosition(
          expectDocumentedResponse(validator, 'getOpenPositions', response),
        );
      });

      it('liefert 422 invalid-stop-loss, wenn der SL gleich dem Entry ist', async (testContext) => {
        if (position === undefined) return testContext.skip(); // hängt vom Live-Konto ab
        const response = await putStop(
          position.positionId,
          JSON.stringify({ initialStopLoss: position.avgOpenPrice }),
        );

        expectDocumentedResponse(validator, 'setInitialStopLoss', response);
        expectProblem(validator, response, 422, 'invalid-stop-loss');
      });

      it('liefert 422 invalid-stop-loss auf der Gewinnseite', async (testContext) => {
        if (position === undefined) return testContext.skip();
        const wrongSide =
          position.side === 'LONG'
            ? Number(position.avgOpenPrice) * 2
            : Number(position.avgOpenPrice) / 2;
        const response = await putStop(
          position.positionId,
          JSON.stringify({ initialStopLoss: String(wrongSide) }),
        );

        expectDocumentedResponse(validator, 'setInitialStopLoss', response);
        expectProblem(validator, response, 422, 'invalid-stop-loss');
      });

      it('speichert einen gültigen SL (200 InitialStopLoss), DELETE setzt zurück (204)', async (testContext) => {
        if (position === undefined) return testContext.skip();
        const validStop =
          position.side === 'LONG'
            ? Number(position.avgOpenPrice) / 2
            : Number(position.avgOpenPrice) * 2;
        const putResponse = await putStop(
          position.positionId,
          JSON.stringify({ initialStopLoss: String(validStop) }),
        );

        expect(putResponse.status).toBe(200);
        const body = expectDocumentedResponse(validator, 'setInitialStopLoss', putResponse);
        expect(validator.validateSchema('InitialStopLoss', body).errors).toEqual([]);
        expect(isJsonObject(body) ? body['source'] : undefined).toBe('MANUAL');

        const deleteResponse = await send({
          method: 'DELETE',
          path: stopPath(position.positionId),
        });
        expect(deleteResponse.status).toBe(204);
      });
    });
  }
});

describe(OPERATION_TEST_NAMES.clearInitialStopLoss, () => {
  it('liefert 204 auch ohne vorhandenen Override (idempotent) oder ein dokumentiertes Problem', async () => {
    const response = await send({ method: 'DELETE', path: stopPath(UNKNOWN_POSITION_ID) });

    expectDocumentedResponse(validator, 'clearInitialStopLoss', response);
  });

  it('liefert 400 bad-request bei einer positionId ohne Ziffern', async () => {
    const response = await send({ method: 'DELETE', path: stopPath('abc') });

    expectDocumentedResponse(validator, 'clearInitialStopLoss', response);
    expectProblem(validator, response, 400, 'bad-request');
  });

  it('liefert 403 forbidden-origin bei fremdem Origin', async () => {
    const response = await send({
      method: 'DELETE',
      path: stopPath(UNKNOWN_POSITION_ID),
      headers: { origin: 'http://evil.example' },
    });

    expectDocumentedResponse(validator, 'clearInitialStopLoss', response);
    expectProblem(validator, response, 403, 'forbidden-origin');
  });
});
