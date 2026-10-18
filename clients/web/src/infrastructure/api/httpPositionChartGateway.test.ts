import { describe, expect, it, vi } from 'vitest';
import type { components } from '@/infrastructure/api/generated';
import { createHttpPositionChartGateway } from '@/infrastructure/api/httpPositionChartGateway';
import {
  CHART_POSITION_ID,
  xagChartWithoutProtection,
  xagLongChart,
} from '@/test/fixtures/positionChart';

type ProblemBody = components['schemas']['Problem'];

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json; charset=utf-8' },
  });
}

function problemResponse(body: ProblemBody, extraHeaders: Record<string, string> = {}): Response {
  return new Response(JSON.stringify(body), {
    status: body.status,
    headers: { 'Content-Type': 'application/problem+json; charset=utf-8', ...extraHeaders },
  });
}

function fakeFetchReturning(response: Response) {
  return vi.fn<typeof fetch>().mockResolvedValue(response);
}

function newSignal(): AbortSignal {
  return new AbortController().signal;
}

describe('createHttpPositionChartGateway', () => {
  describe('Anfrage', () => {
    it('fragt GET /api/positions/{id}/chart mit interval und limit ab', async () => {
      const fetchFn = fakeFetchReturning(jsonResponse(xagLongChart));
      const gateway = createHttpPositionChartGateway(fetchFn);

      await gateway.fetchChart(CHART_POSITION_ID, '1h', 200, newSignal());

      const [requestUrl, requestInit] = fetchFn.mock.calls[0] ?? [];
      expect(String(requestUrl)).toBe(
        `/api/positions/${CHART_POSITION_ID}/chart?interval=1h&limit=200`,
      );
      expect(requestInit?.method ?? 'GET').toBe('GET');
    });

    it('kodiert die positionId in der URL', async () => {
      const fetchFn = fakeFetchReturning(jsonResponse(xagLongChart));
      const gateway = createHttpPositionChartGateway(fetchFn);

      await gateway.fetchChart('12/../3 4', '15m', 300, newSignal());

      const [requestUrl] = fetchFn.mock.calls[0] ?? [];
      expect(String(requestUrl)).toContain('/positions/12%2F..%2F3%204/chart?');
    });

    it('verwendet eine abweichende Basis-URL', async () => {
      const fetchFn = fakeFetchReturning(jsonResponse(xagLongChart));
      const gateway = createHttpPositionChartGateway(fetchFn, 'http://127.0.0.1:3000');

      await gateway.fetchChart(CHART_POSITION_ID, '15m', 300, newSignal());

      const [requestUrl] = fetchFn.mock.calls[0] ?? [];
      expect(String(requestUrl)).toBe(
        `http://127.0.0.1:3000/positions/${CHART_POSITION_ID}/chart?interval=15m&limit=300`,
      );
    });
  });

  describe('Antwort', () => {
    it('liefert den Chart mit allen Feldern, Dezimalwerte bleiben Strings', async () => {
      const gateway = createHttpPositionChartGateway(
        fakeFetchReturning(jsonResponse(xagLongChart)),
      );

      const result = await gateway.fetchChart(CHART_POSITION_ID, '15m', 300, newSignal());

      expect(result).toEqual({ kind: 'ok', chart: xagLongChart });
    });

    it('liefert leere SL/TP-Listen unverändert', async () => {
      const gateway = createHttpPositionChartGateway(
        fakeFetchReturning(jsonResponse(xagChartWithoutProtection)),
      );

      const result = await gateway.fetchChart(CHART_POSITION_ID, '15m', 300, newSignal());

      expect(result).toEqual({ kind: 'ok', chart: xagChartWithoutProtection });
    });

    it.each([
      ['Kerzen fehlen', { ...xagLongChart, candles: undefined }],
      ['Entry ohne Preis', { ...xagLongChart, entry: { time: 1 } }],
      ['unbekanntes Intervall', { ...xagLongChart, interval: '3m' }],
      ['Preis kein Dezimal-String', { ...xagLongChart, entry: { price: 55.75, time: 1 } }],
      [
        'SL ohne distanceRatio',
        { ...xagLongChart, stopLoss: [{ orderId: '1', triggerPrice: '1', quantity: '1' }] },
      ],
      ['unbekannter Marker', { ...xagLongChart, markers: [{ time: 1, kind: 'EXIT', price: '1' }] }],
      ['Kerze mit falschem Typ', { ...xagLongChart, candles: [{ time: 'x' }] }],
      ['kein Objekt', []],
    ])('liefert invalid-body bei ungültiger Form (%s)', async (_name, body) => {
      const gateway = createHttpPositionChartGateway(fakeFetchReturning(jsonResponse(body)));

      const result = await gateway.fetchChart(CHART_POSITION_ID, '15m', 300, newSignal());

      expect(result).toEqual({ kind: 'problem', problemType: 'invalid-body' });
    });

    it('liefert invalid-body bei kaputtem JSON', async () => {
      const gateway = createHttpPositionChartGateway(
        fakeFetchReturning(new Response('{nope', { status: 200 })),
      );

      const result = await gateway.fetchChart(CHART_POSITION_ID, '15m', 300, newSignal());

      expect(result).toEqual({ kind: 'problem', problemType: 'invalid-body' });
    });
  });

  describe('Fehler', () => {
    it.each([
      [400, 'bad-request'],
      [404, 'not-found'],
      [502, 'upstream-error'],
      [503, 'not-configured'],
      [504, 'unreachable'],
    ] as const)('mappt Problem+JSON mit Status %i auf %s', async (status, type) => {
      const gateway = createHttpPositionChartGateway(
        fakeFetchReturning(problemResponse({ type, title: 'Fehler', status })),
      );

      const result = await gateway.fetchChart(CHART_POSITION_ID, '15m', 300, newSignal());

      expect(result).toEqual({ kind: 'problem', problemType: type });
    });

    it('übernimmt Retry-After in Sekunden bei rate-limited', async () => {
      const gateway = createHttpPositionChartGateway(
        fakeFetchReturning(
          problemResponse(
            { type: 'rate-limited', title: 'Zu viele', status: 429 },
            { 'Retry-After': '30' },
          ),
        ),
      );

      const result = await gateway.fetchChart(CHART_POSITION_ID, '15m', 300, newSignal());

      expect(result).toEqual({
        kind: 'problem',
        problemType: 'rate-limited',
        retryAfterSeconds: 30,
      });
    });

    it('liefert invalid-body bei einem unbekannten Problem-Typ', async () => {
      const gateway = createHttpPositionChartGateway(
        fakeFetchReturning(
          new Response(JSON.stringify({ type: 'neu', title: 'x', status: 500 }), {
            status: 500,
            headers: { 'Content-Type': 'application/problem+json' },
          }),
        ),
      );

      const result = await gateway.fetchChart(CHART_POSITION_ID, '15m', 300, newSignal());

      expect(result).toEqual({ kind: 'problem', problemType: 'invalid-body' });
    });

    it('liefert backend-unreachable bei Fehlerstatus ohne problem+json', async () => {
      const gateway = createHttpPositionChartGateway(
        fakeFetchReturning(new Response('Bad Gateway', { status: 502 })),
      );

      const result = await gateway.fetchChart(CHART_POSITION_ID, '15m', 300, newSignal());

      expect(result).toEqual({ kind: 'problem', problemType: 'backend-unreachable' });
    });

    it('liefert backend-unreachable bei einem Netzfehler', async () => {
      const fetchFn = vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch'));
      const gateway = createHttpPositionChartGateway(fetchFn);

      const result = await gateway.fetchChart(CHART_POSITION_ID, '15m', 300, newSignal());

      expect(result).toEqual({ kind: 'problem', problemType: 'backend-unreachable' });
    });

    it('wirft den AbortError weiter, wenn der Aufrufer abbricht', async () => {
      const controller = new AbortController();
      const fetchFn = vi.fn<typeof fetch>().mockImplementation(() => {
        controller.abort();
        return Promise.reject(new DOMException('Aborted', 'AbortError'));
      });
      const gateway = createHttpPositionChartGateway(fetchFn);

      await expect(
        gateway.fetchChart(CHART_POSITION_ID, '15m', 300, controller.signal),
      ).rejects.toMatchObject({ name: 'AbortError' });
    });

    it('liefert backend-unreachable bei Timeout', async () => {
      const fetchFn = vi.fn<typeof fetch>().mockImplementation(
        (_url, init) =>
          new Promise<Response>((_resolve, reject) => {
            init?.signal?.addEventListener('abort', () =>
              reject(new DOMException('Timeout', 'TimeoutError')),
            );
          }),
      );
      const gateway = createHttpPositionChartGateway(fetchFn, '/api', { timeoutMs: 20 });

      const result = await gateway.fetchChart(CHART_POSITION_ID, '15m', 300, newSignal());

      expect(result).toEqual({ kind: 'problem', problemType: 'backend-unreachable' });
    });
  });
});
