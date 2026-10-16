import { describe, expect, it } from 'vitest';
import { formatImportResult } from '@/presentation/position/cli/formatImportResult';
import type { ImportPositionsResult } from '@/application/position/ImportPositionsUseCase';

const emptyResult: ImportPositionsResult = {
  savedPositionCount: 0,
  symbols: [],
  suspiciousSymbols: [],
  failedSymbols: [],
};

describe('formatImportResult', () => {
  it('zeigt die Anzahl gespeicherter Positionen', () => {
    const output = formatImportResult({
      ...emptyResult,
      savedPositionCount: 42,
      symbols: [
        { symbol: 'ONDO-USDT', positionCount: 40 },
        { symbol: 'BTC-USDT', positionCount: 2 },
      ],
    });

    expect(output).toContain('42');
  });

  it('zeigt die Symbole mit ihrer Positionsanzahl', () => {
    const output = formatImportResult({
      ...emptyResult,
      savedPositionCount: 3,
      symbols: [{ symbol: 'ONDO-USDT', positionCount: 3 }],
    });

    expect(output).toContain('ONDO-USDT');
  });

  it('hebt Verdachtsfälle mit Symbolnamen hervor', () => {
    const output = formatImportResult({
      ...emptyResult,
      suspiciousSymbols: ['ETH-USDT', 'XRP-USDT'],
    });

    expect(output).toContain('ETH-USDT');
    expect(output).toContain('XRP-USDT');
    expect(output).toMatch(/verdacht/i);
  });

  it('zeigt Fehler mit Symbol und Grund', () => {
    const output = formatImportResult({
      ...emptyResult,
      failedSymbols: [{ symbol: 'BTC-USDT', reason: 'Rate-Limit erschöpft' }],
    });

    expect(output).toContain('BTC-USDT');
    expect(output).toContain('Rate-Limit erschöpft');
    expect(output).toMatch(/fehler/i);
  });

  it('erwähnt weder Verdacht noch Fehler, wenn alles in Ordnung ist', () => {
    const output = formatImportResult({
      ...emptyResult,
      savedPositionCount: 1,
      symbols: [{ symbol: 'ONDO-USDT', positionCount: 1 }],
    });

    expect(output).toContain('1');
    expect(output).not.toMatch(/verdacht/i);
    expect(output).not.toMatch(/fehler/i);
  });

  it('liefert auch bei leerer Historie einen nicht leeren Text', () => {
    const output = formatImportResult(emptyResult);

    expect(output.trim().length).toBeGreaterThan(0);
    expect(output).toContain('0');
  });
});
