import type { ImportPositionsResult } from '@/application/position/ImportPositionsUseCase';

/** Deutsche Zusammenfassung eines Import-Laufs für die Konsole. Verdachtsfälle und Fehler stehen am Ende. */
export function formatImportResult(result: ImportPositionsResult): string {
  const lines = [
    `Import abgeschlossen: ${result.savedPositionCount} Positionen abgerufen und gespeichert (neu oder aktualisiert).`,
  ];

  if (result.symbols.length === 0) {
    lines.push('Keine Symbole mit Handelsaktivität gefunden.');
  } else {
    lines.push(`Symbole (${result.symbols.length}):`);
    const sortedSymbols = [...result.symbols].sort((left, right) =>
      left.symbol.localeCompare(right.symbol),
    );
    for (const { symbol, positionCount } of sortedSymbols) {
      lines.push(`  ${symbol}: ${positionCount}`);
    }
  }

  if (result.suspiciousSymbols.length > 0) {
    lines.push(
      '',
      `ACHTUNG Verdacht auf still leere Antwort (realisierter PnL, aber 0 Positionen): ${result.suspiciousSymbols.join(', ')}`,
      'Der Stand dieser Symbole wurde nicht vorgerückt, der nächste Lauf prüft sie erneut.',
    );
  }

  if (result.failedSymbols.length > 0) {
    lines.push('', `FEHLER bei ${result.failedSymbols.length} Symbol(en):`);
    for (const { symbol, reason } of result.failedSymbols) {
      lines.push(`  ${symbol}: ${reason}`);
    }
  }

  return lines.join('\n');
}
