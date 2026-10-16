import type { PositionHistorySource } from '@/domain/position/interfaces/PositionHistorySource';
import type { PositionRecord } from '@/domain/position/interfaces/PositionRecord';
import type { BingXGetClient } from '@/infrastructure/bingx/BingXGetClient';
import { mapPositionHistoryResponse } from '@/infrastructure/position/bingx/positionHistoryMapper';

const POSITION_HISTORY_PATH = '/openApi/swap/v1/trade/positionHistory';
const PAGE_SIZE = 100;
const DEFAULT_MAX_PAGES = 50;

export type BingXPositionHistorySourceOptions = Readonly<{ maxPages?: number }>;

/**
 * Lädt geschlossene Positionen eines Symbols von BingX und blättert dabei durch alle Seiten.
 * pageIndex beginnt bei 0. Das Zeitfenster (max. 90 Tage) wählt der Use Case.
 */
export class BingXPositionHistorySource implements PositionHistorySource {
  private readonly maxPages: number;

  constructor(
    private readonly client: BingXGetClient,
    options: BingXPositionHistorySourceOptions = {},
  ) {
    this.maxPages = options.maxPages ?? DEFAULT_MAX_PAGES;
  }

  async fetchClosedPositions(
    symbol: string,
    fromMs: number,
    toMs: number,
  ): Promise<PositionRecord[]> {
    const records: PositionRecord[] = [];
    for (let pageIndex = 0; pageIndex < this.maxPages; pageIndex += 1) {
      const data = await this.client.get(POSITION_HISTORY_PATH, {
        symbol,
        startTs: fromMs,
        endTs: toMs,
        pageIndex,
        pageSize: PAGE_SIZE,
      });
      const pageRecords = mapPositionHistoryResponse(data);
      records.push(...pageRecords);
      if (pageRecords.length < PAGE_SIZE) return records;
    }
    throw new Error(
      `positionHistory für ${symbol}: mehr als ${this.maxPages} volle Seiten, Abbruch zum Schutz vor Endlosschleife.`,
    );
  }
}
