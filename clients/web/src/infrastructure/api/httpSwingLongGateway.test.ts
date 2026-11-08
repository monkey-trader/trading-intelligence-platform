import { describe, expect, it, vi } from 'vitest';
import { createHttpSwingLongGateway } from '@/infrastructure/api/httpSwingLongGateway';

// Port: src/domain/position/SwingLongGateway.ts (setSwingLong / clearSwingLong -> SwingLongResult).
// Adapter: createHttpSwingLongGateway(fetchFn = fetch, baseUrl = '/api'); PUT/DELETE, Antwort 204.

const POSITION_ID = '9100000000000000004';
const URL_PATH = `/api/positions/${POSITION_ID}/swing-long`;

function problem(type: string, status: number): Response {
  return new Response(JSON.stringify({ type, title: 't', status }), {
    status,
    headers: { 'Content-Type': 'application/problem+json; charset=utf-8' },
  });
}

describe('createHttpSwingLongGateway.setSwingLong', () => {
  it('sendet PUT ohne Body und liefert bei 204 ok', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 204 }));

    const result = await createHttpSwingLongGateway(fetchFn).setSwingLong(POSITION_ID);

    expect(result).toEqual({ kind: 'ok' });
    const [url, init] = fetchFn.mock.calls[0] ?? [];
    expect(String(url)).toBe(URL_PATH);
    expect(init?.method).toBe('PUT');
    expect(init?.body ?? undefined).toBeUndefined();
  });

  it('kodiert die positionId in der URL und nutzt die baseUrl', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 204 }));
    await createHttpSwingLongGateway(fetchFn, '/x').setSwingLong('a/b');
    expect(String(fetchFn.mock.calls[0]?.[0])).toBe('/x/positions/a%2Fb/swing-long');
  });

  it.each([
    ['bad-request', 400],
    ['forbidden-origin', 403],
    ['internal', 500],
  ])('bildet %s (%i) auf ein Problem ab', async (type, status) => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(problem(type, status));
    const result = await createHttpSwingLongGateway(fetchFn).setSwingLong(POSITION_ID);
    expect(result).toEqual({ kind: 'problem', problemType: type });
  });

  it('unbekannter Problem-Typ ergibt invalid-body', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(problem('seltsam', 418));
    const result = await createHttpSwingLongGateway(fetchFn).setSwingLong(POSITION_ID);
    expect(result).toEqual({ kind: 'problem', problemType: 'invalid-body' });
  });

  it('Fehlerstatus ohne problem+json ergibt backend-unreachable', async () => {
    const fetchFn = vi
      .fn<typeof fetch>()
      .mockResolvedValue(new Response('<html>', { status: 502 }));
    const result = await createHttpSwingLongGateway(fetchFn).setSwingLong(POSITION_ID);
    expect(result).toEqual({ kind: 'problem', problemType: 'backend-unreachable' });
  });

  it('Netzwerkfehler ergibt backend-unreachable', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch'));
    const result = await createHttpSwingLongGateway(fetchFn).setSwingLong(POSITION_ID);
    expect(result).toEqual({ kind: 'problem', problemType: 'backend-unreachable' });
  });
});

describe('createHttpSwingLongGateway.clearSwingLong', () => {
  it('sendet DELETE und liefert bei 204 ok', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 204 }));

    const result = await createHttpSwingLongGateway(fetchFn).clearSwingLong(POSITION_ID);

    expect(result).toEqual({ kind: 'ok' });
    const [url, init] = fetchFn.mock.calls[0] ?? [];
    expect(String(url)).toBe(URL_PATH);
    expect(init?.method).toBe('DELETE');
  });

  it('bildet Problem-Antworten ab', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(problem('forbidden-origin', 403));
    const result = await createHttpSwingLongGateway(fetchFn).clearSwingLong(POSITION_ID);
    expect(result).toEqual({ kind: 'problem', problemType: 'forbidden-origin' });
  });

  it('Netzwerkfehler ergibt backend-unreachable', async () => {
    const fetchFn = vi.fn<typeof fetch>().mockRejectedValue(new TypeError('Failed to fetch'));
    const result = await createHttpSwingLongGateway(fetchFn).clearSwingLong(POSITION_ID);
    expect(result).toEqual({ kind: 'problem', problemType: 'backend-unreachable' });
  });
});
