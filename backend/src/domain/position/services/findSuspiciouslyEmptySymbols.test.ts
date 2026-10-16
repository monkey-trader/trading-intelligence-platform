import { describe, expect, it } from 'vitest';
import { findSuspiciouslyEmptySymbols } from '@/domain/position/services/findSuspiciouslyEmptySymbols';

describe('findSuspiciouslyEmptySymbols', () => {
  it('meldet Symbole mit realisiertem PnL, aber 0 Positionen, sortiert', () => {
    const symbolsWithPnl = new Set(['XRP-USDT', 'BTC-USDT', 'ONDO-USDT']);
    const positionCountBySymbol = new Map([
      ['XRP-USDT', 0],
      ['BTC-USDT', 0],
      ['ONDO-USDT', 3],
    ]);

    expect(findSuspiciouslyEmptySymbols(symbolsWithPnl, positionCountBySymbol)).toEqual([
      'BTC-USDT',
      'XRP-USDT',
    ]);
  });

  it('behandelt ein Symbol ohne Eintrag in der Zählung wie 0 Positionen', () => {
    const symbolsWithPnl = new Set(['ETH-USDT']);

    expect(findSuspiciouslyEmptySymbols(symbolsWithPnl, new Map())).toEqual(['ETH-USDT']);
  });

  it('meldet Symbole ohne realisierten PnL nicht, auch wenn sie 0 Positionen haben', () => {
    const symbolsWithPnl = new Set<string>();
    const positionCountBySymbol = new Map([['ETH-USDT', 0]]);

    expect(findSuspiciouslyEmptySymbols(symbolsWithPnl, positionCountBySymbol)).toEqual([]);
  });

  it('liefert eine leere Liste, wenn alle Symbole Positionen haben', () => {
    const symbolsWithPnl = new Set(['ONDO-USDT']);
    const positionCountBySymbol = new Map([['ONDO-USDT', 1]]);

    expect(findSuspiciouslyEmptySymbols(symbolsWithPnl, positionCountBySymbol)).toEqual([]);
  });
});
