import { describe, expect, it } from 'vitest';
import { expectDocumentedResponse, expectProblem, jsonBodyOf } from '@/contract/assertions';
import { createContractContext } from '@/contract/contractContext';
import { hostHeaderFor } from '@/config';
import { singleHeader } from '@/httpClient';
import { isJsonObject } from '@/openapi';
import { mediaTypeOf } from '@/validator';

/** Allgemeine Regeln aus info.description in docs/openapi.yaml, gültig für alle Pfade. */
const { config, validator, send } = createContractContext();

const UNKNOWN_PATH = '/gibt-es-nicht-contract-test';
/** Ungültige Prozent-Kodierung; der httpClient sendet den Pfad unverändert. */
const MALFORMED_PATH = '/%zz';

describe('Allgemeine Regeln: Host-Prüfung vor dem Routing', () => {
  it('antwortet bei fremdem Host auf einem unbekannten Pfad mit 403 forbidden-host statt 404', async () => {
    const response = await send({
      method: 'GET',
      path: UNKNOWN_PATH,
      headers: { host: hostHeaderFor(config.baseUrl, 'evil.example') },
    });

    expectProblem(validator, response, 403, 'forbidden-host');
  });

  it('gibt den abgelehnten Host nicht im Body zurück', async () => {
    const response = await send({
      method: 'GET',
      path: '/health',
      headers: { host: hostHeaderFor(config.baseUrl, 'evil.example') },
    });

    expect(response.body).not.toContain('evil.example');
  });

  it('antwortet bei fremdem Host auf einem ungültigen Pfad (/%zz) mit 403 forbidden-host statt 400', async () => {
    const response = await send({
      method: 'GET',
      path: MALFORMED_PATH,
      headers: { host: hostHeaderFor(config.baseUrl, 'evil.example') },
    });

    expectProblem(validator, response, 403, 'forbidden-host');
  });

  it.each(['LOCALHOST', 'LocalHost'])(
    'vergleicht den Host ohne Groß-/Kleinschreibung (%s:<port> ist erlaubt)',
    async (hostname) => {
      const response = await send({
        method: 'GET',
        path: '/health',
        headers: { host: hostHeaderFor(config.baseUrl, hostname) },
      });

      expect(response.status).toBe(200);
      expectDocumentedResponse(validator, 'getHealth', response);
    },
  );

  it.each(['localhost', '127.0.0.1', '[::1]'])(
    'erlaubt %s:<port> laut info.description',
    async (hostname) => {
      const response = await send({
        method: 'GET',
        path: '/health',
        headers: { host: hostHeaderFor(config.baseUrl, hostname) },
      });

      expect(response.status).toBe(200);
    },
  );
});

describe('Allgemeine Regeln: 404 not-found', () => {
  it('antwortet auf einen unbekannten Pfad mit 404 not-found', async () => {
    const response = await send({ method: 'GET', path: UNKNOWN_PATH });

    expectProblem(validator, response, 404, 'not-found');
  });

  it('gibt den unbekannten Pfad nicht im Body zurück', async () => {
    const response = await send({ method: 'GET', path: UNKNOWN_PATH });

    expect(response.body).not.toContain(UNKNOWN_PATH);
  });

  it('antwortet auf POST /health (nicht beschriebene Methode) mit 404 not-found', async () => {
    const response = await send({ method: 'POST', path: '/health' });

    expectProblem(validator, response, 404, 'not-found');
  });

  it('antwortet auf HEAD /health mit 404 und application/problem+json', async () => {
    const response = await send({ method: 'HEAD', path: '/health' });

    expect(response.status).toBe(404);
    expect(mediaTypeOf(singleHeader(response.headers, 'content-type'))).toBe(
      'application/problem+json',
    );
    expect(response.body).toBe('');
  });
});

describe('Allgemeine Regeln: 404 not-found unabhängig von Methode und Body', () => {
  it('antwortet auf POST mit kaputtem JSON-Body an einen unbekannten Pfad mit 404 not-found', async () => {
    const response = await send({
      method: 'POST',
      path: UNKNOWN_PATH,
      headers: { 'content-type': 'application/json' },
      body: '{',
    });

    expectProblem(validator, response, 404, 'not-found');
  });

  it.each([
    ['OPTIONS', '/health'],
    ['DELETE', '/positions/open'],
  ])('antwortet auf %s %s (nicht beschriebene Methode) mit 404 not-found', async (method, path) => {
    const response = await send({ method, path });

    expectProblem(validator, response, 404, 'not-found');
  });
});

describe('Allgemeine Regeln: 400 bad-request für formal fehlerhafte Anfragen', () => {
  it('antwortet auf GET mit ungültiger Prozent-Kodierung (/%zz) mit 400 bad-request', async () => {
    const response = await send({ method: 'GET', path: MALFORMED_PATH });

    expectProblem(validator, response, 400, 'bad-request');
  });

  it('gibt bei 400 bad-request weder den Pfad noch interne Felder zurück', async () => {
    const response = await send({ method: 'GET', path: MALFORMED_PATH });
    const body = jsonBodyOf(response);

    expect(response.body).not.toContain('%zz');
    expect(isJsonObject(body), 'Body ist ein Objekt').toBe(true);
    for (const internalField of ['code', 'message', 'error', 'statusCode']) {
      expect(isJsonObject(body) ? body[internalField] : undefined, internalField).toBeUndefined();
    }
  });
});
