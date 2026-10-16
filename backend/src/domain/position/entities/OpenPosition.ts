/** Aktuell offene Position mit primitiven Feldern. Beträge bleiben Dezimal-Strings. */
export type OpenPosition = Readonly<{
  positionId: string;
  symbol: string;
  side: 'LONG' | 'SHORT';
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
  openTime: number;
  updateTime: number;
}>;
