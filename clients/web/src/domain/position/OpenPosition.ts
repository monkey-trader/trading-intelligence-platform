/** CRV-Level (2R/3R) als Preis; 2R = Break-even-Trigger, 3R = TP1. */
export type OpenPositionLevel = Readonly<{
  kind: 'TWO_R' | 'THREE_R';
  rMultiple: number;
  price: string;
  percent: string;
}>;

/** Herkunft des ursprünglichen SL. */
export type InitialStopLossSource = 'AUTO' | 'MANUAL';

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
  /** Stop-Loss-Preis; null = keiner gesetzt. */
  stopLoss: string | null;
  /** Take-Profit-Preis; null = keiner gesetzt. */
  takeProfit: string | null;
  /** true = SL auf/über Einstand; null = Orders unbekannt. */
  breakEven: boolean | null;
  /** Ursprünglicher SL; null = unbekannt. */
  initialStopLoss: string | null;
  /** Herkunft des Risikos: AUTO (aus Orders), MANUAL (von Hand); null = unbekannt. */
  initialStopLossSource: InitialStopLossSource | null;
  /** Risiko-Distanz (1R in Preiseinheiten); null = unbekannt. */
  riskDistance: string | null;
  /** Aktueller Gewinn in R; null = Risiko unbekannt. */
  currentRMultiple: string | null;
  /** true, sobald der Kurs seit Eröffnung 2R erreicht hat; null = unbekannt. */
  breakEvenTriggerReached: boolean | null;
  /** Kerzenbeginn (Epoch-Millisekunden) des ersten 2R; null = nicht erreicht/unbekannt. */
  reached2RAt: number | null;
  /** CRV-Level; fehlt, wenn das Risiko unbekannt ist. */
  levels?: readonly OpenPositionLevel[] | undefined;
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
