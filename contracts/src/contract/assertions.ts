import { expect } from 'vitest';
import { parseJsonBody, singleHeader, type HttpResponse } from '@/httpClient';
import { isJsonObject } from '@/openapi';
import { mediaTypeOf, type OpenApiValidator } from '@/validator';

const PROBLEM_MEDIA_TYPE = 'application/problem+json';
const INTEGER_PATTERN = /^\d+$/;

/** Parst den Body als JSON und macht eine verständliche Assertion daraus, falls das scheitert. */
export function jsonBodyOf(response: HttpResponse): unknown {
  try {
    return parseJsonBody(response);
  } catch {
    expect.fail(`Body ist kein JSON: ${response.body.slice(0, 200)}`);
  }
}

/** Regel aus info.description: Retry-After nur bei 503 rate-limited, dort als ganze Zahl. */
export function expectRetryAfterRule(response: HttpResponse, body: unknown): void {
  const retryAfter = singleHeader(response.headers, 'retry-after');
  const isRateLimited =
    response.status === 503 && isJsonObject(body) && body['type'] === 'rate-limited';
  if (isRateLimited) {
    expect(retryAfter, 'Retry-After bei 503 rate-limited').toMatch(INTEGER_PATTERN);
  } else {
    expect(
      response.headers['retry-after'],
      `kein Retry-After bei ${response.status}`,
    ).toBeUndefined();
  }
}

/** Fehlerantwort: application/problem+json, Schema Problem, status im Body = HTTP-Status. */
export function expectProblemShape(
  validator: OpenApiValidator,
  response: HttpResponse,
  body: unknown,
): void {
  expect(mediaTypeOf(singleHeader(response.headers, 'content-type'))).toBe(PROBLEM_MEDIA_TYPE);
  expect(validator.validateSchema('Problem', body).errors).toEqual([]);
  expect(isJsonObject(body) ? body['status'] : undefined, 'status im Body').toBe(response.status);
}

/**
 * Antwort einer Operation: Status, Content-Type und Body laut Spec, Retry-After-Regel und bei
 * Fehlern die Problem-Form. Gibt den geparsten Body zurück.
 */
export function expectDocumentedResponse(
  validator: OpenApiValidator,
  operationId: string,
  response: HttpResponse,
): unknown {
  expect(
    validator.isDocumentedStatus(operationId, response.status),
    `Status ${response.status} für ${operationId} dokumentiert (${validator.documentedStatuses(operationId).join(', ')})`,
  ).toBe(true);
  const body = jsonBodyOf(response);
  const result = validator.validateResponse({
    operationId,
    status: response.status,
    contentType: singleHeader(response.headers, 'content-type'),
    body,
  });
  expect(result.errors).toEqual([]);
  if (response.status >= 400) expectProblemShape(validator, response, body);
  expectRetryAfterRule(response, body);
  return body;
}

/** Fehlerantwort mit erwartetem Status und Problem-Typ, auch für Pfade außerhalb der Spec. */
export function expectProblem(
  validator: OpenApiValidator,
  response: HttpResponse,
  expectedStatus: number,
  expectedType: string,
): void {
  expect(response.status).toBe(expectedStatus);
  const body = jsonBodyOf(response);
  expectProblemShape(validator, response, body);
  expect(isJsonObject(body) ? body['type'] : undefined, 'Problem-Typ').toBe(expectedType);
  expectRetryAfterRule(response, body);
}
