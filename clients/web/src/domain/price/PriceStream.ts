/** Kurs einer offenen Position (Dezimal-Strings, der Client rechnet nichts). */
export type PositionPrice = Readonly<{
  positionId: string;
  symbol: string;
  price: string;
  change24hPercent: string;
  distanceToBreakEvenPercent: string | null;
  distanceToTp1Percent: string | null;
}>;

export type PriceSnapshot = Readonly<{
  prices: readonly PositionPrice[];
  fetchedAt: number;
  stale: boolean;
}>;

/** 'live' = Stream steht, 'ausgefallen' = Fehler oder geschlossen. */
export type PriceStreamStatus = 'live' | 'ausgefallen';

export type PriceStreamHandlers = Readonly<{
  onSnapshot: (snapshot: PriceSnapshot) => void;
  onStatus: (status: PriceStreamStatus) => void;
}>;

/** Port: abonniert den Kurs-Stream; der Rückgabewert beendet ihn (close). */
export type PriceStreamGateway = Readonly<{
  subscribe: (handlers: PriceStreamHandlers) => () => void;
}>;
