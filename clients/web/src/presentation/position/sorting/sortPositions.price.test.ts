import { describe, expect, it } from 'vitest';
import type { OpenPosition } from '@/domain/position/OpenPosition';
import type { PositionPrice } from '@/domain/price/PriceStream';
import { suiLongPosition } from '@/test/fixtures/openPositions';
import { nextSortState, sortPositions } from '@/presentation/position/sorting/sortPositions';

const position = (positionId: string): OpenPosition => ({
  ...suiLongPosition,
  positionId,
  initialStopLoss: '0.80',
});
const priceOf = (positionId: string, price: string): PositionPrice => ({
  positionId,
  symbol: 'X-USDT',
  price,
  change24hPercent: '0',
  distanceToBreakEvenPercent: null,
  distanceToTp1Percent: null,
});
const prices = { a: priceOf('a', '3'), b: priceOf('b', '0.0000123'), c: priceOf('c', '100') };
const ids = (list: readonly OpenPosition[]) => list.map((entry) => entry.positionId);
const rows = [position('a'), position('b'), position('c'), position('d')];

describe('Sortierung nach Kurs', () => {
  it('Zyklus aufsteigend, absteigend, aus', () => {
    const first = nextSortState(null, 'price');
    expect(first).toEqual({ key: 'price', direction: 'asc' });
    const second = nextSortState(first, 'price');
    expect(second).toEqual({ key: 'price', direction: 'desc' });
    expect(nextSortState(second, 'price')).toBeNull();
  });
  it('aufsteigend; Position ohne Kurs (d) steht am Ende', () => {
    expect(ids(sortPositions(rows, { key: 'price', direction: 'asc' }, prices))).toEqual([
      'b',
      'a',
      'c',
      'd',
    ]);
  });
  it('absteigend; unbekannter Kurs bleibt am Ende', () => {
    expect(ids(sortPositions(rows, { key: 'price', direction: 'desc' }, prices))).toEqual([
      'c',
      'a',
      'b',
      'd',
    ]);
  });
  it('ohne Kurse bleibt die API-Reihenfolge', () => {
    expect(ids(sortPositions(rows, { key: 'price', direction: 'asc' }, {}))).toEqual([
      'a',
      'b',
      'c',
      'd',
    ]);
  });
});
