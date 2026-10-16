import { describe, expect, it } from 'vitest';
import type { DiscoverHistoricalSymbolsResult } from '@/application/position/DiscoverHistoricalSymbolsUseCase';
import { formatDiscoveryResult } from '@/presentation/position/cli/formatDiscoveryResult';

const emptyResult: DiscoverHistoricalSymbolsResult = {
  alreadyCompleted: false,
  checkedSymbolCount: 0,
  discoveredSymbols: [],
  failedSymbols: [],
};

describe('formatDiscoveryResult', () => {
  it('meldet eine bereits abgeschlossene Suche mit einer kurzen Zeile', () => {
    const output = formatDiscoveryResult({ ...emptyResult, alreadyCompleted: true });

    expect(output.split('\n')).toHaveLength(1);
    expect(output).toMatch(/abgeschlossen/i);
  });

  it('zeigt die Anzahl geprüfter Kontrakte und die gefundenen Symbole', () => {
    const output = formatDiscoveryResult({
      ...emptyResult,
      checkedSymbolCount: 1184,
      discoveredSymbols: [{ symbol: 'NCFXEUR2GBP-USDT', positionCount: 1 }],
    });

    expect(output).toContain('1184');
    expect(output).toContain('NCFXEUR2GBP-USDT: 1');
  });

  it('zeigt Fehler nur, wenn es welche gibt', () => {
    expect(formatDiscoveryResult(emptyResult)).not.toMatch(/fehler/i);

    const output = formatDiscoveryResult({
      ...emptyResult,
      failedSymbols: [{ symbol: 'SEI-USDT', reason: 'code 100410' }],
    });

    expect(output).toMatch(/fehler/i);
    expect(output).toContain('SEI-USDT: code 100410');
  });
});
