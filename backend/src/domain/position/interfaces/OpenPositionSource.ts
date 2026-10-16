import type { OpenPositionsSnapshot } from '@/domain/position/entities/OpenPositionsSnapshot';

/** Port: liefert die aktuell offenen Positionen samt Abfragezeitpunkt. */
export type OpenPositionSource = {
  fetchOpenPositions(): Promise<OpenPositionsSnapshot>;
};
