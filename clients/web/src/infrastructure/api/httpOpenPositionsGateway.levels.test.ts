import { describe, expect, it, vi } from 'vitest';
import { createHttpOpenPositionsGateway } from '@/infrastructure/api/httpOpenPositionsGateway';
import { suiLongPosition } from '@/test/fixtures/openPositions';

// Erwartet: levels (optional) je Position wird als { kind, rMultiple, price, percent }[] übernommen;
// fehlend, leer oder ungültig = undefined.

const levelsBody = [
  { kind: 'TWO_R', rMultiple: 2, price: '1.0800', percent: '12.5' },
  { kind: 'THREE_R', rMultiple: 3, price: '1.1400', percent: '18.75' },
];

async function fetchWith(levels: unknown) {
  const {
    initialStopLoss,
    riskDistance,
    currentRMultiple,
    breakEvenTriggerReached,
    reached2RAt,
    ...base
  } = suiLongPosition;
  void [initialStopLoss, riskDistance, currentRMultiple, breakEvenTriggerReached, reached2RAt];
  const body = {
    positions: [levels === undefined ? base : { ...base, levels }],
    totalUnrealizedProfit: '25',
    fetchedAt: 1792154096000,
  };
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

describe('httpOpenPositionsGateway levels', () => {
  it('verwirft ONE_R und liefert nur die gültigen Level', async () => {
    const position = await fetchWith([
      { kind: 'ONE_R', rMultiple: 1, price: '1.0200', percent: '6.25' },
      ...levelsBody,
    ]);
    expect(position?.levels).toEqual(levelsBody);
  });

  it('übernimmt coincidesWithTakeProfit nicht (nur im Chart-Vertrag)', async () => {
    const position = await fetchWith([{ ...levelsBody[1], coincidesWithTakeProfit: true }]);
    expect(position?.levels?.[0]).not.toHaveProperty('coincidesWithTakeProfit');
  });

  it('übernimmt beide Level unverändert (Preise bleiben Strings)', async () => {
    const position = await fetchWith(levelsBody);
    expect(position?.levels).toEqual(levelsBody);
  });

  it.each([
    ['fehlend', undefined],
    ['leer', []],
  ])('levels %s ergibt undefined', async (_name, value) => {
    const position = await fetchWith(value);
    expect(position?.levels).toBeUndefined();
  });
});
