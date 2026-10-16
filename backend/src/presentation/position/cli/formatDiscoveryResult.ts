import type { DiscoverHistoricalSymbolsResult } from '@/application/position/DiscoverHistoricalSymbolsUseCase';

/** Deutsche Zusammenfassung der Suche nach alten Symbolen für die Konsole. */
export function formatDiscoveryResult(result: DiscoverHistoricalSymbolsResult): string {
  if (result.alreadyCompleted) {
    return 'Suche nach alten Symbolen: bereits abgeschlossen.';
  }

  const lines = [
    `Suche nach alten Symbolen: ${result.checkedSymbolCount} Kontrakte geprüft, ${result.discoveredSymbols.length} mit Positionen gefunden.`,
  ];
  for (const { symbol, positionCount } of result.discoveredSymbols) {
    lines.push(`  ${symbol}: ${positionCount}`);
  }

  if (result.failedSymbols.length > 0) {
    lines.push(
      '',
      `FEHLER bei ${result.failedSymbols.length} Kontrakt(en), der nächste Lauf versucht sie erneut:`,
    );
    for (const { symbol, reason } of result.failedSymbols) {
      lines.push(`  ${symbol}: ${reason}`);
    }
  }

  return lines.join('\n');
}
