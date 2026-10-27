import { describe, expect, it } from 'vitest';
import { expectDocumentedResponse, expectProblem } from '@/contract/assertions';
import { createContractContext } from '@/contract/contractContext';
import { OPERATION_TEST_NAMES } from '@/contract/coverage';

const { config, validator, send } = createContractContext();

// Positions-ID, zu der es garantiert keine Notiz gibt.
const UNKNOWN_POSITION_ID = '9999999999999999999';

describe(OPERATION_TEST_NAMES.getPositionNote, () => {
  if (config.expectConfigured) {
    it('liefert 404 not-found für eine Position ohne Notiz', async () => {
      const response = await send({
        method: 'GET',
        path: `/positions/${UNKNOWN_POSITION_ID}/note`,
      });

      expectDocumentedResponse(validator, 'getPositionNote', response);
      expectProblem(validator, response, 404, 'not-found');
    });
  } else {
    it('liefert ohne Datenbank 503 not-configured', async () => {
      const response = await send({
        method: 'GET',
        path: `/positions/${UNKNOWN_POSITION_ID}/note`,
      });

      expectDocumentedResponse(validator, 'getPositionNote', response);
      expectProblem(validator, response, 503, 'not-configured');
    });
  }

  it('liefert 400 bad-request bei einer nicht numerischen positionId', async () => {
    const response = await send({ method: 'GET', path: '/positions/abc/note' });

    expectProblem(validator, response, 400, 'bad-request');
  });
});
