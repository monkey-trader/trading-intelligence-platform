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
  it('hat die BingX-Stil-Reihenfolge mit 8 eindeutigen Spalten', () => {
    expect([...DEFAULT_COLUMN_ORDER]).toEqual([
      'symbol',
      'positionValue',
      'margin',
      'unrealizedProfit',
      'realisedProfit',
      'entry',
      'tpSl',
      'beTarget',
    ]);
    expect(new Set(DEFAULT_COLUMN_ORDER).size).toBe(8);
  });
  it('Storage-Key ist v2', () => {
    expect(COLUMN_ORDER_STORAGE_KEY).toBe('tip_open_positions_column_order_v2');
  });
});

describe('moveColumn', () => {
  it('schiebt eine Spalte nach vorn auf die Position der Zielspalte', () => {
    const result = moveColumn(DEFAULT, 'entry', 'margin');
    expect(result.slice(0, 4)).toEqual(['symbol', 'positionValue', 'entry', 'margin']);
    expect(result).toHaveLength(8);
  });
  it('schiebt eine Spalte nach hinten auf die Position der Zielspalte', () => {
    const result = moveColumn(DEFAULT, 'positionValue', 'unrealizedProfit');
    expect(result.slice(0, 4)).toEqual(['symbol', 'margin', 'unrealizedProfit', 'positionValue']);
  });
  it('mutiert die Eingabe nicht', () => {
    const input = [...DEFAULT];
    moveColumn(input, 'entry', 'margin');
    expect(input).toEqual(DEFAULT);
  });
  it('from === to ändert nichts', () => {
    expect(moveColumn(DEFAULT, 'entry', 'entry')).toEqual(DEFAULT);
  });
  it('symbol ist als Quelle nicht verschiebbar', () => {
    expect(moveColumn(DEFAULT, 'symbol', 'entry')).toEqual(DEFAULT);
  });
  it('symbol ist als Ziel nicht erlaubt: Symbol bleibt erste Spalte', () => {
    expect(moveColumn(DEFAULT, 'entry', 'symbol')).toEqual(DEFAULT);
  });
  it('unbekannte Keys ändern nichts', () => {
    expect(moveColumn(DEFAULT, 'nope' as ColumnKey, 'entry')).toEqual(DEFAULT);
  });
});

describe('sanitizeColumnOrder', () => {
  it('gültige vollständige Reihenfolge bleibt erhalten', () => {
    const custom = moveColumn(DEFAULT, 'entry', 'margin');
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
    const result = sanitizeColumnOrder(['symbol', 'entry', 'entry', 'margin']);
    expect(result.filter((key) => key === 'entry')).toHaveLength(1);
    expect(result.slice(0, 3)).toEqual(['symbol', 'entry', 'margin']);
    expect(result).toHaveLength(8);
  });
  it('hängt fehlende Keys in Standardposition-Reihenfolge an', () => {
    const result = sanitizeColumnOrder(['symbol', 'tpSl', 'entry']);
    expect(result.slice(0, 3)).toEqual(['symbol', 'tpSl', 'entry']);
    expect(result.slice(3)).toEqual(
      DEFAULT.filter((key) => !['symbol', 'tpSl', 'entry'].includes(key)),
    );
  });
  it('ignoriert die entfernte Spalte rMultiple in alten gespeicherten Reihenfolgen', () => {
    const result = sanitizeColumnOrder(['symbol', 'rMultiple', 'entry']);
    expect(result).toHaveLength(8);
    expect(result.slice(0, 2)).toEqual(['symbol', 'entry']);
    expect(result).not.toContain('rMultiple');
  });
  it('erzwingt symbol als erste Spalte', () => {
    const result = sanitizeColumnOrder(['entry', 'symbol', 'margin']);
    expect(result[0]).toBe('symbol');
    expect(result).toHaveLength(8);
  });
  it('ignoriert entfernte alte Keys (side, mark, openTime, breakEven) und ergänzt fehlende', () => {
    const result = sanitizeColumnOrder(['symbol', 'side', 'mark', 'openTime', 'breakEven', 'tpSl']);
    expect(result).toEqual([
      'symbol',
      'tpSl',
      ...DEFAULT.filter((k) => k !== 'symbol' && k !== 'tpSl'),
    ]);
  });
});

describe('isDefaultColumnOrder', () => {
  it('true für Standard, false nach Verschieben', () => {
    expect(isDefaultColumnOrder(DEFAULT)).toBe(true);
    expect(isDefaultColumnOrder(moveColumn(DEFAULT, 'entry', 'margin'))).toBe(false);
  });
});
