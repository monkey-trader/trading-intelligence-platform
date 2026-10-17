import { createHmac } from 'node:crypto';
import { describe, expect, it } from 'vitest';
import { BingXClient } from '@/infrastructure/bingx/BingXClient';
import { BingXApiError, BingXHttpError, BingXRateLimitError } from '@/infrastructure/bingx/errors';
import rangeTooLargeFixture from '@fixtures/bingx/positionHistory.error.rangeTooLarge.json';
import page0Fixture from '@fixtures/bingx/positionHistory.page0.json';

const API_KEY = 'TEST-API-KEY-1234567890';
const API_SECRET = 'TEST-API-SECRET-0987654321';
const BASE_URL = 'https://bingx.example.test';
const POSITION_HISTORY_PATH = '/openApi/swap/v1/trade/positionHistory';

type RecordedRequest = {
  url: URL;
  rawUrl: string;
  method: string | undefined;
  headers: Headers;
  signal: AbortSignal | null | undefined;
};
type FakeReply = { status: number; body: unknown } | { networkError: string };

/** Fake für fetch: spielt vorbereitete Antworten der Reihe nach ab und merkt sich jede Anfrage. */
function createFakeFetch(replies: readonly FakeReply[]) {
  const requests: RecordedRequest[] = [];
  let replyIndex = 0;

  const fetchFn = async (input: string | URL | Request, init?: RequestInit): Promise<Response> => {
    const rawUrl = input instanceof Request ? input.url : String(input);
    requests.push({
      url: new URL(rawUrl),
      rawUrl,
      method: init?.method ?? (input instanceof Request ? input.method : undefined),
      headers: new Headers(init?.headers),
      signal: init?.signal,
    });
    const reply = replies[Math.min(replyIndex, replies.length - 1)];
    replyIndex += 1;
    if (!reply) throw new Error('Keine Fake-Antwort vorbereitet');
    if ('networkError' in reply) throw new TypeError(reply.networkError);
    return new Response(JSON.stringify(reply.body), {
      status: reply.status,
      headers: { 'content-type': 'application/json' },
    });
  };

  return { fetchFn, requests };
}

function createClient(
  replies: readonly FakeReply[],
  options: { maxRetries?: number; requestPauseMs?: number } = {},
) {
  const fake = createFakeFetch(replies);
  const sleepCalls: number[] = [];
  let currentMs = 1_780_000_000_000;

  const client = new BingXClient({
    apiKey: API_KEY,
    apiSecret: API_SECRET,
    baseUrl: BASE_URL,
    fetchFn: fake.fetchFn,
    sleep: async (milliseconds: number) => {
      sleepCalls.push(milliseconds);
    },
    // Jeder Aufruf liefert eine neue Zeit, damit ein Retry einen neuen timestamp bekommt.
    now: () => {
      currentMs += 1_000;
      return currentMs;
    },
    maxRetries: options.maxRetries ?? 3,
    requestPauseMs: options.requestPauseMs ?? 0,
  });

  return { client, requests: fake.requests, sleepCalls };
}

const okReply = (data: unknown): FakeReply => ({ status: 200, body: { code: 0, msg: '', data } });
const rateLimitReply: FakeReply = { status: 429, body: { code: 100410, msg: 'rate limited' } };
const apiCodeReply = (code: number): FakeReply => ({
  status: 200,
  body: { code, msg: `BingX meldet ${code}`, data: {} },
});

/** Teilt die URL in den signierten Teil und die Signatur. */
function splitSignedQuery(rawUrl: string): { signedQuery: string; signature: string } {
  const query = rawUrl.slice(rawUrl.indexOf('?') + 1);
  const match = /^(.*)&signature=([0-9a-f]+)$/.exec(query);
  if (!match?.[1] || !match[2]) throw new Error(`Keine Signatur am Ende der URL: ${rawUrl}`);
  return { signedQuery: match[1], signature: match[2] };
}

async function captureRejection(promise: Promise<unknown>): Promise<unknown> {
  try {
    await promise;
  } catch (error) {
    return error;
  }
  throw new Error('Es wurde ein Fehler erwartet, aber keiner geworfen.');
}

function expectNoSecretsIn(error: unknown): void {
  expect(error).toBeInstanceOf(Error);
  const text = `${String(error)} ${error instanceof Error ? error.message : ''}`;
  expect(text).not.toContain(API_KEY);
  expect(text).not.toContain(API_SECRET);
  expect(text).not.toContain('signature=');
}

describe('BingXClient', () => {
  describe('Happy Path', () => {
    it('gibt das Feld data der Antwort zurück', async () => {
      const { client } = createClient([okReply(page0Fixture.data)]);

      const data = await client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' });

      expect(data).toEqual(page0Fixture.data);
    });

    it('sendet nur GET-Requests an baseUrl + path', async () => {
      const { client, requests } = createClient([okReply([])]);

      await client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' });

      expect(requests).toHaveLength(1);
      expect(requests[0]?.method).toBe('GET');
      expect(requests[0]?.url.origin).toBe(BASE_URL);
      expect(requests[0]?.url.pathname).toBe(POSITION_HISTORY_PATH);
    });

    it('setzt den Header X-BX-APIKEY', async () => {
      const { client, requests } = createClient([okReply([])]);

      await client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' });

      expect(requests[0]?.headers.get('X-BX-APIKEY')).toBe(API_KEY);
    });

    it('überträgt alle Parameter plus timestamp aus dem injizierten now', async () => {
      const { client, requests } = createClient([okReply([])]);

      await client.get(POSITION_HISTORY_PATH, {
        symbol: 'ONDO-USDT',
        startTs: 1780000000000,
        endTs: 1780100000000,
        pageIndex: 0,
        pageSize: 100,
      });

      const params = requests[0]?.url.searchParams;
      expect(params?.get('symbol')).toBe('ONDO-USDT');
      expect(params?.get('startTs')).toBe('1780000000000');
      expect(params?.get('endTs')).toBe('1780100000000');
      expect(params?.get('pageIndex')).toBe('0');
      expect(params?.get('pageSize')).toBe('100');
      expect(Number(params?.get('timestamp'))).toBeGreaterThan(1_780_000_000_000);
    });

    it('signiert den Query-String (inkl. timestamp) korrekt mit HMAC-SHA256 und hängt signature ans Ende', async () => {
      const { client, requests } = createClient([okReply([])]);

      await client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT', startTs: 1780000000000 });

      const { signedQuery, signature } = splitSignedQuery(requests[0]?.rawUrl ?? '');
      expect(signedQuery).toContain('timestamp=');
      expect(signedQuery).toContain('symbol=ONDO-USDT');
      expect(signature).toBe(createHmac('sha256', API_SECRET).update(signedQuery).digest('hex'));
    });

    it('gibt fetch ein AbortSignal für den Timeout mit', async () => {
      const { client, requests } = createClient([okReply([])]);

      await client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' });

      expect(requests[0]?.signal).toBeInstanceOf(AbortSignal);
    });

    it('legt nach einem erfolgreichen Request die Pause requestPauseMs über sleep ein', async () => {
      const { client, sleepCalls } = createClient([okReply([])], { requestPauseMs: 250 });

      await client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' });

      expect(sleepCalls).toContain(250);
    });
  });

  describe('HTTP 429 (Rate-Limit)', () => {
    it('wiederholt mit neuem timestamp UND neuer Signatur', async () => {
      const { client, requests } = createClient([rateLimitReply, okReply(['ok'])]);

      const data = await client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' });

      expect(data).toEqual(['ok']);
      expect(requests).toHaveLength(2);
      const [firstRequest, secondRequest] = requests;
      const firstTimestamp = firstRequest?.url.searchParams.get('timestamp');
      const secondTimestamp = secondRequest?.url.searchParams.get('timestamp');
      expect(secondTimestamp).not.toBe(firstTimestamp);
      const firstSigned = splitSignedQuery(firstRequest?.rawUrl ?? '');
      const secondSigned = splitSignedQuery(secondRequest?.rawUrl ?? '');
      expect(secondSigned.signature).not.toBe(firstSigned.signature);
      expect(secondSigned.signature).toBe(
        createHmac('sha256', API_SECRET).update(secondSigned.signedQuery).digest('hex'),
      );
    });

    it('wartet exponentiell wachsend über das injizierte sleep', async () => {
      const { client, sleepCalls } = createClient(
        [rateLimitReply, rateLimitReply, rateLimitReply, okReply([])],
        { maxRetries: 3, requestPauseMs: 0 },
      );

      await client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' });

      const backoffDelays = sleepCalls.filter((milliseconds) => milliseconds > 0);
      expect(backoffDelays).toHaveLength(3);
      expect(backoffDelays[0]).toBeGreaterThan(0);
      expect(backoffDelays[1]).toBe((backoffDelays[0] ?? 0) * 2);
      expect(backoffDelays[2]).toBe((backoffDelays[1] ?? 0) * 2);
    });

    it('wirft BingXRateLimitError, wenn alle Retries erschöpft sind', async () => {
      const { client, requests } = createClient([rateLimitReply], { maxRetries: 2 });

      const error = await captureRejection(
        client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' }),
      );

      expect(error).toBeInstanceOf(BingXRateLimitError);
      // erster Versuch + 2 Retries
      expect(requests).toHaveLength(3);
      expectNoSecretsIn(error);
    });
  });

  describe('Rate-Limit und vorübergehende Fehler bei HTTP 200 / 5xx / Netzwerk', () => {
    it('wiederholt bei HTTP 200 mit code 100410 (Rate-Limit) mit neuer Signatur', async () => {
      const { client, requests } = createClient([apiCodeReply(100410), okReply(['ok'])]);

      const data = await client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' });

      expect(data).toEqual(['ok']);
      expect(requests).toHaveLength(2);
      const firstSigned = splitSignedQuery(requests[0]?.rawUrl ?? '');
      const secondSigned = splitSignedQuery(requests[1]?.rawUrl ?? '');
      expect(secondSigned.signature).not.toBe(firstSigned.signature);
    });

    it('wirft BingXRateLimitError, wenn code 100410 auch nach allen Retries bleibt', async () => {
      const { client, requests } = createClient([apiCodeReply(100410)], { maxRetries: 2 });

      const error = await captureRejection(
        client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' }),
      );

      expect(error).toBeInstanceOf(BingXRateLimitError);
      expect(requests).toHaveLength(3);
      expectNoSecretsIn(error);
    });

    it.each([100500, 100503])('wiederholt bei vorübergehendem API-Fehler code %i', async (code) => {
      const { client, requests } = createClient([apiCodeReply(code), okReply(['ok'])]);

      const data = await client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' });

      expect(data).toEqual(['ok']);
      expect(requests).toHaveLength(2);
    });

    it('wiederholt bei HTTP 503 und liefert danach das Ergebnis', async () => {
      const { client, requests, sleepCalls } = createClient([
        { status: 503, body: { message: 'busy' } },
        okReply(['ok']),
      ]);

      const data = await client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' });

      expect(data).toEqual(['ok']);
      expect(requests).toHaveLength(2);
      expect(sleepCalls.filter((milliseconds) => milliseconds > 0)).toHaveLength(1);
    });

    it('wiederholt bei einem Netzwerkfehler (z.B. Timeout) und liefert danach das Ergebnis', async () => {
      const { client, requests } = createClient([
        { networkError: 'fetch failed' },
        okReply(['ok']),
      ]);

      const data = await client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' });

      expect(data).toEqual(['ok']);
      expect(requests).toHaveLength(2);
    });

    it('wirft nach erschöpften Retries bei Netzwerkfehlern einen Fehler ohne Secrets', async () => {
      const { client, requests } = createClient([{ networkError: 'fetch failed' }], {
        maxRetries: 2,
      });

      const error = await captureRejection(
        client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' }),
      );

      expect(requests).toHaveLength(3);
      expect(error instanceof Error ? error.message : '').toContain(POSITION_HISTORY_PATH);
      expectNoSecretsIn(error);
    });
  });

  describe('Fehlerfälle', () => {
    it('wirft BingXApiError mit code, wenn code != 0', async () => {
      const { client } = createClient([{ status: 200, body: rangeTooLargeFixture }]);

      const error = await captureRejection(
        client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' }),
      );

      expect(error).toBeInstanceOf(BingXApiError);
      expect(error instanceof BingXApiError ? error.code : undefined).toBe(109400);
      expect(error instanceof BingXApiError ? error.path : undefined).toBe(POSITION_HISTORY_PATH);
      expect(error instanceof Error ? error.message : '').toContain('109400');
      expectNoSecretsIn(error);
    });

    it('wiederholt bei code != 0 nicht', async () => {
      const { client, requests } = createClient([{ status: 200, body: rangeTooLargeFixture }]);

      await captureRejection(client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' }));

      expect(requests).toHaveLength(1);
    });

    it('wirft BingXHttpError bei HTTP 500', async () => {
      const { client } = createClient([{ status: 500, body: { message: 'internal' } }]);

      const error = await captureRejection(
        client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' }),
      );

      expect(error).toBeInstanceOf(BingXHttpError);
      expect(error instanceof BingXHttpError ? error.status : undefined).toBe(500);
      expectNoSecretsIn(error);
    });

    it('wirft BingXHttpError bei HTTP 401, ohne zu wiederholen', async () => {
      const { client, requests } = createClient([
        { status: 401, body: { message: 'unauthorized' } },
      ]);

      const error = await captureRejection(
        client.get(POSITION_HISTORY_PATH, { symbol: 'ONDO-USDT' }),
      );

      expect(error).toBeInstanceOf(BingXHttpError);
      expect(requests).toHaveLength(1);
      expectNoSecretsIn(error);
    });
  });
});
