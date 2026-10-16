import type { DatabaseSync } from 'node:sqlite';
import type { PositionRecord } from '@/domain/position/interfaces/PositionRecord';
import type { PositionRepository } from '@/domain/position/interfaces/PositionRepository';

// Upsert nur über ON CONFLICT DO UPDATE: die Zeile bleibt bestehen, `notes` wird nie angefasst.
// INSERT OR REPLACE würde die Zeile löschen und neu anlegen.
const UPSERT_SQL = `
INSERT INTO positions (
  position_id, symbol, side, isolated, open_time, close_time,
  avg_open_price, avg_close_price, realised_profit, net_profit,
  position_amount, closed_amount, commission, total_funding,
  leverage, fully_closed, raw, imported_at
) VALUES (
  :positionId, :symbol, :side, :isolated, :openTime, :closeTime,
  :avgOpenPrice, :avgClosePrice, :realisedProfit, :netProfit,
  :positionAmount, :closedAmount, :commission, :totalFunding,
  :leverage, :fullyClosed, :raw, :importedAt
)
ON CONFLICT(position_id) DO UPDATE SET
  symbol = excluded.symbol,
  side = excluded.side,
  isolated = excluded.isolated,
  open_time = excluded.open_time,
  close_time = excluded.close_time,
  avg_open_price = excluded.avg_open_price,
  avg_close_price = excluded.avg_close_price,
  realised_profit = excluded.realised_profit,
  net_profit = excluded.net_profit,
  position_amount = excluded.position_amount,
  closed_amount = excluded.closed_amount,
  commission = excluded.commission,
  total_funding = excluded.total_funding,
  leverage = excluded.leverage,
  fully_closed = excluded.fully_closed,
  raw = excluded.raw,
  imported_at = excluded.imported_at
`;

/** Speichert Positionen in SQLite, idempotent über `position_id`. */
export class SqlitePositionRepository implements PositionRepository {
  constructor(
    private readonly database: DatabaseSync,
    private readonly now: () => number = Date.now,
  ) {}

  async upsertMany(records: readonly PositionRecord[]): Promise<number> {
    if (records.length === 0) return 0;
    const statement = this.database.prepare(UPSERT_SQL);
    const importedAt = this.now();

    this.database.exec('BEGIN');
    try {
      for (const { position, rawPayload } of records) {
        statement.run({
          positionId: position.positionId,
          symbol: position.symbol,
          side: position.side,
          isolated: position.isolated ? 1 : 0,
          openTime: position.openTime,
          closeTime: position.closeTime,
          avgOpenPrice: position.avgOpenPrice,
          avgClosePrice: position.avgClosePrice,
          realisedProfit: position.realisedProfit,
          netProfit: position.netProfit,
          positionAmount: position.positionAmount,
          closedAmount: position.closedAmount,
          commission: position.commission,
          totalFunding: position.totalFunding,
          leverage: position.leverage,
          fullyClosed: position.fullyClosed ? 1 : 0,
          raw: rawPayload,
          importedAt,
        });
      }
      this.database.exec('COMMIT');
    } catch (error: unknown) {
      this.database.exec('ROLLBACK');
      throw error;
    }
    return records.length;
  }
}
