import { describe, expect, it } from 'vitest';
import { expectDocumentedResponse, expectProblem } from '@/contract/assertions';
import { createContractContext } from '@/contract/contractContext';
import { OPERATION_TEST_NAMES } from '@/contract/coverage';
import { hostHeaderFor } from '@/config';
import { singleHeader } from '@/httpClient';
import { mediaTypeOf } from '@/validator';

const { config, validator, send } = createContractContext();

describe(OPERATION_TEST_NAMES.getHealth, () => {
  it('antwortet mit 200, application/json und einem Body laut HealthResponse', async () => {
    const response = await send({ method: 'GET', path: '/health' });

    expect(response.status).toBe(200);
    expect(mediaTypeOf(singleHeader(response.headers, 'content-type'))).toBe('application/json');
    expect(expectDocumentedResponse(validator, 'getHealth', response)).toEqual({ status: 'ok' });
  });

  it('antwortet bei fremdem Host mit 403 forbidden-host laut Spec', async () => {
    const response = await send({
      method: 'GET',
      path: '/health',
      headers: { host: hostHeaderFor(config.baseUrl, 'evil.example') },
    });

    expectDocumentedResponse(validator, 'getHealth', response);
    expectProblem(validator, response, 403, 'forbidden-host');
  });
});
