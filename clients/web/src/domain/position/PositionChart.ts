/** Intervalle der Kerzen, identisch zum API-Vertrag (aufsteigend nach Länge). */
export const CHART_INTERVALS = ['5m', '15m', '1h', '4h', '1d'] as const;

export type ChartInterval = (typeof CHART_INTERVALS)[number];

export const DEFAULT_CHART_INTERVAL: ChartInterval = '15m';

/** Anzahl der Kerzen, die standardmäßig geladen werden. */
export const DEFAULT_CHART_LIMIT = 300;

/** Eine Kerze; Zeit in Epoch-Millisekunden, Preise als Dezimal-Strings. */
export type Candle = Readonly<{
  time: number;
  open: string;
  high: string;
  low: string;
  close: string;
  volume: string;
}>;

/** Offene Stop-Loss- oder Take-Profit-Order. */
export type ProtectiveOrder = Readonly<{
  orderId: string;
  triggerPrice: string;
  quantity: string;
  /** Abstand zum Entry relativ zum Entry, vorzeichenbehaftet. */
  distanceRatio: string;
}>;

export type ChartMarker = Readonly<{
  time: number;
  kind: 'ENTRY';
  price: string;
}>;

/** CRV-Level aus dem ursprünglichen Risiko; TWO_R ist der Break-even-Trigger. */
export type RLevel = Readonly<{
  kind: 'ONE_R' | 'TWO_R' | 'THREE_R';
  rMultiple: string;
  price: string;
  /** Abstand zum Entry relativ zum Entry. */
  percent: string;
}>;

export type PositionChart = Readonly<{
  positionId: string;
  symbol: string;
  side: 'LONG' | 'SHORT';
  interval: ChartInterval;
  candles: readonly Candle[];
  entry: Readonly<{ price: string; time: number }>;
  stopLoss: readonly ProtectiveOrder[];
  takeProfit: readonly ProtectiveOrder[];
  markers: readonly ChartMarker[];
  /** Fehlt oder leer, wenn das Risiko unbekannt ist. */
  levels?: readonly RLevel[] | undefined;
  fetchedAt: number;
}>;
