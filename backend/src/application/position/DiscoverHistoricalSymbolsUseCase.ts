import { importPositionsInWindows } from '@/application/position/importPositionsInWindows';
import {
  errorMessage,
  POSITION_STATE_KEY_PREFIX,
  positionStateKey,
} from '@/application/position/importStateKeys';
import type { ContractSymbolSource } from '@/domain/position/interfaces/ContractSymbolSource';
import type { ImportStateRepository } from '@/domain/position/interfaces/ImportStateRepository';
import type { PositionHistorySource } from '@/domain/position/interfaces/PositionHistorySource';
import type { PositionRepository } from '@/domain/position/interfaces/PositionRepository';

const COMPLETED_STATE_KEY = 'contract_discovery_completed';
const CHECKED_STATE_KEY_PREFIX = 'contract_discovery_checked:';

export type DiscoverHistoricalSymbolsDependencies = Readonly<{
  contractSymbolSource: ContractSymbolSource;
  positionHistorySource: PositionHistorySource;
  positionRepository: PositionRepository;
  importStateRepository: ImportStateRepository;
  /** Größe eines Abfragefensters, muss unter dem 90-Tage-Limit von BingX liegen. */
  windowSizeMs: number;
}>;

export type DiscoverHistoricalSymbolsCommand = Readonly<{
  fullImportStartMs: number;
  /** Ende der Suche. Jüngere Symbole findet bereits der Income-Scan (~90 Tage). */
  discoveryUntilMs: number;
  onProgress?: (checkedCount: number, totalCount: number, symbol: string) => void;
}>;

export type DiscoverHistoricalSymbolsResult = {
  /** Die Suche lief schon früher vollständig durch, es wurde nichts abgefragt. */
  alreadyCompleted: boolean;
  checkedSymbolCount: number;
  discoveredSymbols: { symbol: string; positionCount: number }[];
  failedSymbols: { symbol: string; reason: string }[];
};

/**
 * Einmalige Suche nach Symbolen, die nur vor mehr als ~90 Tagen gehandelt wurden. Diese findet der
 * Income-Scan nicht, weil BingX Income nur ~90 Tage aufbewahrt. Deshalb wird positionHistory für
 * jeden Kontrakt abgefragt, der weder bekannt noch schon geprüft ist.
 *
 * Fortsetzbar: Jeder Kontrakt wird sofort als geprüft markiert. Gefundene Symbole bekommen einen
 * Import-Stand, ab dann pflegt sie der normale Import. Sind alle Kontrakte ohne Fehler geprüft,
 * wird die Suche als abgeschlossen markiert und danach übersprungen.
 *
 * Abschluss- und Prüfvermerk speichern den durchsuchten Vollimport-Start. Liegt ein späterer
 * Start davor (IMPORT_FULL_DAYS erhöht), gilt der Vermerk nicht mehr und es wird neu gesucht.
 */
export class DiscoverHistoricalSymbolsUseCase {
  constructor(private readonly dependencies: DiscoverHistoricalSymbolsDependencies) {}

  async execute(
    command: DiscoverHistoricalSymbolsCommand,
  ): Promise<DiscoverHistoricalSymbolsResult> {
    const { importStateRepository } = this.dependencies;
    const result: DiscoverHistoricalSymbolsResult = {
      alreadyCompleted: false,
      checkedSymbolCount: 0,
      discoveredSymbols: [],
      failedSymbols: [],
    };

    const completedFromMs = await importStateRepository.get(COMPLETED_STATE_KEY);
    if (completedFromMs !== undefined && completedFromMs <= command.fullImportStartMs) {
      return { ...result, alreadyCompleted: true };
    }

    const pendingSymbols = await this.pendingSymbols(command.fullImportStartMs);
    for (const [index, symbol] of pendingSymbols.entries()) {
      try {
        const positionCount = await importPositionsInWindows(
          this.dependencies.positionHistorySource,
          this.dependencies.positionRepository,
          {
            symbol,
            fromMs: command.fullImportStartMs,
            toMs: command.discoveryUntilMs,
            windowSizeMs: this.dependencies.windowSizeMs,
          },
        );
        if (positionCount > 0) {
          // Der normale Import macht ab hier (minus Überlappung) weiter.
          await importStateRepository.set(positionStateKey(symbol), command.discoveryUntilMs);
          result.discoveredSymbols.push({ symbol, positionCount });
        }
        await importStateRepository.set(checkedStateKey(symbol), command.fullImportStartMs);
        result.checkedSymbolCount += 1;
      } catch (error: unknown) {
        result.failedSymbols.push({ symbol, reason: errorMessage(error) });
      }
      command.onProgress?.(index + 1, pendingSymbols.length, symbol);
    }

    if (result.failedSymbols.length === 0) {
      await importStateRepository.set(COMPLETED_STATE_KEY, command.fullImportStartMs);
    }
    return result;
  }

  /** Alle Kontrakte ohne Import-Stand und ohne gültigen Prüfvermerk, sortiert. */
  private async pendingSymbols(fullImportStartMs: number): Promise<string[]> {
    const { contractSymbolSource, importStateRepository } = this.dependencies;
    const checkedSymbols: string[] = [];
    for (const stateKey of await importStateRepository.listKeys(CHECKED_STATE_KEY_PREFIX)) {
      const checkedFromMs = await importStateRepository.get(stateKey);
      if (checkedFromMs !== undefined && checkedFromMs <= fullImportStartMs) {
        checkedSymbols.push(stateKey.slice(CHECKED_STATE_KEY_PREFIX.length));
      }
    }
    const knownSymbols = new Set([
      ...symbolsFromKeys(
        await importStateRepository.listKeys(POSITION_STATE_KEY_PREFIX),
        POSITION_STATE_KEY_PREFIX,
      ),
      ...checkedSymbols,
    ]);
    const contractSymbols = await contractSymbolSource.listContractSymbols();
    return [...new Set(contractSymbols)].filter((symbol) => !knownSymbols.has(symbol)).sort();
  }
}

function checkedStateKey(symbol: string): string {
  return `${CHECKED_STATE_KEY_PREFIX}${symbol}`;
}

function symbolsFromKeys(stateKeys: readonly string[], prefix: string): string[] {
  return stateKeys.map((stateKey) => stateKey.slice(prefix.length));
}
