export type PeriodKey = 'today' | '7d' | '30d';

/** Netto-PnL eines Zeitraums und Prozent der Equity am Periodenanfang. */
export type PeriodPnl = Readonly<{
  period: PeriodKey;
  realizedPnl: string;
  /** undefined, wenn die Basis 0 oder negativ ist. */
  pnlPercent: string | undefined;
}>;

/** Kontostand und Monats-Performance (Beträge als Dezimal-Strings, verlustfrei). */
export type AccountSummary = Readonly<{
  balance: string;
  equity: string;
  availableMargin: string;
  /** Kalendermonat YYYY-MM, auf den sich der Monats-PnL bezieht. */
  month: string;
  monthlyRealizedPnl: string;
  /** Monats-PnL in Prozent des Kontostands; undefined bei Kontostand 0. */
  monthlyPnlPercent: string | undefined;
  /** Heute, 7T, 30T in dieser Reihenfolge. */
  periods: readonly PeriodPnl[];
  unrealizedPnl: string;
  /** Unrealisierter PnL in Prozent der Equity; undefined, wenn nicht berechenbar. */
  unrealizedPnlPercent: string | undefined;
}>;
