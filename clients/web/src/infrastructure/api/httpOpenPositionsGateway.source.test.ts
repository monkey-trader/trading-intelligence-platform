import { describe, expect, it, vi } from 'vitest';
import { createHttpOpenPositionsGateway } from '@/infrastructure/api/httpOpenPositionsGateway';
import { suiLongPosition } from '@/test/fixtures/openPositions';

// Erwartet: OpenPosition.initialStopLossSource: 'AUTO' | 'MANUAL' | null; fehlend oder unbekannter Wert = null.

async function fetchWith(extra: Record<string, unknown>, removeKey = false) {
  const { initialStopLossSource, ...base } = suiLongPosition;
  void initialStopLossSource;
  const position = removeKey ? base : { ...base, ...extra };
  const body = { positions: [position], totalUnrealizedProfit: '25', fetchedAt: 1792154096000 };
  const fetchFn = vi.fn<typeof fetch>().mockResolvedValue(
    new Response(JSON.stringify(body), {
      status: 200,
      headers: { 'Content-Type': 'application/json; charset=utf-8' },
    }),
  );
  const result = await createHttpOpenPositionsGateway(fetchFn).fetchOpenPositions(
    new AbortController().signal,
  );
  if (result.kind !== 'ok') throw new Error(`erwartet ok, erhalten ${result.kind}`);
  return result.snapshot.positions[0];
}

describe('httpOpenPositionsGateway initialStopLossSource', () => {
  it.each(['AUTO', 'MANUAL'] as const)('übernimmt %s', async (source) => {
    expect((await fetchWith({ initialStopLossSource: source }))?.initialStopLossSource).toBe(
      source,
    );
  });

  it('übernimmt null', async () => {
    expect((await fetchWith({ initialStopLossSource: null }))?.initialStopLossSource).toBeNull();
  });

  it('fehlendes Feld ergibt null', async () => {
    expect((await fetchWith({}, true))?.initialStopLossSource).toBeNull();
  });

  it('unbekannter Wert ergibt null', async () => {
    expect((await fetchWith({ initialStopLossSource: 'BOTH' }))?.initialStopLossSource).toBeNull();
  });
});
