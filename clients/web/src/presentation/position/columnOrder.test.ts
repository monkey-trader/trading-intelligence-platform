import { describe, expect, it } from 'vitest';
import {
  COLUMN_ORDER_STORAGE_KEY,
  DEFAULT_COLUMN_ORDER,
  isDefaultColumnOrder,
  moveColumn,
  sanitizeColumnOrder,
  type ColumnKey,
} from '@/presentation/position/columnOrder';

const DEFAULT = [...DEFAULT_COLUMN_ORDER];

describe('DEFAULT_COLUMN_ORDER', () => {
  it('hat die Standardreihenfolge mit 7 eindeutigen Spalten', () => {
    expect([...DEFAULT_COLUMN_ORDER]).toEqual([
      'symbol',
      'valueMargin',
      'unrealizedProfit',
      'realisedProfit',
      'entrySl',
      'be',
      'tp1',
    ]);
    expect(new Set(DEFAULT_COLUMN_ORDER).size).toBe(7);
  });
  it('Storage-Key ist v2', () => {
    expect(COLUMN_ORDER_STORAGE_KEY).toBe('tip_open_positions_column_order_v2');
  });
});

describe('moveColumn', () => {
  it('schiebt eine Spalte nach vorn auf die Position der Zielspalte', () => {
    const result = moveColumn(DEFAULT, 'entrySl', 'unrealizedProfit');
    expect(result.slice(0, 4)).toEqual(['symbol', 'valueMargin', 'entrySl', 'unrealizedProfit']);
    expect(result).toHaveLength(7);
  });
  it('schiebt eine Spalte nach hinten auf die Position der Zielspalte', () => {
    const result = moveColumn(DEFAULT, 'valueMargin', 'realisedProfit');
    expect(result.slice(0, 4)).toEqual([
      'symbol',
      'unrealizedProfit',
      'realisedProfit',
      'valueMargin',
    ]);
  });
  it('mutiert die Eingabe nicht', () => {
    const input = [...DEFAULT];
    moveColumn(input, 'entrySl', 'valueMargin');
    expect(input).toEqual(DEFAULT);
  });
  it('from === to ändert nichts', () => {
    expect(moveColumn(DEFAULT, 'entrySl', 'entrySl')).toEqual(DEFAULT);
  });
  it('symbol ist als Quelle nicht verschiebbar', () => {
    expect(moveColumn(DEFAULT, 'symbol', 'entrySl')).toEqual(DEFAULT);
  });
  it('symbol ist als Ziel nicht erlaubt: Symbol bleibt erste Spalte', () => {
    expect(moveColumn(DEFAULT, 'entrySl', 'symbol')).toEqual(DEFAULT);
  });
  it('unbekannte Keys ändern nichts', () => {
    expect(moveColumn(DEFAULT, 'nope' as ColumnKey, 'entrySl')).toEqual(DEFAULT);
  });
});

describe('sanitizeColumnOrder', () => {
  it('gültige vollständige Reihenfolge bleibt erhalten', () => {
    const custom = moveColumn(DEFAULT, 'entrySl', 'valueMargin');
    expect(sanitizeColumnOrder(custom)).toEqual(custom);
  });
  it.each([null, undefined, 'x', 42, {}, [], ['a', 1]])(
    'kaputte Eingabe %j -> Standard',
    (input) => {
      expect(sanitizeColumnOrder(input)).toEqual(DEFAULT);
    },
  );
  it('verwirft unbekannte Keys', () => {
    expect(sanitizeColumnOrder(['symbol', 'foo', ...DEFAULT.slice(1)])).toEqual(DEFAULT);
  });
  it('entfernt Duplikate (erstes Vorkommen zählt)', () => {
    const result = sanitizeColumnOrder(['symbol', 'entrySl', 'entrySl', 'valueMargin']);
    expect(result.filter((key) => key === 'entrySl')).toHaveLength(1);
    expect(result.slice(0, 3)).toEqual(['symbol', 'entrySl', 'valueMargin']);
    expect(result).toHaveLength(7);
  });
  it('hängt fehlende Keys in Standardposition-Reihenfolge an', () => {
    const result = sanitizeColumnOrder(['symbol', 'tp1', 'entrySl']);
    expect(result.slice(0, 3)).toEqual(['symbol', 'tp1', 'entrySl']);
    expect(result.slice(3)).toEqual(
      DEFAULT.filter((key) => !['symbol', 'tp1', 'entrySl'].includes(key)),
    );
  });
  it('ignoriert die entfernte Spalte rMultiple in alten gespeicherten Reihenfolgen', () => {
    const result = sanitizeColumnOrder(['symbol', 'rMultiple', 'entrySl']);
    expect(result).toHaveLength(7);
    expect(result.slice(0, 2)).toEqual(['symbol', 'entrySl']);
    expect(result).not.toContain('rMultiple');
  });
  it('erzwingt symbol als erste Spalte', () => {
    const result = sanitizeColumnOrder(['entrySl', 'symbol', 'valueMargin']);
    expect(result[0]).toBe('symbol');
    expect(result).toHaveLength(7);
  });
  it('ignoriert entfernte alte Keys (side, mark, openTime, breakEven, tpSl, beTarget) und ergänzt fehlende', () => {
    const result = sanitizeColumnOrder([
      'symbol',
      'side',
      'mark',
      'openTime',
      'breakEven',
      'tpSl',
      'beTarget',
      'tp1',
    ]);
    expect(result).toEqual([
      'symbol',
      'tp1',
      ...DEFAULT.filter((k) => k !== 'symbol' && k !== 'tp1'),
    ]);
  });
});

describe('isDefaultColumnOrder', () => {
  it('true für Standard, false nach Verschieben', () => {
    expect(isDefaultColumnOrder(DEFAULT)).toBe(true);
    expect(isDefaultColumnOrder(moveColumn(DEFAULT, 'entrySl', 'valueMargin'))).toBe(false);
  });
});
