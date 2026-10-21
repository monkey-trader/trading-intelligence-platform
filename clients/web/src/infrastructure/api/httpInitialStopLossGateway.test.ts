import { describe, expect, it, vi } from 'vitest';
import { createHttpInitialStopLossGateway } from '@/infrastructure/api/httpInitialStopLossGateway';

// Erwartet (Port src/domain/position/InitialStopLossGateway.ts):
//   InitialStopLossGateway = {
//     setInitialStopLoss(positionId: string, value: string): Promise<InitialStopLossResult>;
//     clearInitialStopLoss(positionId: string): Promise<InitialStopLossResult>;
//   }
//   InitialStopLossResult = { kind: 'ok' } | { kind: 'problem'; problemType: InitialStopLossProblemType }
//   InitialStopLossProblemType = 'invalid-stop-loss' | 'not-found' | 'bad-request' | 'forbidden-origin' |
//     'not-configured' | 'rate-limited' | 'unreachable' | 'upstream-error' | 'invalid-response' | 'internal' |
//     'backend-unreachable' | 'invalid-body'
// Adapter: createHttpInitialStopLossGateway(fetchFn = fetch, baseUrl = '/api').

const POSITION_ID = '9100000000000000001';
const URL_PATH = `/api/positions/${POSITION_ID}/initial-stop-loss`;

const okBody = {
  positionId: POSITION_ID,
  initialStopLoss: '0.9000',
  entryPrice: '0.9600',
  riskDistance: '0.06',
  source: 'MANUAL',
  setAt: 1791800000000,
};

function problem(type: string, status: number): Response {
  return new Response(JSON.stringify({ type, title: 't', status }), {
    status,
    headers: { 'Content-Type': 'application/problem+json; charset=utf-8' },
  });
}

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json; charset=utf-8' },
  });
}

describe('createHttpInitialStopLossGateway.setInitialStopLoss', () => {
  it('sendet PUT mit JSON-Body und Content-Type', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(json(okBody));

    const result = await createHttpInitialStopLossGateway(fetchFn).setInitialStopLoss(
      POSITION_ID,
      '0.9000',
    );

    expect(result).toEqual({ kind: 'ok' });
    const [url, init] = fetchFn.mock.calls[0] ?? [];
    expect(String(url)).toBe(URL_PATH);
    expect(init?.method).toBe('PUT');
    expect(new Headers(init?.headers).get('Content-Type')).toBe('application/json');
    expect(JSON.parse(String(init?.body))).toEqual({ initialStopLoss: '0.9000' });
  });

  it('kodiert die positionId in der URL', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(json(okBody));
    await createHttpInitialStopLossGateway(fetchFn).setInitialStopLoss('a/b', '1');
    expect(String(fetchFn.mock.calls[0]?.[0])).toBe('/api/positions/a%2Fb/initial-stop-loss');
  });

  it.each([
    ['invalid-stop-loss', 422],
    ['not-found', 404],
    ['bad-request', 400],
    ['forbidden-origin', 403],
    ['rate-limited', 503],
    ['unreachable', 504],
    ['upstream-error', 502],
  ])('bildet %s (%i) auf ein Problem ab', async (type, status) => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(problem(type, status));
    const result = await createHttpInitialStopLossGateway(fetchFn).setInitialStopLoss(
      POSITION_ID,
      '1',
    );
    expect(result).toEqual({ kind: 'problem', problemType: type });
  });

  it('Netzwerkfehler ergibt backend-unreachable', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch'));
    const result = await createHttpInitialStopLossGateway(fetchFn).setInitialStopLoss(
      POSITION_ID,
      '1',
    );
    expect(result).toEqual({ kind: 'problem', problemType: 'backend-unreachable' });
  });

  it('Fehlerstatus ohne problem+json ergibt backend-unreachable', async () => {
    const fetchFn = vi
      .fn<typeof fetch>()
      .mockResolvedValue(new Response('<html>', { status: 502 }));
    const result = await createHttpInitialStopLossGateway(fetchFn).setInitialStopLoss(
      POSITION_ID,
      '1',
    );
    expect(result).toEqual({ kind: 'problem', problemType: 'backend-unreachable' });
  });

  it('200 mit unpassendem Body ergibt invalid-body', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(json({ foo: 1 }));
    const result = await createHttpInitialStopLossGateway(fetchFn).setInitialStopLoss(
      POSITION_ID,
      '1',
    );
    expect(result).toEqual({ kind: 'problem', problemType: 'invalid-body' });
  });
});

describe('createHttpInitialStopLossGateway.clearInitialStopLoss', () => {
  it('sendet DELETE ohne Body und liefert bei 204 ok', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 204 }));

    const result =
      await createHttpInitialStopLossGateway(fetchFn).clearInitialStopLoss(POSITION_ID);

    expect(result).toEqual({ kind: 'ok' });
    const [url, init] = fetchFn.mock.calls[0] ?? [];
    expect(String(url)).toBe(URL_PATH);
    expect(init?.method).toBe('DELETE');
    expect(init?.body ?? undefined).toBeUndefined();
  });

  it('bildet Problem-Antworten ab', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(problem('forbidden-origin', 403));
    const result =
      await createHttpInitialStopLossGateway(fetchFn).clearInitialStopLoss(POSITION_ID);
    expect(result).toEqual({ kind: 'problem', problemType: 'forbidden-origin' });
  });

  it('Netzwerkfehler ergibt backend-unreachable', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch'));
    const result =
      await createHttpInitialStopLossGateway(fetchFn).clearInitialStopLoss(POSITION_ID);
    expect(result).toEqual({ kind: 'problem', problemType: 'backend-unreachable' });
  });
});
