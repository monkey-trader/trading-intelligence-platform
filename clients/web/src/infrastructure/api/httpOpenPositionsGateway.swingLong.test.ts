import { describe, expect, it, vi } from 'vitest';
import { createHttpOpenPositionsGateway } from '@/infrastructure/api/httpOpenPositionsGateway';
import { suiLongPosition, swingLongPosition } from '@/test/fixtures/openPositions';

// Vertrag: OpenPosition.swingLong (boolean) und OpenPositionsSnapshot.swingLongUnrealizedProfit (Decimal-String).
// Der Client rechnet nicht: Summen kommen unverändert vom Backend.

async function fetchBody(body: unknown) {
  const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(
    new Response(JSON.stringify(body), {
      status: 200,
      headers: { 'Content-Type': 'application/json; charset=utf-8' },
    }),
  );
  return createHttpOpenPositionsGateway(fetchFn).fetchOpenPositions(new AbortController().signal);
}

const body = {
  positions: [suiLongPosition, swingLongPosition],
  totalUnrealizedProfit: '25',
  swingLongUnrealizedProfit: '40.5',
  fetchedAt: 1792154096000,
};

describe('httpOpenPositionsGateway Swing Long', () => {
  it('übernimmt swingLong je Position und die getrennte Summe unverändert', async () => {
    const result = await fetchBody(body);

    expect(result.kind).toBe('ok');
    if (result.kind !== 'ok') return;
    expect(result.snapshot.positions.map((position) => position.swingLong)).toEqual([false, true]);
    expect(result.snapshot.totalUnrealizedProfit).toBe('25');
    expect(result.snapshot.swingLongUnrealizedProfit).toBe('40.5');
  });

  it.each([
    ['swingLong fehlt', { ...body, positions: [{ ...suiLongPosition, swingLong: undefined }] }],
    [
      'swingLong ist ein String',
      { ...body, positions: [{ ...suiLongPosition, swingLong: 'true' }] },
    ],
    ['swingLongUnrealizedProfit fehlt', { ...body, swingLongUnrealizedProfit: undefined }],
    ['swingLongUnrealizedProfit ist eine Zahl', { ...body, swingLongUnrealizedProfit: 40.5 }],
  ])('%s ergibt invalid-body', async (_name, wrong) => {
    expect(await fetchBody(wrong)).toEqual({ kind: 'problem', problemType: 'invalid-body' });
  });
});
