import { describe, expect, it, vi } from 'vitest';
import { createHttpAccountSummaryGateway } from '@/infrastructure/api/httpAccountSummaryGateway';

const BODY = {
  balance: '1234.56',
  equity: '1300.10',
  availableMargin: '1000.00',
  month: '2026-10',
  monthlyRealizedPnl: '61.73',
  monthlyPnlPercent: '5.00',
  periods: [{ period: 'today', realizedPnl: '33.82', pnlPercent: '1.59' }],
  unrealizedPnl: '125.08',
  unrealizedPnlPercent: '5.79',
  swingLongUnrealizedPnl: '40.5',
  swingLongRealizedPnl: '-12.25',
};

function respond(body: unknown) {
  return vi.fn().mockResolvedValue(
    new Response(JSON.stringify(body), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    }),
  );
}

describe('httpAccountSummaryGateway Swing Long', () => {
  it('übernimmt die Swing-Long-Werte unverändert', async () => {
    const summary = await createHttpAccountSummaryGateway(respond(BODY)).fetchAccountSummary(
      new AbortController().signal,
    );

    expect(summary?.swingLongUnrealizedPnl).toBe('40.5');
    expect(summary?.swingLongRealizedPnl).toBe('-12.25');
  });

  it.each(['swingLongUnrealizedPnl', 'swingLongRealizedPnl'])(
    'liefert undefined, wenn %s fehlt oder kein Dezimal-String ist',
    async (key) => {
      const signal = new AbortController().signal;
      const missing = { ...BODY, [key]: undefined };
      const invalid = { ...BODY, [key]: 'abc' };

      expect(
        await createHttpAccountSummaryGateway(respond(missing)).fetchAccountSummary(signal),
      ).toBeUndefined();
      expect(
        await createHttpAccountSummaryGateway(respond(invalid)).fetchAccountSummary(signal),
      ).toBeUndefined();
    },
  );
});
