import type { PositionHistorySource } from '@/domain/position/interfaces/PositionHistorySource';
import type { PositionRepository } from '@/domain/position/interfaces/PositionRepository';

export type WindowedImportRange = Readonly<{
  symbol: string;
  fromMs: number;
  toMs: number;
  windowSizeMs: number;
}>;

/**
 * Holt die geschlossenen Positionen eines Symbols in Zeitfenstern und speichert sie.
 * BingX filtert inklusiv [startTs, endTs], deshalb beginnt jedes Fenster 1 ms nach dem vorigen.
 * Liefert die Anzahl verschiedener Positionen (positionId), auch wenn eine mehrfach kam.
 */
export async function importPositionsInWindows(
  positionHistorySource: PositionHistorySource,
  positionRepository: PositionRepository,
  range: WindowedImportRange,
): Promise<number> {
  const seenPositionIds = new Set<string>();
  for (let windowStartMs = range.fromMs; windowStartMs < range.toMs;) {
    const windowEndMs = Math.min(windowStartMs + range.windowSizeMs, range.toMs);
    const records = await positionHistorySource.fetchClosedPositions(
      range.symbol,
      windowStartMs,
      windowEndMs,
    );
    if (records.length > 0) {
      await positionRepository.upsertMany(records);
      for (const record of records) seenPositionIds.add(record.position.positionId);
    }
    windowStartMs = windowEndMs + 1;
  }
  return seenPositionIds.size;
}
