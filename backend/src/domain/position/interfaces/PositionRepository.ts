import type { PositionRecord } from '@/domain/position/interfaces/PositionRecord';

/** Port: speichert Positionen idempotent (Upsert über die Positions-ID). Notizen bleiben unberührt. */
export type PositionRepository = {
  /** Liefert die Anzahl der verarbeiteten Datensätze. */
  upsertMany(records: readonly PositionRecord[]): Promise<number>;
};
