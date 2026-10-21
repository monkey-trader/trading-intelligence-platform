import { describe, expect, it, vi } from 'vitest';
import { createHttpAccountSummaryGateway } from '@/infrastructure/api/httpAccountSummaryGateway';

const BODY = {
  balance: '1234.56',
  equity: '1300.10',
  availableMargin: '1000.00',
  month: '2026-10',
  monthlyRealizedPnl: '61.73',
  monthlyPnlPercent: '5.00',
  periods: [
    { period: 'today', realizedPnl: '33.82', pnlPercent: '1.59' },
    { period: '7d', realizedPnl: '-12.5', pnlPercent: null },
    { period: '30d', realizedPnl: '61.73', pnlPercent: '2.9' },
  ],
  unrealizedPnl: '125.08',
  unrealizedPnlPercent: '5.79',
};

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

describe('createHttpAccountSummaryGateway', () => {
  it('liest GET /account/summary und mappt die Felder', async () => {
    const fetchFn = vi.fn().mockResolvedValue(jsonResponse(BODY));
    const gateway = createHttpAccountSummaryGateway(fetchFn, '/api');

    const summary = await gateway.fetchAccountSummary(new AbortController().signal);

    expect(fetchFn.mock.calls[0]?.[0]).toBe('/api/account/summary');
    expect(summary).toEqual({
      ...BODY,
      periods: [
        { period: 'today', realizedPnl: '33.82', pnlPercent: '1.59' },
        { period: '7d', realizedPnl: '-12.5', pnlPercent: undefined },
        { period: '30d', realizedPnl: '61.73', pnlPercent: '2.9' },
      ],
    });
  });

  it('mappt monthlyPnlPercent null auf undefined', async () => {
    const fetchFn = vi.fn().mockResolvedValue(jsonResponse({ ...BODY, monthlyPnlPercent: null }));
    const gateway = createHttpAccountSummaryGateway(fetchFn);

    const summary = await gateway.fetchAccountSummary(new AbortController().signal);

    expect(summary?.monthlyPnlPercent).toBeUndefined();
  });

  it('liefert undefined bei HTTP-Fehler, kaputtem Body und Netzwerkfehler', async () => {
    const signal = new AbortController().signal;
    const failing = [
      vi.fn().mockResolvedValue(jsonResponse({ type: 'upstream-error' }, 502)),
      vi.fn().mockResolvedValue(jsonResponse({ ...BODY, balance: 'abc' })),
      vi.fn().mockResolvedValue(new Response('kein json', { status: 200 })),
      vi.fn().mockRejectedValue(new TypeError('network')),
    ];
    for (const fetchFn of failing) {
      expect(await createHttpAccountSummaryGateway(fetchFn).fetchAccountSummary(signal)).toBe(
        undefined,
      );
    }
  });

  it('wirft einen Abbruch durch den Aufrufer weiter', async () => {
    const controller = new AbortController();
    controller.abort();
    const fetchFn = vi.fn().mockRejectedValue(new DOMException('abort', 'AbortError'));

    await expect(
      createHttpAccountSummaryGateway(fetchFn).fetchAccountSummary(controller.signal),
    ).rejects.toMatchObject({ name: 'AbortError' });
  });
});
