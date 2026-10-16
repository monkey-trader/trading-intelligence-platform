import { describe, expect, it, vi } from 'vitest';
import type { components } from '@/infrastructure/api/generated';
import { createHttpOpenPositionsGateway } from '@/infrastructure/api/httpOpenPositionsGateway';

type SnapshotBody = components['schemas']['OpenPositionsSnapshot'];
type ProblemBody = components['schemas']['Problem'];

// Antwort von GET /positions/open laut docs/openapi.yaml 0.3.0 (erfundene Werte).
const snapshotBody: SnapshotBody = {
  positions: [
    {
      positionId: '9100000000000000002',
      symbol: 'NCCOGOLD2USD-USDT',
      side: 'SHORT',
      isolated: true,
      positionAmount: '0.25',
      avgOpenPrice: '4150.00',
      markPrice: '4175.00',
      leverage: 15,
      initialMargin: '69.1667',
      margin: '65.0000',
      positionValue: '1043.75',
      unrealizedProfit: '-6.2500',
      realisedProfit: '-0.5187',
      pnlRatio: '-0.0904',
      liquidationPrice: '4410.5',
      liquidationDistanceRatio: '0.05640719',
      openTime: 1791700000000,
      updateTime: 1791710000000,
    },
    {
      positionId: '9100000000000000001',
      symbol: 'SUI-USDT',
      side: 'LONG',
      isolated: true,
      positionAmount: '500',
      avgOpenPrice: '0.9600',
      markPrice: '1.0100',
      leverage: 10,
      initialMargin: '48.0000',
      margin: '60.5000',
      positionValue: '505',
      unrealizedProfit: '25.0000',
      realisedProfit: '-0.2400',
      pnlRatio: '0.5208',
      liquidationPrice: '0',
      liquidationDistanceRatio: null,
      openTime: 1791500000000,
      updateTime: 1791600000000,
    },
  ],
  totalUnrealizedProfit: '18.75',
  fetchedAt: 1792154096000,
};

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

describe('createHttpOpenPositionsGateway', () => {
  describe('Anfrage', () => {
    it('fragt GET /api/positions/open ab, ein Abbruch von außen erreicht den Request', async () => {
      const fetchFn = fakeFetchReturning(jsonResponse(snapshotBody));
      const gateway = createHttpOpenPositionsGateway(fetchFn);
      const controller = new AbortController();

      await gateway.fetchOpenPositions(controller.signal);

      expect(fetchFn).toHaveBeenCalledTimes(1);
      const [requestUrl, requestInit] = fetchFn.mock.calls[0] ?? [];
      expect(String(requestUrl)).toBe('/api/positions/open');
      expect(requestInit?.method ?? 'GET').toBe('GET');
      expect(requestInit?.signal?.aborted).toBe(false);
      controller.abort();
      expect(requestInit?.signal?.aborted).toBe(true);
    });

    it('verwendet eine abweichende Basis-URL', async () => {
      const fetchFn = fakeFetchReturning(jsonResponse(snapshotBody));
      const gateway = createHttpOpenPositionsGateway(fetchFn, 'http://127.0.0.1:3000');

      await gateway.fetchOpenPositions(newSignal());

      expect(String(fetchFn.mock.calls[0]?.[0])).toBe('http://127.0.0.1:3000/positions/open');
    });
  });

  describe('Erfolg (200)', () => {
    it('liefert den Snapshot mit allen Feldern, Dezimalwerte bleiben Strings', async () => {
      const gateway = createHttpOpenPositionsGateway(
        fakeFetchReturning(jsonResponse(snapshotBody)),
      );

      const result = await gateway.fetchOpenPositions(newSignal());

      expect(result).toEqual({ kind: 'ok', snapshot: snapshotBody });
    });

    it('übernimmt liquidationDistanceRatio null unverändert', async () => {
      const gateway = createHttpOpenPositionsGateway(
        fakeFetchReturning(jsonResponse(snapshotBody)),
      );

      const result = await gateway.fetchOpenPositions(newSignal());

      if (result.kind !== 'ok') throw new Error(`erwartet ok, erhalten ${result.kind}`);
      expect(result.snapshot.positions[1]?.liquidationDistanceRatio).toBeNull();
      expect(result.snapshot.positions[0]?.liquidationDistanceRatio).toBe('0.05640719');
    });

    it('liefert eine leere Liste, wenn keine Position offen ist', async () => {
      const emptyBody: SnapshotBody = {
        positions: [],
        totalUnrealizedProfit: '0',
        fetchedAt: 1792154096000,
      };
      const gateway = createHttpOpenPositionsGateway(fakeFetchReturning(jsonResponse(emptyBody)));

      const result = await gateway.fetchOpenPositions(newSignal());

      expect(result).toEqual({ kind: 'ok', snapshot: emptyBody });
    });
  });

  describe('Problem-Antworten (application/problem+json)', () => {
    const problemCases: readonly ProblemBody[] = [
      { type: 'forbidden-host', title: 'Host nicht erlaubt', status: 403 },
      { type: 'not-configured', title: 'BingX nicht konfiguriert', status: 503 },
      { type: 'rate-limited', title: 'BingX ausgelastet', status: 503 },
      { type: 'unreachable', title: 'BingX nicht erreichbar', status: 504 },
      { type: 'upstream-error', title: 'BingX meldet einen Fehler', status: 502 },
      { type: 'invalid-response', title: 'Unerwartete Antwort', status: 502 },
      { type: 'internal', title: 'Interner Fehler', status: 500 },
    ];

    it.each(problemCases)('liefert problem $type bei Status $status', async (problemBody) => {
      const gateway = createHttpOpenPositionsGateway(
        fakeFetchReturning(problemResponse(problemBody)),
      );

      const result = await gateway.fetchOpenPositions(newSignal());

      expect(result).toEqual({ kind: 'problem', problemType: problemBody.type });
      expect(result).not.toHaveProperty('retryAfterSeconds');
    });

    it('übernimmt Retry-After in Sekunden bei rate-limited', async () => {
      const gateway = createHttpOpenPositionsGateway(
        fakeFetchReturning(
          problemResponse(
            { type: 'rate-limited', title: 'BingX ausgelastet', status: 503 },
            { 'Retry-After': '30' },
          ),
        ),
      );

      const result = await gateway.fetchOpenPositions(newSignal());

      expect(result).toEqual({
        kind: 'problem',
        problemType: 'rate-limited',
        retryAfterSeconds: 30,
      });
    });

    it('ignoriert einen unlesbaren Retry-After-Header', async () => {
      const gateway = createHttpOpenPositionsGateway(
        fakeFetchReturning(
          problemResponse(
            { type: 'rate-limited', title: 'BingX ausgelastet', status: 503 },
            { 'Retry-After': 'bald' },
          ),
        ),
      );

      const result = await gateway.fetchOpenPositions(newSignal());

      expect(result).toEqual({ kind: 'problem', problemType: 'rate-limited' });
      expect(result).not.toHaveProperty('retryAfterSeconds');
    });

    it('liefert invalid-body bei einem unbekannten Problem-Typ', async () => {
      const response = new Response(
        JSON.stringify({ type: 'something-else', title: 'Unbekannt', status: 500 }),
        { status: 500, headers: { 'Content-Type': 'application/problem+json' } },
      );
      const gateway = createHttpOpenPositionsGateway(fakeFetchReturning(response));

      const result = await gateway.fetchOpenPositions(newSignal());

      expect(result).toEqual({ kind: 'problem', problemType: 'invalid-body' });
    });

    // Entscheidung: Eine Fehlerantwort ohne problem+json stammt nicht von unserem Backend, sondern
    // z.B. vom Vite-Proxy, wenn das Backend nicht läuft. Der hilfreiche Hinweis ist dann "Backend starten".
    it.each([
      [502, 'text/html', '<html>Bad Gateway</html>'],
      [500, 'text/plain', ''],
      [504, 'application/json', '{"message":"timeout"}'],
    ])(
      'liefert backend-unreachable bei einer Fehlerantwort %i ohne Problem-Body (%s)',
      async (status, contentType, body) => {
        const response = new Response(body, { status, headers: { 'Content-Type': contentType } });
        const gateway = createHttpOpenPositionsGateway(fakeFetchReturning(response));

        const result = await gateway.fetchOpenPositions(newSignal());

        expect(result).toEqual({ kind: 'problem', problemType: 'backend-unreachable' });
      },
    );
  });

  describe('Netz- und Formatfehler', () => {
    it('liefert backend-unreachable bei einem Netzfehler (TypeError)', async () => {
      const fetchFn = vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch'));
      const gateway = createHttpOpenPositionsGateway(fetchFn);

      const result = await gateway.fetchOpenPositions(newSignal());

      expect(result).toEqual({ kind: 'problem', problemType: 'backend-unreachable' });
    });

    it('liefert invalid-body bei kaputtem JSON', async () => {
      const response = new Response('{"positions": [', {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      });
      const gateway = createHttpOpenPositionsGateway(fakeFetchReturning(response));

      const result = await gateway.fetchOpenPositions(newSignal());

      expect(result).toEqual({ kind: 'problem', problemType: 'invalid-body' });
    });

    const wrongShapes: ReadonlyArray<readonly [string, unknown]> = [
      ['ein Array statt der Hülle', [snapshotBody.positions[0]]],
      ['null', null],
      ['positions ist keine Liste', { ...snapshotBody, positions: 'keine Liste' }],
      ['totalUnrealizedProfit fehlt', { positions: [], fetchedAt: 1792154096000 }],
      ['totalUnrealizedProfit ist eine Zahl', { ...snapshotBody, totalUnrealizedProfit: 18.75 }],
      ['fetchedAt ist keine Zahl', { ...snapshotBody, fetchedAt: '2026-10-16' }],
      ['eine Position ist null', { ...snapshotBody, positions: [null] }],
      [
        'eine Position ohne positionId',
        {
          ...snapshotBody,
          positions: [{ ...snapshotBody.positions[0], positionId: undefined }],
        },
      ],
      [
        'eine Position mit unbekannter Seite',
        { ...snapshotBody, positions: [{ ...snapshotBody.positions[0], side: 'BOTH' }] },
      ],
      [
        'unrealizedProfit ist eine Zahl',
        { ...snapshotBody, positions: [{ ...snapshotBody.positions[0], unrealizedProfit: 25 }] },
      ],
    ];

    it.each(wrongShapes)('liefert invalid-body bei falscher Form: %s', async (_label, body) => {
      const gateway = createHttpOpenPositionsGateway(fakeFetchReturning(jsonResponse(body)));

      const result = await gateway.fetchOpenPositions(newSignal());

      expect(result).toEqual({ kind: 'problem', problemType: 'invalid-body' });
    });
  });

  describe('Abbruch', () => {
    it('wirft den AbortError weiter, statt ein Ergebnis zu liefern', async () => {
      const abortError = new DOMException('The operation was aborted.', 'AbortError');
      const fetchFn = vi.fn<typeof fetch>().mockRejectedValue(abortError);
      const gateway = createHttpOpenPositionsGateway(fetchFn);
      const controller = new AbortController();
      controller.abort();

      await expect(gateway.fetchOpenPositions(controller.signal)).rejects.toMatchObject({
        name: 'AbortError',
      });
    });
  });
});

describe('createHttpOpenPositionsGateway – Timeout', () => {
  it('liefert backend-unreachable, wenn das Backend nicht innerhalb des Timeouts antwortet', async () => {
    const hangingFetch: typeof fetch = (_input, init) =>
      new Promise<Response>((_resolve, reject) => {
        init?.signal?.addEventListener('abort', () => {
          reject(init.signal?.reason ?? new DOMException('Zeitüberschreitung', 'TimeoutError'));
        });
      });
    const gateway = createHttpOpenPositionsGateway(hangingFetch, '/api', { timeoutMs: 20 });

    const result = await gateway.fetchOpenPositions(new AbortController().signal);

    expect(result).toEqual({ kind: 'problem', problemType: 'backend-unreachable' });
  });
});
