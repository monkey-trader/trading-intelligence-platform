import { importPositionsInWindows } from '@/application/position/importPositionsInWindows';
import {
  errorMessage,
  POSITION_STATE_KEY_PREFIX,
  positionStateKey,
  suspiciousStateKey,
} from '@/application/position/importStateKeys';
import type { ImportStateRepository } from '@/domain/position/interfaces/ImportStateRepository';
import type { PositionHistorySource } from '@/domain/position/interfaces/PositionHistorySource';
import type { PositionRepository } from '@/domain/position/interfaces/PositionRepository';
import type { TradedSymbolSource } from '@/domain/position/interfaces/TradedSymbolSource';
import { findSuspiciouslyEmptySymbols } from '@/domain/position/services/findSuspiciouslyEmptySymbols';
import type { Clock } from '@/domain/shared/interfaces/Clock';

const INCOME_SCAN_STATE_KEY = 'income_scan_until';

export type ImportPositionsDependencies = Readonly<{
  positionHistorySource: PositionHistorySource;
  tradedSymbolSource: TradedSymbolSource;
  positionRepository: PositionRepository;
  importStateRepository: ImportStateRepository;
  clock: Clock;
  /** Größe eines Abfragefensters, muss unter dem 90-Tage-Limit von BingX liegen. */
  windowSizeMs: number;
  /** Überlappung beim inkrementellen Lauf, damit an der Grenze nichts verloren geht. */
  overlapMs: number;
}>;

export type ImportPositionsCommand = Readonly<{ fullImportStartMs: number }>;

export type ImportPositionsResult = {
  savedPositionCount: number;
  symbols: { symbol: string; positionCount: number }[];
  suspiciousSymbols: string[];
  failedSymbols: { symbol: string; reason: string }[];
};

type SymbolImportOutcome = {
  symbol: string;
  /** Anzahl verschiedener Positionen (positionId), auch wenn eine mehrfach geliefert wurde. */
  positionCount: number;
  suspicious: boolean;
  /** Gesetzt, wenn der Import des Symbols fehlgeschlagen ist. */
  failureReason: string | undefined;
};

/**
 * Importiert geschlossene Positionen inkrementell: Income-Scan für die gehandelten Symbole,
 * dann pro Symbol in Zeitfenstern abfragen und speichern. Paging übernimmt der Adapter.
 *
 * Der State wird direkt nach jedem Symbol geschrieben, `income_scan_until` erst am Ende.
 * Ein Verdacht wird gespeichert und bleibt bestehen, bis Positionen kommen, auch wenn der
 * inkrementelle Income-Scan das Symbol später nicht mehr findet.
 * Verdächtige oder fehlgeschlagene Symbole rücken nicht vor. Hatten sie noch keinen State,
 * bekommen sie einen auf dem Bereichsbeginn, damit sie im nächsten Lauf bekannt bleiben, auch wenn
 * der inkrementelle Income-Scan sie dann nicht mehr findet.
 */
export class ImportPositionsUseCase {
  constructor(private readonly dependencies: ImportPositionsDependencies) {}

  async execute(command: ImportPositionsCommand): Promise<ImportPositionsResult> {
    const { tradedSymbolSource, importStateRepository, clock } = this.dependencies;
    const nowMs = clock.now();

    const scanStartMs = await this.rangeStart(INCOME_SCAN_STATE_KEY, command.fullImportStartMs);
    const scannedSymbols = await tradedSymbolSource.listSymbolsWithRealizedPnl(scanStartMs, nowMs);
    const symbolsToImport = await this.collectSymbols(scannedSymbols);

    const outcomes: SymbolImportOutcome[] = [];
    for (const symbol of symbolsToImport) {
      outcomes.push(
        await this.importSymbol(symbol, scannedSymbols, command.fullImportStartMs, nowMs),
      );
    }
    await importStateRepository.set(INCOME_SCAN_STATE_KEY, nowMs);

    const succeededOutcomes = outcomes.filter((outcome) => outcome.failureReason === undefined);
    return {
      savedPositionCount: succeededOutcomes.reduce(
        (sum, outcome) => sum + outcome.positionCount,
        0,
      ),
      symbols: succeededOutcomes.map(({ symbol, positionCount }) => ({ symbol, positionCount })),
      suspiciousSymbols: succeededOutcomes
        .filter((outcome) => outcome.suspicious)
        .map((outcome) => outcome.symbol)
        .sort(),
      failedSymbols: outcomes.flatMap((outcome) =>
        outcome.failureReason === undefined
          ? []
          : [{ symbol: outcome.symbol, reason: outcome.failureReason }],
      ),
    };
  }

  /** Beginn des Bereichs: gespeicherter Stand minus Überlappung, aber nie vor dem Vollimport-Start. */
  private async rangeStart(stateKey: string, fullImportStartMs: number): Promise<number> {
    const storedUntilMs = await this.dependencies.importStateRepository.get(stateKey);
    if (storedUntilMs === undefined) return fullImportStartMs;
    return Math.max(fullImportStartMs, storedUntilMs - this.dependencies.overlapMs);
  }

  /** Gescannte Symbole plus alle Symbole, die schon einmal importiert wurden, sortiert. */
  private async collectSymbols(scannedSymbols: ReadonlySet<string>): Promise<string[]> {
    const stateKeys =
      await this.dependencies.importStateRepository.listKeys(POSITION_STATE_KEY_PREFIX);
    const knownSymbols = stateKeys.map((stateKey) =>
      stateKey.slice(POSITION_STATE_KEY_PREFIX.length),
    );
    return [...new Set([...scannedSymbols, ...knownSymbols])].sort();
  }

  private async importSymbol(
    symbol: string,
    scannedSymbols: ReadonlySet<string>,
    fullImportStartMs: number,
    nowMs: number,
  ): Promise<SymbolImportOutcome> {
    const { positionHistorySource, positionRepository, importStateRepository, windowSizeMs } =
      this.dependencies;
    const stateKey = positionStateKey(symbol);
    const hadState = (await importStateRepository.get(stateKey)) !== undefined;
    const rangeStartMs = await this.rangeStart(stateKey, fullImportStartMs);

    // Nicht vorrücken: vorhandenen State behalten, sonst auf rangeStart setzen.
    const keepWithoutAdvancing = async (): Promise<void> => {
      if (!hadState) await importStateRepository.set(stateKey, rangeStartMs);
    };

    let positionCount: number;
    try {
      positionCount = await importPositionsInWindows(positionHistorySource, positionRepository, {
        symbol,
        fromMs: rangeStartMs,
        toMs: nowMs,
        windowSizeMs,
      });
    } catch (error: unknown) {
      await keepWithoutAdvancing();
      return { symbol, positionCount: 0, suspicious: false, failureReason: errorMessage(error) };
    }

    const suspicionKey = suspiciousStateKey(symbol);
    const wasSuspicious = (await importStateRepository.get(suspicionKey)) !== undefined;
    const hasRealizedPnl = scannedSymbols.has(symbol) || wasSuspicious;
    const suspicious =
      findSuspiciouslyEmptySymbols(
        hasRealizedPnl ? new Set([symbol]) : new Set<string>(),
        new Map([[symbol, positionCount]]),
      ).length > 0;

    if (suspicious) {
      await keepWithoutAdvancing();
      if (!wasSuspicious) await importStateRepository.set(suspicionKey, nowMs);
    } else {
      await importStateRepository.set(stateKey, nowMs);
      if (wasSuspicious) await importStateRepository.delete(suspicionKey);
    }

    return { symbol, positionCount, suspicious, failureReason: undefined };
  }
}
