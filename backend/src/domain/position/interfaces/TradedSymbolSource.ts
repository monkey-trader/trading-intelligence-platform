/** Port: liefert die Symbole, für die im Zeitraum realisierter PnL gebucht wurde. */
export type TradedSymbolSource = {
  listSymbolsWithRealizedPnl(fromMs: number, toMs: number): Promise<ReadonlySet<string>>;
};
