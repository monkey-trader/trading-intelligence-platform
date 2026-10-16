import { describe, expect, it } from 'vitest';
import { DiscoverHistoricalSymbolsUseCase } from '@/application/position/DiscoverHistoricalSymbolsUseCase';
import type { Position } from '@/domain/position/entities/Position';
import type { ContractSymbolSource } from '@/domain/position/interfaces/ContractSymbolSource';
import type { ImportStateRepository } from '@/domain/position/interfaces/ImportStateRepository';
import type { PositionHistorySource } from '@/domain/position/interfaces/PositionHistorySource';
import type { PositionRecord } from '@/domain/position/interfaces/PositionRecord';
import type { PositionRepository } from '@/domain/position/interfaces/PositionRepository';

const DAY_MS = 86_400_000;
const WINDOW_SIZE_MS = 89 * DAY_MS;
const NOW_MS = 1_790_000_000_000;
const FULL_IMPORT_START_MS = NOW_MS - 365 * DAY_MS;
const DISCOVERY_UNTIL_MS = NOW_MS - 60 * DAY_MS;

const COMPLETED_KEY = 'contract_discovery_completed';
const checkedKey = (symbol: string): string => `contract_discovery_checked:${symbol}`;
const positionStateKey = (symbol: string): string => `position_history_until:${symbol}`;

// ---------------------------------------------------------------- Fakes (bewusst im Test definiert)

type FetchCall = { symbol: string; fromMs: number; toMs: number };
type FetchHandler = (call: FetchCall) => PositionRecord[];

class FakePositionHistorySource implements PositionHistorySource {
  readonly calls: FetchCall[] = [];

  constructor(private readonly handler: FetchHandler = () => []) {}

  async fetchClosedPositions(
    symbol: string,
    fromMs: number,
    toMs: number,
  ): Promise<PositionRecord[]> {
    const call = { symbol, fromMs, toMs };
    this.calls.push(call);
    return this.handler(call);
  }

  fetchedSymbols(): string[] {
    return [...new Set(this.calls.map((call) => call.symbol))].sort();
  }
}

class FakeContractSymbolSource implements ContractSymbolSource {
  callCount = 0;

  constructor(private readonly symbols: readonly string[]) {}

  async listContractSymbols(): Promise<string[]> {
    this.callCount += 1;
    return [...this.symbols];
  }
}

class FakePositionRepository implements PositionRepository {
  readonly storedById = new Map<string, PositionRecord>();

  async upsertMany(records: readonly PositionRecord[]): Promise<number> {
    for (const record of records) this.storedById.set(record.position.positionId, record);
    return records.length;
  }
}

class FakeImportStateRepository implements ImportStateRepository {
  readonly valuesByKey = new Map<string, number>();

  constructor(initialState: Record<string, number> = {}) {
    for (const [key, epochMs] of Object.entries(initialState)) this.valuesByKey.set(key, epochMs);
  }

  async get(key: string): Promise<number | undefined> {
    return this.valuesByKey.get(key);
  }

  async set(key: string, epochMs: number): Promise<void> {
    this.valuesByKey.set(key, epochMs);
  }

  async delete(key: string): Promise<void> {
    this.valuesByKey.delete(key);
  }

  async listKeys(prefix: string): Promise<string[]> {
    return [...this.valuesByKey.keys()].filter((key) => key.startsWith(prefix));
  }
}

let positionIdCounter = 8_000_000_000_000_000_000n;

function positionRecord(symbol: string, closeTime: number): PositionRecord {
  positionIdCounter += 1n;
  const position: Position = {
    positionId: positionIdCounter.toString(),
    symbol,
    side: 'SHORT',
    isolated: true,
    openTime: closeTime - 60_000,
    closeTime,
    avgOpenPrice: '1.0',
    avgClosePrice: '0.9',
    realisedProfit: '0.1',
    netProfit: '0.09',
    positionAmount: '1',
    closedAmount: '1',
    commission: '-0.01',
    totalFunding: '0',
    leverage: 3,
    fullyClosed: true,
  };
  return { position, rawPayload: JSON.stringify(position) };
}

type Setup = {
  contractSymbols?: readonly string[];
  initialState?: Record<string, number>;
  fetchHandler?: FetchHandler;
};

function createUseCase(setup: Setup = {}) {
  const contractSymbolSource = new FakeContractSymbolSource(setup.contractSymbols ?? []);
  const positionHistorySource = new FakePositionHistorySource(setup.fetchHandler);
  const positionRepository = new FakePositionRepository();
  const importStateRepository = new FakeImportStateRepository(setup.initialState);

  const useCase = new DiscoverHistoricalSymbolsUseCase({
    contractSymbolSource,
    positionHistorySource,
    positionRepository,
    importStateRepository,
    windowSizeMs: WINDOW_SIZE_MS,
  });

  return {
    useCase,
    contractSymbolSource,
    positionHistorySource,
    positionRepository,
    importStateRepository,
  };
}

const command = { fullImportStartMs: FULL_IMPORT_START_MS, discoveryUntilMs: DISCOVERY_UNTIL_MS };

// ---------------------------------------------------------------- Tests

describe('DiscoverHistoricalSymbolsUseCase', () => {
  describe('Auswahl der Kontrakte', () => {
    it('prüft nur Kontrakte, die weder bekannt noch schon geprüft sind', async () => {
      const { useCase, positionHistorySource } = createUseCase({
        contractSymbols: ['BTC-USDT', 'ETH-USDT', 'SEI-USDT', 'XRP-USDT'],
        initialState: {
          [positionStateKey('BTC-USDT')]: NOW_MS,
          [checkedKey('XRP-USDT')]: FULL_IMPORT_START_MS,
        },
      });

      const result = await useCase.execute(command);

      expect(positionHistorySource.fetchedSymbols()).toEqual(['ETH-USDT', 'SEI-USDT']);
      expect(result.checkedSymbolCount).toBe(2);
    });

    it('tut nichts und fragt nicht einmal die Kontrakte ab, wenn die Suche abgeschlossen ist', async () => {
      const { useCase, contractSymbolSource, positionHistorySource } = createUseCase({
        contractSymbols: ['ETH-USDT'],
        initialState: { [COMPLETED_KEY]: FULL_IMPORT_START_MS },
      });

      const result = await useCase.execute(command);

      expect(result.alreadyCompleted).toBe(true);
      expect(contractSymbolSource.callCount).toBe(0);
      expect(positionHistorySource.calls).toEqual([]);
    });
  });

  describe('Zeitfenster', () => {
    it('fragt von fullImportStartMs bis discoveryUntilMs in Fenstern ohne doppelte Grenze ab', async () => {
      const { useCase, positionHistorySource } = createUseCase({ contractSymbols: ['ETH-USDT'] });

      await useCase.execute(command);

      const calls = positionHistorySource.calls;
      expect(calls[0]?.fromMs).toBe(FULL_IMPORT_START_MS);
      expect(calls.at(-1)?.toMs).toBe(DISCOVERY_UNTIL_MS);
      expect(calls).toHaveLength(
        Math.ceil((DISCOVERY_UNTIL_MS - FULL_IMPORT_START_MS) / WINDOW_SIZE_MS),
      );
      for (const call of calls) {
        expect(call.toMs - call.fromMs).toBeLessThanOrEqual(WINDOW_SIZE_MS);
      }
      calls.slice(1).forEach((call, index) => {
        expect(call.fromMs).toBe((calls[index]?.toMs ?? 0) + 1);
      });
    });

    it('fragt nichts ab, wenn discoveryUntilMs nicht nach fullImportStartMs liegt, markiert aber als geprüft', async () => {
      const { useCase, positionHistorySource, importStateRepository } = createUseCase({
        contractSymbols: ['ETH-USDT'],
      });

      const result = await useCase.execute({
        fullImportStartMs: DISCOVERY_UNTIL_MS,
        discoveryUntilMs: DISCOVERY_UNTIL_MS,
      });

      expect(positionHistorySource.calls).toEqual([]);
      expect(importStateRepository.valuesByKey.has(checkedKey('ETH-USDT'))).toBe(true);
      expect(result.alreadyCompleted).toBe(false);
    });
  });

  describe('Gefundene Symbole', () => {
    it('speichert gefundene Positionen und macht das Symbol dem Import bekannt', async () => {
      const oldPosition = positionRecord('NCFXEUR2GBP-USDT', FULL_IMPORT_START_MS + DAY_MS);
      const { useCase, positionRepository, importStateRepository } = createUseCase({
        contractSymbols: ['ETH-USDT', 'NCFXEUR2GBP-USDT'],
        fetchHandler: ({ symbol, fromMs }) =>
          symbol === 'NCFXEUR2GBP-USDT' && fromMs === FULL_IMPORT_START_MS ? [oldPosition] : [],
      });

      const result = await useCase.execute(command);

      expect([...positionRepository.storedById.keys()]).toEqual([oldPosition.position.positionId]);
      expect(result.discoveredSymbols).toEqual([{ symbol: 'NCFXEUR2GBP-USDT', positionCount: 1 }]);
      // Der normale Import macht ab discoveryUntilMs (minus Überlappung) weiter.
      expect(importStateRepository.valuesByKey.get(positionStateKey('NCFXEUR2GBP-USDT'))).toBe(
        DISCOVERY_UNTIL_MS,
      );
    });

    it('legt für Kontrakte ohne Positionen keinen Import-State an (sonst würde jeder Lauf sie abfragen)', async () => {
      const { useCase, importStateRepository } = createUseCase({ contractSymbols: ['ETH-USDT'] });

      const result = await useCase.execute(command);

      expect(importStateRepository.valuesByKey.has(positionStateKey('ETH-USDT'))).toBe(false);
      // Der Vermerk speichert den durchsuchten Vollimport-Start.
      expect(importStateRepository.valuesByKey.get(checkedKey('ETH-USDT'))).toBe(
        FULL_IMPORT_START_MS,
      );
      expect(result.discoveredSymbols).toEqual([]);
    });

    it('zählt eine Position, die mehrfach geliefert wird, nur einmal', async () => {
      const samePosition = positionRecord('SEI-USDT', FULL_IMPORT_START_MS + WINDOW_SIZE_MS);
      const { useCase } = createUseCase({
        contractSymbols: ['SEI-USDT'],
        fetchHandler: () => [samePosition],
      });

      const result = await useCase.execute(command);

      expect(result.discoveredSymbols).toEqual([{ symbol: 'SEI-USDT', positionCount: 1 }]);
    });
  });

  describe('Abschluss und Fortsetzbarkeit', () => {
    it('setzt contract_discovery_completed, wenn alle Kontrakte ohne Fehler geprüft sind', async () => {
      const { useCase, importStateRepository } = createUseCase({
        contractSymbols: ['ETH-USDT', 'SEI-USDT'],
      });

      await useCase.execute(command);

      expect(importStateRepository.valuesByKey.get(COMPLETED_KEY)).toBe(FULL_IMPORT_START_MS);
    });

    it('markiert ein fehlerhaftes Symbol nicht als geprüft, macht weiter und schließt nicht ab', async () => {
      const { useCase, importStateRepository } = createUseCase({
        contractSymbols: ['ETH-USDT', 'SEI-USDT', 'XRP-USDT'],
        fetchHandler: ({ symbol }) => {
          if (symbol === 'SEI-USDT') throw new Error('code 100410');
          return [];
        },
      });

      const result = await useCase.execute(command);

      expect(result.failedSymbols).toEqual([{ symbol: 'SEI-USDT', reason: 'code 100410' }]);
      expect(importStateRepository.valuesByKey.has(checkedKey('SEI-USDT'))).toBe(false);
      expect(importStateRepository.valuesByKey.has(checkedKey('ETH-USDT'))).toBe(true);
      expect(importStateRepository.valuesByKey.has(checkedKey('XRP-USDT'))).toBe(true);
      expect(importStateRepository.valuesByKey.has(COMPLETED_KEY)).toBe(false);
    });

    it('markiert jedes Symbol sofort als geprüft, bevor das nächste abgefragt wird', async () => {
      const keysSeenAtSecondSymbol: string[] = [];
      const { useCase, importStateRepository } = createUseCase({
        contractSymbols: ['ETH-USDT', 'SEI-USDT'],
        fetchHandler: ({ symbol }) => {
          if (symbol === 'SEI-USDT' && keysSeenAtSecondSymbol.length === 0) {
            keysSeenAtSecondSymbol.push(...importStateRepository.valuesByKey.keys(), '-');
          }
          return [];
        },
      });

      await useCase.execute(command);

      expect(keysSeenAtSecondSymbol).toContain(checkedKey('ETH-USDT'));
    });

    it('setzt nach einem Abbruch fort und fragt bereits geprüfte Kontrakte nicht erneut ab', async () => {
      let abortAtSei = true;
      const { useCase, positionHistorySource, importStateRepository } = createUseCase({
        contractSymbols: ['ETH-USDT', 'SEI-USDT', 'XRP-USDT'],
        fetchHandler: ({ symbol }) => {
          if (symbol === 'SEI-USDT' && abortAtSei) throw new Error('Abbruch (Ctrl+C)');
          return [];
        },
      });
      await useCase.execute(command);
      expect(importStateRepository.valuesByKey.has(COMPLETED_KEY)).toBe(false);

      abortAtSei = false;
      positionHistorySource.calls.length = 0;
      const result = await useCase.execute(command);

      expect(positionHistorySource.fetchedSymbols()).toEqual(['SEI-USDT']);
      expect(result.failedSymbols).toEqual([]);
      expect(importStateRepository.valuesByKey.has(COMPLETED_KEY)).toBe(true);
    });

    it('reicht einen Fehler der Kontraktliste durch, ohne etwas zu markieren', async () => {
      const importStateRepository = new FakeImportStateRepository();
      const failingUseCase = new DiscoverHistoricalSymbolsUseCase({
        contractSymbolSource: {
          listContractSymbols: async () => {
            throw new Error('quote/contracts kaputt');
          },
        },
        positionHistorySource: new FakePositionHistorySource(),
        positionRepository: new FakePositionRepository(),
        importStateRepository,
        windowSizeMs: WINDOW_SIZE_MS,
      });
      await expect(failingUseCase.execute(command)).rejects.toThrow('quote/contracts kaputt');
      expect(importStateRepository.valuesByKey.size).toBe(0);
    });

    it('sucht erneut, wenn später ein früherer Vollimport-Start verlangt wird (IMPORT_FULL_DAYS erhöht)', async () => {
      const { useCase, positionHistorySource } = createUseCase({ contractSymbols: ['ETH-USDT'] });
      await useCase.execute(command);
      positionHistorySource.calls.length = 0;

      const earlierStartMs = FULL_IMPORT_START_MS - 200 * DAY_MS;
      const result = await useCase.execute({ ...command, fullImportStartMs: earlierStartMs });

      expect(result.alreadyCompleted).toBe(false);
      expect(positionHistorySource.calls[0]).toMatchObject({
        symbol: 'ETH-USDT',
        fromMs: earlierStartMs,
      });
    });

    it('sucht nicht erneut, wenn der Vollimport-Start nur durch Zeitablauf später liegt', async () => {
      const { useCase, contractSymbolSource } = createUseCase({ contractSymbols: ['ETH-USDT'] });
      await useCase.execute(command);

      const result = await useCase.execute({
        fullImportStartMs: FULL_IMPORT_START_MS + DAY_MS,
        discoveryUntilMs: DISCOVERY_UNTIL_MS + DAY_MS,
      });

      expect(result.alreadyCompleted).toBe(true);
      expect(contractSymbolSource.callCount).toBe(1);
    });

    it('meldet den Fortschritt pro geprüftem Symbol', async () => {
      const progress: { checked: number; total: number; symbol: string }[] = [];
      const { useCase } = createUseCase({ contractSymbols: ['SEI-USDT', 'ETH-USDT'] });

      await useCase.execute({
        ...command,
        onProgress: (checked, total, symbol) => progress.push({ checked, total, symbol }),
      });

      expect(progress).toEqual([
        { checked: 1, total: 2, symbol: 'ETH-USDT' },
        { checked: 2, total: 2, symbol: 'SEI-USDT' },
      ]);
    });
  });
});
