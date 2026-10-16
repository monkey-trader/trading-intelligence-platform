/** Geschlossene Position mit primitiven Feldern. Beträge bleiben Dezimal-Strings. */
export type Position = Readonly<{
  positionId: string;
  symbol: string;
  side: 'LONG' | 'SHORT';
  isolated: boolean;
  openTime: number;
  closeTime: number;
  avgOpenPrice: string;
  avgClosePrice: string;
  realisedProfit: string;
  netProfit: string;
  positionAmount: string;
  closedAmount: string;
  commission: string;
  totalFunding: string;
  leverage: number;
  fullyClosed: boolean;
}>;
