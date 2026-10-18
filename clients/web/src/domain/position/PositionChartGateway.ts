import type { ChartInterval } from '@/domain/position/PositionChart';
import type { PositionChartResult } from '@/domain/position/PositionChartResult';

/**
 * Port zum Abfragen des Charts einer offenen Position.
 * Liefert Fehler als Ergebnis; nur ein Abbruch über das Signal wird als AbortError geworfen.
 */
export type PositionChartGateway = Readonly<{
  fetchChart: (
    positionId: string,
    interval: ChartInterval,
    limit: number,
    signal: AbortSignal,
  ) => Promise<PositionChartResult>;
}>;
