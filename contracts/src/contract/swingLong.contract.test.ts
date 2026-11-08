import { describe, expect, it } from 'vitest';
import { expectDocumentedResponse, expectProblem } from '@/contract/assertions';
import { createContractContext } from '@/contract/contractContext';
import { OPERATION_TEST_NAMES } from '@/contract/coverage';

const { validator, send } = createContractContext();

/** 19-stellig, gehört keiner echten Position; das Flag wird am Ende wieder entfernt. */
const UNKNOWN_POSITION_ID = '9199999999999999999';

function swingPath(positionId: string): string {
  return `/positions/${positionId}/swing-long`;
}

describe(OPERATION_TEST_NAMES.setSwingLong, () => {
  it('liefert 400 bad-request bei einer positionId ohne Ziffern', async () => {
    const response = await send({ method: 'PUT', path: swingPath('abc') });

    expectDocumentedResponse(validator, 'setSwingLong', response);
    expectProblem(validator, response, 400, 'bad-request');
  });

  it('liefert 403 forbidden-origin bei fremdem Origin', async () => {
    const response = await send({
      method: 'PUT',
      path: swingPath(UNKNOWN_POSITION_ID),
      headers: { origin: 'http://evil.example' },
    });

    expectDocumentedResponse(validator, 'setSwingLong', response);
    expectProblem(validator, response, 403, 'forbidden-origin');
  });

  it('antwortet 204 ohne Body, auch beim zweiten Aufruf (idempotent)', async () => {
    try {
      for (let attempt = 0; attempt < 2; attempt += 1) {
        const response = await send({ method: 'PUT', path: swingPath(UNKNOWN_POSITION_ID) });

        expectDocumentedResponse(validator, 'setSwingLong', response);
        // Ohne BingX-Keys (CI) gibt es keine Datenbank: dokumentiert ist dann 503 not-configured.
        if (response.status === 503) {
          expectProblem(validator, response, 503, 'not-configured');
          return;
        }
        expect(response.status).toBe(204);
      }
    } finally {
      await send({ method: 'DELETE', path: swingPath(UNKNOWN_POSITION_ID) });
    }
  });
});

describe(OPERATION_TEST_NAMES.clearSwingLong, () => {
  it('liefert 400 bad-request bei einer positionId ohne Ziffern', async () => {
    const response = await send({ method: 'DELETE', path: swingPath('abc') });

    expectDocumentedResponse(validator, 'clearSwingLong', response);
    expectProblem(validator, response, 400, 'bad-request');
  });

  it('liefert 403 forbidden-origin bei fremdem Origin', async () => {
    const response = await send({
      method: 'DELETE',
      path: swingPath(UNKNOWN_POSITION_ID),
      headers: { origin: 'http://evil.example' },
    });

    expectDocumentedResponse(validator, 'clearSwingLong', response);
    expectProblem(validator, response, 403, 'forbidden-origin');
  });

  it('antwortet 204 auch ohne vorhandenes Flag', async () => {
    const response = await send({ method: 'DELETE', path: swingPath(UNKNOWN_POSITION_ID) });

    expectDocumentedResponse(validator, 'clearSwingLong', response);
    // Ohne BingX-Keys (CI) gibt es keine Datenbank: dokumentiert ist dann 503 not-configured.
    if (response.status === 503) {
      expectProblem(validator, response, 503, 'not-configured');
      return;
    }
    expect(response.status).toBe(204);
  });
});
