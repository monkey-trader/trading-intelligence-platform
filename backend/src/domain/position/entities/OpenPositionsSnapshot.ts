import type { OpenPosition } from '@/domain/position/entities/OpenPosition';

/** Offene Positionen zu einem Zeitpunkt. `fetchedAt` ist der Abfragezeitpunkt in Epoch-Millisekunden. */
export type OpenPositionsSnapshot = Readonly<{
  positions: readonly OpenPosition[];
  fetchedAt: number;
}>;
