import { describe, expect, it } from 'vitest';
import {
  DEFAULT_COLUMN_ORDER,
  isDefaultColumnOrder,
  moveColumn,
  sanitizeColumnOrder,
  type ColumnKey,
} from '@/presentation/position/columnOrder';

const DEFAULT = [...DEFAULT_COLUMN_ORDER];

describe('DEFAULT_COLUMN_ORDER', () => {
  it('beginnt mit symbol und enthält 13 eindeutige Spalten', () => {
    expect(DEFAULT_COLUMN_ORDER[0]).toBe('symbol');
    expect(new Set(DEFAULT_COLUMN_ORDER).size).toBe(13);
  });
});

describe('moveColumn', () => {
  it('schiebt eine Spalte nach vorn auf die Position der Zielspalte', () => {
    const result = moveColumn(DEFAULT, 'mark', 'side');
    expect(result.slice(0, 4)).toEqual(['symbol', 'mark', 'side', 'amount']);
    expect(result).toHaveLength(13);
  });
  it('schiebt eine Spalte nach hinten auf die Position der Zielspalte', () => {
    const result = moveColumn(DEFAULT, 'side', 'entry');
    expect(result.slice(0, 5)).toEqual(['symbol', 'amount', 'entry', 'side', 'mark']);
  });
  it('mutiert die Eingabe nicht', () => {
    const input = [...DEFAULT];
    moveColumn(input, 'mark', 'side');
    expect(input).toEqual(DEFAULT);
  });
  it('from === to ändert nichts', () => {
    expect(moveColumn(DEFAULT, 'mark', 'mark')).toEqual(DEFAULT);
  });
  it('symbol ist als Quelle nicht verschiebbar', () => {
    expect(moveColumn(DEFAULT, 'symbol', 'mark')).toEqual(DEFAULT);
  });
  it('symbol ist als Ziel nicht erlaubt: Symbol bleibt erste Spalte', () => {
    expect(moveColumn(DEFAULT, 'mark', 'symbol')).toEqual(DEFAULT);
  });
  it('unbekannte Keys ändern nichts', () => {
    expect(moveColumn(DEFAULT, 'nope' as ColumnKey, 'mark')).toEqual(DEFAULT);
  });
});

describe('sanitizeColumnOrder', () => {
  it('gültige vollständige Reihenfolge bleibt erhalten', () => {
    const custom = moveColumn(DEFAULT, 'mark', 'side');
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
    const result = sanitizeColumnOrder(['symbol', 'mark', 'mark', 'side']);
    expect(result.filter((key) => key === 'mark')).toHaveLength(1);
    expect(result.slice(0, 3)).toEqual(['symbol', 'mark', 'side']);
    expect(result).toHaveLength(13);
  });
  it('hängt fehlende Keys in Standardposition-Reihenfolge an', () => {
    const result = sanitizeColumnOrder(['symbol', 'openTime', 'mark']);
    expect(result.slice(0, 3)).toEqual(['symbol', 'openTime', 'mark']);
    expect(result.slice(3)).toEqual(
      DEFAULT.filter((key) => !['symbol', 'openTime', 'mark'].includes(key)),
    );
  });
  it('erzwingt symbol als erste Spalte', () => {
    const result = sanitizeColumnOrder(['mark', 'symbol', 'side']);
    expect(result[0]).toBe('symbol');
    expect(result).toHaveLength(13);
  });
});

describe('isDefaultColumnOrder', () => {
  it('true für Standard, false nach Verschieben', () => {
    expect(isDefaultColumnOrder(DEFAULT)).toBe(true);
    expect(isDefaultColumnOrder(moveColumn(DEFAULT, 'mark', 'side'))).toBe(false);
  });
});
