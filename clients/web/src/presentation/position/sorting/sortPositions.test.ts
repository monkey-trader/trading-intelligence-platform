import { describe, expect, it, vi } from 'vitest';
import type { OpenPosition } from '@/domain/position/OpenPosition';
import { suiLongPosition } from '@/test/fixtures/openPositions';
import {
  nextSortState,
  sortPositions,
  type SortKey,
  type SortState,
} from '@/presentation/position/sorting/sortPositions';

function position(positionId: string, overrides: Partial<OpenPosition> = {}): OpenPosition {
  return { ...suiLongPosition, positionId, initialStopLoss: '0.80', ...overrides };
}

function ids(positions: readonly OpenPosition[]): string[] {
  return positions.map((entry) => entry.positionId);
}

function levelsOf(be: string, tp1: string): OpenPosition['levels'] {
  return [
    { kind: 'TWO_R', rMultiple: 2, price: be, percent: '1' },
    { kind: 'THREE_R', rMultiple: 3, price: tp1, percent: '1' },
  ];
}

describe('nextSortState', () => {
  it('erster Klick sortiert aufsteigend', () => {
    expect(nextSortState(null, 'entry')).toEqual({ key: 'entry', direction: 'asc' });
  });

  it('zweiter Klick auf denselben Schlüssel sortiert absteigend', () => {
    expect(nextSortState({ key: 'entry', direction: 'asc' }, 'entry')).toEqual({
      key: 'entry',
      direction: 'desc',
    });
  });

  it('dritter Klick hebt die Sortierung auf (Reihenfolge der API)', () => {
    expect(nextSortState({ key: 'entry', direction: 'desc' }, 'entry')).toBeNull();
  });

  it('anderer Schlüssel beginnt wieder aufsteigend', () => {
    expect(nextSortState({ key: 'entry', direction: 'desc' }, 'margin')).toEqual({
      key: 'margin',
      direction: 'asc',
    });
  });
});

describe('sortPositions', () => {
  const rows = [
    position('a', {
      symbol: 'BBB-USDT',
      positionValue: '300',
      margin: '30',
      unrealizedProfit: '5',
      realisedProfit: '-1',
      avgOpenPrice: '2',
      stopLoss: '1.5',
      levels: levelsOf('2.5', '3'),
    }),
    position('b', {
      symbol: 'CCC-USDT',
      positionValue: '1000',
      margin: '10',
      unrealizedProfit: '-20',
      realisedProfit: '4',
      avgOpenPrice: '10',
      stopLoss: '9',
      levels: levelsOf('12', '13'),
    }),
    position('c', {
      symbol: 'AAA-USDT',
      positionValue: '90',
      margin: '200',
      unrealizedProfit: '0.5',
      realisedProfit: '0',
      avgOpenPrice: '0.5',
      stopLoss: '0.4',
      levels: levelsOf('0.7', '0.8'),
    }),
  ];

  const expectedAscending: ReadonlyArray<[SortKey, string[]]> = [
    ['symbol', ['c', 'a', 'b']],
    ['value', ['c', 'a', 'b']],
    ['margin', ['b', 'a', 'c']],
    ['unrealizedProfit', ['b', 'c', 'a']],
    ['realisedProfit', ['a', 'c', 'b']],
    ['entry', ['c', 'a', 'b']],
    ['stopLoss', ['c', 'a', 'b']],
    ['be', ['c', 'a', 'b']],
    ['tp1', ['c', 'a', 'b']],
  ];

  it.each(expectedAscending)('sortiert nach %s aufsteigend', (key, expected) => {
    expect(ids(sortPositions(rows, { key, direction: 'asc' }))).toEqual(expected);
  });

  it.each(expectedAscending)('sortiert nach %s absteigend', (key, expected) => {
    expect(ids(sortPositions(rows, { key, direction: 'desc' }))).toEqual([...expected].reverse());
  });

  it('vergleicht Zahlen numerisch, nicht als Text ("9" vor "10")', () => {
    const numeric = [position('ten', { margin: '10' }), position('nine', { margin: '9' })];
    expect(ids(sortPositions(numeric, { key: 'margin', direction: 'asc' }))).toEqual([
      'nine',
      'ten',
    ]);
  });

  it('ohne Sortierung bleibt die Reihenfolge der API', () => {
    expect(ids(sortPositions(rows, null))).toEqual(['a', 'b', 'c']);
  });

  it('ändert die Eingabe nicht', () => {
    const input = [...rows];
    const state: SortState = { key: 'margin', direction: 'asc' };
    sortPositions(input, state);
    expect(ids(input)).toEqual(['a', 'b', 'c']);
  });

  it('ist stabil: gleiche Werte behalten die Reihenfolge der API', () => {
    const equal = [
      position('x', { margin: '5' }),
      position('y', { margin: '5' }),
      position('z', { margin: '1' }),
    ];
    expect(ids(sortPositions(equal, { key: 'margin', direction: 'asc' }))).toEqual(['z', 'x', 'y']);
    expect(ids(sortPositions(equal, { key: 'margin', direction: 'desc' }))).toEqual([
      'x',
      'y',
      'z',
    ]);
  });

  describe('Zeilen ohne SL (initialStopLoss unbekannt)', () => {
    const mixed = [
      position('no1', { initialStopLoss: null, margin: '1' }),
      position('k1', { margin: '50' }),
      position('no2', { initialStopLoss: null, margin: '99' }),
      position('k2', { margin: '20' }),
    ];

    it('stehen aufsteigend am Ende, untereinander in API-Reihenfolge', () => {
      expect(ids(sortPositions(mixed, { key: 'margin', direction: 'asc' }))).toEqual([
        'k2',
        'k1',
        'no1',
        'no2',
      ]);
    });

    it('stehen auch absteigend am Ende, untereinander in API-Reihenfolge', () => {
      expect(ids(sortPositions(mixed, { key: 'margin', direction: 'desc' }))).toEqual([
        'k1',
        'k2',
        'no1',
        'no2',
      ]);
    });

    it('gilt für jeden Sortierschlüssel', () => {
      const keys: SortKey[] = ['symbol', 'value', 'unrealizedProfit', 'entry', 'be', 'tp1'];
      for (const key of keys) {
        for (const direction of ['asc', 'desc'] as const) {
          const result = ids(sortPositions(mixed, { key, direction }));
          expect(result.slice(-2), `${key} ${direction}`).toEqual(['no1', 'no2']);
        }
      }
    });
  });
});

describe('sortPositions Symbol-Locale', () => {
  it('vergleicht Symbole mit fester Locale "de", nicht mit der Umgebungs-Locale', () => {
    const localeSpy = vi.spyOn(String.prototype, 'localeCompare');
    const collatorSpy = vi.spyOn(Intl, 'Collator');
    try {
      sortPositions(
        [position('x', { symbol: 'ÄAA-USDT' }), position('y', { symbol: 'ZZZ-USDT' })],
        { key: 'symbol', direction: 'asc' },
      );
      const usedLocale = [
        ...localeSpy.mock.calls.map((call) => call[1]),
        ...collatorSpy.mock.calls.map((call) => call[0]),
      ];
      expect(usedLocale.length).toBeGreaterThan(0);
      usedLocale.forEach((locale) => expect(String(locale)).toMatch(/^de/));
    } finally {
      localeSpy.mockRestore();
      collatorSpy.mockRestore();
    }
  });

  it('sortiert Umlaute deutsch: Ä steht bei A, vor Z', () => {
    const result = sortPositions(
      [position('z', { symbol: 'ZZZ-USDT' }), position('ae', { symbol: 'ÄAA-USDT' })],
      { key: 'symbol', direction: 'asc' },
    );
    expect(ids(result)).toEqual(['ae', 'z']);
  });
});
