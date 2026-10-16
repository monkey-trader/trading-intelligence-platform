import type { PositionRecord } from '@/domain/position/interfaces/PositionRecord';

/** Port: liefert die geschlossenen Positionen eines Symbols im Zeitraum [fromMs, toMs]. */
export type PositionHistorySource = {
  fetchClosedPositions(symbol: string, fromMs: number, toMs: number): Promise<PositionRecord[]>;
};
