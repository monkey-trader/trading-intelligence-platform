import type { TradedSymbolSource } from '@/domain/position/interfaces/TradedSymbolSource';
import type { BingXGetClient } from '@/infrastructure/bingx/BingXGetClient';
import { UnexpectedBingXResponseError } from '@/infrastructure/bingx/errors';

const INCOME_PATH = '/openApi/swap/v2/user/income';
const INCOME_LIMIT = 1000;
// BingX liefert bei größeren Zeiträumen still leere Listen, daher 7-Tage-Fenster.
// Achtung: Income reicht nur ~90 Tage zurück. Ältere Symbole findet dieser Scan nicht.
const WINDOW_SIZE_MS = 7 * 86_400_000;
const REALIZED_PNL = 'REALIZED_PNL';

type IncomeRow = { symbol: string; incomeType: string; time: number };

/** Ermittelt über die Income-Historie, welche Symbole realisierten PnL hatten. */
export class BingXTradedSymbolSource implements TradedSymbolSource {
  constructor(private readonly client: BingXGetClient) {}

  async listSymbolsWithRealizedPnl(fromMs: number, toMs: number): Promise<ReadonlySet<string>> {
    const symbols = new Set<string>();
    for (let windowStartMs = fromMs; windowStartMs < toMs; windowStartMs += WINDOW_SIZE_MS) {
      const windowEndMs = Math.min(windowStartMs + WINDOW_SIZE_MS, toMs);
      await this.collectWindow(windowStartMs, windowEndMs, symbols);
    }
    return symbols;
  }

  /** Blättert im Fenster weiter (startTime = max(time) + 1), solange volle Seiten kommen. */
  private async collectWindow(
    windowStartMs: number,
    windowEndMs: number,
    symbols: Set<string>,
  ): Promise<void> {
    let pageStartMs = windowStartMs;
    while (pageStartMs <= windowEndMs) {
      const rows = parseIncomeRows(
        await this.client.get(INCOME_PATH, {
          startTime: pageStartMs,
          endTime: windowEndMs,
          limit: INCOME_LIMIT,
        }),
      );
      for (const row of rows) {
        if (row.incomeType === REALIZED_PNL) symbols.add(row.symbol);
      }
      if (rows.length < INCOME_LIMIT) return;
      pageStartMs = Math.max(...rows.map((row) => row.time)) + 1;
    }
  }
}

function parseIncomeRows(data: unknown): IncomeRow[] {
  // Gemessen: Für Fenster außerhalb der ~90 Tage Income-Historie liefert BingX mal [] und mal null.
  if (data === null) return [];
  if (!Array.isArray(data)) {
    throw new UnexpectedBingXResponseError('Antwort von user/income ist keine Liste.');
  }
  return data.map((rawRow: unknown): IncomeRow => {
    if (
      typeof rawRow !== 'object' ||
      rawRow === null ||
      !('symbol' in rawRow) ||
      !('incomeType' in rawRow) ||
      !('time' in rawRow) ||
      typeof rawRow.symbol !== 'string' ||
      typeof rawRow.incomeType !== 'string' ||
      typeof rawRow.time !== 'number'
    ) {
      throw new UnexpectedBingXResponseError(
        'user/income enthält einen Eintrag ohne symbol, incomeType oder time.',
      );
    }
    return { symbol: rawRow.symbol, incomeType: rawRow.incomeType, time: rawRow.time };
  });
}
