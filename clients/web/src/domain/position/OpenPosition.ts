/** Seite einer Position im Hedge-Modus. */
export type PositionSide = 'LONG' | 'SHORT';

/**
 * Offene Position, wie sie das Backend liefert.
 * Dezimalwerte bleiben Strings, damit nichts durch Gleitkomma verfälscht wird.
 */
export type OpenPosition = Readonly<{
  positionId: string;
  symbol: string;
  side: PositionSide;
  isolated: boolean;
  positionAmount: string;
  avgOpenPrice: string;
  markPrice: string;
  leverage: number;
  initialMargin: string;
  margin: string;
  positionValue: string;
  unrealizedProfit: string;
  realisedProfit: string;
  pnlRatio: string;
  /** "0" bedeutet keine Liquidation. */
  liquidationPrice: string;
  /** null ohne Liquidationspreis oder bei Mark-Preis 0. */
  liquidationDistanceRatio: string | null;
  /** Stop-Loss-Preis; null = keiner gesetzt. */
  stopLoss: string | null;
  /** Take-Profit-Preis; null = keiner gesetzt. */
  takeProfit: string | null;
  /** true = SL auf/über Einstand; null = Orders unbekannt. */
  breakEven: boolean | null;
  /** Epoch-Millisekunden. */
  openTime: number;
  /** Epoch-Millisekunden. */
  updateTime: number;
}>;

/** Alle offenen Positionen zu einem Zeitpunkt, samt exakter PnL-Summe. */
export type OpenPositionsSnapshot = Readonly<{
  positions: readonly OpenPosition[];
  totalUnrealizedProfit: string;
  /** Zeitpunkt der Abfrage bei BingX in Epoch-Millisekunden. */
  fetchedAt: number;
}>;
