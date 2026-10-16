import type { OpenPositionsResult } from '@/domain/position/OpenPositionsResult';

/**
 * Port zum Abfragen der offenen Positionen.
 * Liefert Fehler als Ergebnis; nur ein Abbruch über das Signal wird als AbortError geworfen.
 */
export type OpenPositionsGateway = Readonly<{
  fetchOpenPositions: (signal: AbortSignal) => Promise<OpenPositionsResult>;
}>;
