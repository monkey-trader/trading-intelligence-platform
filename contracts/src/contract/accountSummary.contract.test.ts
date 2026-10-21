import { describe, expect, it } from 'vitest';
import { expectDocumentedResponse, expectProblem } from '@/contract/assertions';
import { createContractContext } from '@/contract/contractContext';
import { OPERATION_TEST_NAMES } from '@/contract/coverage';
import { hostHeaderFor } from '@/config';
import { isJsonObject } from '@/openapi';

const { config, validator, send } = createContractContext();

describe(OPERATION_TEST_NAMES.getAccountSummary, () => {
  it('antwortet mit einem Status, Content-Type und Body laut Spec', async () => {
    const response = await send({ method: 'GET', path: '/account/summary' });

    expectDocumentedResponse(validator, 'getAccountSummary', response);
  });

  if (config.expectConfigured) {
    it('liefert mit Keys 200 und eine AccountSummary mit Perioden Heute, 7T, 30T', async () => {
      const response = await send({ method: 'GET', path: '/account/summary' });

      expect(response.status).toBe(200);
      const body = expectDocumentedResponse(validator, 'getAccountSummary', response);
      expect(validator.validateSchema('AccountSummary', body).errors).toEqual([]);
      const periods = isJsonObject(body) && Array.isArray(body['periods']) ? body['periods'] : [];
      expect(periods.map((entry) => (isJsonObject(entry) ? entry['period'] : undefined))).toEqual([
        'today',
        '7d',
        '30d',
      ]);
    });
  } else {
    it('liefert ohne Keys 503 not-configured', async () => {
      const response = await send({ method: 'GET', path: '/account/summary' });

      expectDocumentedResponse(validator, 'getAccountSummary', response);
      expectProblem(validator, response, 503, 'not-configured');
    });
  }

  it('antwortet bei fremdem Host mit 403 forbidden-host laut Spec', async () => {
    const response = await send({
      method: 'GET',
      path: '/account/summary',
      headers: { host: hostHeaderFor(config.baseUrl, 'evil.example') },
    });

    expectDocumentedResponse(validator, 'getAccountSummary', response);
    expectProblem(validator, response, 403, 'forbidden-host');
  });
});
