/**
 * Symbole mit realisiertem PnL, für die trotzdem keine Position gefunden wurde.
 * Das deutet auf eine still leere Antwort der Börse hin. Ergebnis ist sortiert.
 */
export function findSuspiciouslyEmptySymbols(
  symbolsWithPnl: ReadonlySet<string>,
  positionCountBySymbol: ReadonlyMap<string, number>,
): string[] {
  return [...symbolsWithPnl]
    .filter((symbol) => (positionCountBySymbol.get(symbol) ?? 0) === 0)
    .sort();
}
