import { describe, expect, it } from 'vitest';
import { ImportPositionsUseCase } from '@/application/position/ImportPositionsUseCase';
import type { Position } from '@/domain/position/entities/Position';
import type { PositionRecord } from '@/domain/position/interfaces/PositionRecord';
import type { PositionHistorySource } from '@/domain/position/interfaces/PositionHistorySource';
import type { TradedSymbolSource } from '@/domain/position/interfaces/TradedSymbolSource';
import type { PositionRepository } from '@/domain/position/interfaces/PositionRepository';
import type { ImportStateRepository } from '@/domain/position/interfaces/ImportStateRepository';
import type { Clock } from '@/domain/shared/interfaces/Clock';

const DAY_MS = 86_400_000;
const WINDOW_SIZE_MS = 89 * DAY_MS;
const OVERLAP_MS = 3_600_000;
const NOW_MS = 1_790_000_000_000;
const FULL_IMPORT_START_MS = NOW_MS - 200 * DAY_MS;

const INCOME_SCAN_KEY = 'income_scan_until';
const positionStateKey = (symbol: string): string => `position_history_until:${symbol}`;
const suspiciousKey = (symbol: string): string => `position_suspicious:${symbol}`;

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

  callsFor(symbol: string): FetchCall[] {
    return this.calls.filter((call) => call.symbol === symbol);
  }
}

class FakeTradedSymbolSource implements TradedSymbolSource {
  readonly calls: { fromMs: number; toMs: number }[] = [];

  constructor(
    private readonly symbols: readonly string[] = [],
    private readonly failure?: Error,
  ) {}

  async listSymbolsWithRealizedPnl(fromMs: number, toMs: number): Promise<ReadonlySet<string>> {
    this.calls.push({ fromMs, toMs });
    if (this.failure) throw this.failure;
    return new Set(this.symbols);
  }
}

class FakePositionRepository implements PositionRepository {
  readonly upsertCalls: (readonly PositionRecord[])[] = [];
  readonly storedById = new Map<string, PositionRecord>();

  async upsertMany(records: readonly PositionRecord[]): Promise<number> {
    this.upsertCalls.push(records);
    for (const record of records) this.storedById.set(record.position.positionId, record);
    return records.length;
  }
}

class FakeImportStateRepository implements ImportStateRepository {
  readonly valuesByKey = new Map<string, number>();
  readonly setCalls: { key: string; epochMs: number }[] = [];

  constructor(initialState: Record<string, number> = {}) {
    for (const [key, epochMs] of Object.entries(initialState)) this.valuesByKey.set(key, epochMs);
  }

  async get(key: string): Promise<number | undefined> {
    return this.valuesByKey.get(key);
  }

  async set(key: string, epochMs: number): Promise<void> {
    this.setCalls.push({ key, epochMs });
    this.valuesByKey.set(key, epochMs);
  }

  async delete(key: string): Promise<void> {
    this.valuesByKey.delete(key);
  }

  async listKeys(prefix: string): Promise<string[]> {
    return [...this.valuesByKey.keys()].filter((key) => key.startsWith(prefix));
  }
}

const fixedClock: Clock = { now: () => NOW_MS };

// ---------------------------------------------------------------- Hilfen

let positionIdCounter = 9_000_000_000_000_000_000n;

function positionRecord(symbol: string, closeTime: number): PositionRecord {
  positionIdCounter += 1n;
  const position: Position = {
    positionId: positionIdCounter.toString(),
    symbol,
    side: 'LONG',
    isolated: true,
    openTime: closeTime - 60_000,
    closeTime,
    avgOpenPrice: '1.0',
    avgClosePrice: '1.1',
    realisedProfit: '0.1',
    netProfit: '0.09',
    positionAmount: '1',
    closedAmount: '1',
    commission: '-0.01',
    totalFunding: '0',
    leverage: 5,
    fullyClosed: true,
  };
  return { position, rawPayload: JSON.stringify(position) };
}

type Setup = {
  scannedSymbols?: readonly string[];
  scanFailure?: Error;
  initialState?: Record<string, number>;
  fetchHandler?: FetchHandler;
};

function createUseCase(setup: Setup = {}) {
  const positionHistorySource = new FakePositionHistorySource(setup.fetchHandler);
  const tradedSymbolSource = new FakeTradedSymbolSource(setup.scannedSymbols, setup.scanFailure);
  const positionRepository = new FakePositionRepository();
  const importStateRepository = new FakeImportStateRepository(setup.initialState);

  const useCase = new ImportPositionsUseCase({
    positionHistorySource,
    tradedSymbolSource,
    positionRepository,
    importStateRepository,
    clock: fixedClock,
    windowSizeMs: WINDOW_SIZE_MS,
    overlapMs: OVERLAP_MS,
  });

  return {
    useCase,
    positionHistorySource,
    tradedSymbolSource,
    positionRepository,
    importStateRepository,
  };
}

// ---------------------------------------------------------------- Tests

describe('ImportPositionsUseCase', () => {
  describe('Zeitfenster', () => {
    it('fragt beim Vollimport ab fullImportStartMs in 89-Tage-Schritten bis now ab', async () => {
      const { useCase, positionHistorySource } = createUseCase({ scannedSymbols: ['BTC-USDT'] });

      await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(positionHistorySource.callsFor('BTC-USDT')).toEqual([
        {
          symbol: 'BTC-USDT',
          fromMs: FULL_IMPORT_START_MS,
          toMs: FULL_IMPORT_START_MS + WINDOW_SIZE_MS,
        },
        {
          symbol: 'BTC-USDT',
          fromMs: FULL_IMPORT_START_MS + WINDOW_SIZE_MS + 1,
          toMs: FULL_IMPORT_START_MS + 2 * WINDOW_SIZE_MS + 1,
        },
        { symbol: 'BTC-USDT', fromMs: FULL_IMPORT_START_MS + 2 * WINDOW_SIZE_MS + 2, toMs: NOW_MS },
      ]);
    });

    it('fragt kein Fenster über now hinaus ab und kein Fenster ist größer als windowSizeMs', async () => {
      const fullImportStartMs = NOW_MS - 365 * DAY_MS;
      const { useCase, positionHistorySource } = createUseCase({ scannedSymbols: ['BTC-USDT'] });

      await useCase.execute({ fullImportStartMs });

      const calls = positionHistorySource.callsFor('BTC-USDT');
      expect(calls.length).toBe(Math.ceil((365 * DAY_MS) / WINDOW_SIZE_MS));
      expect(calls[0]?.fromMs).toBe(fullImportStartMs);
      expect(calls.at(-1)?.toMs).toBe(NOW_MS);
      for (const call of calls) {
        expect(call.toMs).toBeLessThanOrEqual(NOW_MS);
        expect(call.fromMs).toBeLessThan(call.toMs);
        expect(call.toMs - call.fromMs).toBeLessThanOrEqual(WINDOW_SIZE_MS);
      }
      // Lückenlos und ohne doppelte Grenze: BingX filtert inklusiv, also beginnt jedes Fenster
      // 1 ms nach dem Ende des vorigen.
      calls.slice(1).forEach((call, index) => {
        expect(call.fromMs).toBe((calls[index]?.toMs ?? 0) + 1);
      });
    });

    it('setzt beim inkrementellen Lauf pro Symbol auf State minus Überlappung auf', async () => {
      const lastImportMs = NOW_MS - 10 * DAY_MS;
      const { useCase, positionHistorySource } = createUseCase({
        scannedSymbols: ['BTC-USDT'],
        initialState: {
          [INCOME_SCAN_KEY]: NOW_MS - 2 * 3_600_000,
          [positionStateKey('BTC-USDT')]: lastImportMs,
        },
      });

      await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(positionHistorySource.callsFor('BTC-USDT')).toEqual([
        { symbol: 'BTC-USDT', fromMs: lastImportMs - OVERLAP_MS, toMs: NOW_MS },
      ]);
    });

    it('beginnt nie vor fullImportStartMs, auch wenn der State älter ist', async () => {
      const { useCase, positionHistorySource } = createUseCase({
        scannedSymbols: ['BTC-USDT'],
        initialState: { [positionStateKey('BTC-USDT')]: FULL_IMPORT_START_MS - 50 * DAY_MS },
      });

      await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(positionHistorySource.callsFor('BTC-USDT')[0]?.fromMs).toBe(FULL_IMPORT_START_MS);
    });

    it('speichert die Positionen jedes Fensters über das Repository', async () => {
      const firstWindowRecord = positionRecord('BTC-USDT', FULL_IMPORT_START_MS + DAY_MS);
      const lastWindowRecord = positionRecord('BTC-USDT', NOW_MS - DAY_MS);
      const { useCase, positionRepository } = createUseCase({
        scannedSymbols: ['BTC-USDT'],
        fetchHandler: ({ fromMs, toMs }) =>
          [firstWindowRecord, lastWindowRecord].filter(
            (record) => record.position.closeTime >= fromMs && record.position.closeTime <= toMs,
          ),
      });

      const result = await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect([...positionRepository.storedById.keys()].sort()).toEqual(
        [firstWindowRecord.position.positionId, lastWindowRecord.position.positionId].sort(),
      );
      expect(result.savedPositionCount).toBe(2);
      expect(result.symbols).toContainEqual({ symbol: 'BTC-USDT', positionCount: 2 });
    });
  });

  describe('Income-Scan und Symbolauswahl', () => {
    it('scannt beim ersten Lauf ab fullImportStartMs bis now', async () => {
      const { useCase, tradedSymbolSource } = createUseCase();

      await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(tradedSymbolSource.calls).toEqual([{ fromMs: FULL_IMPORT_START_MS, toMs: NOW_MS }]);
    });

    it('scannt inkrementell ab income_scan_until minus Überlappung', async () => {
      const lastScanMs = NOW_MS - 5 * 3_600_000;
      const { useCase, tradedSymbolSource } = createUseCase({
        initialState: { [INCOME_SCAN_KEY]: lastScanMs },
      });

      await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(tradedSymbolSource.calls).toEqual([{ fromMs: lastScanMs - OVERLAP_MS, toMs: NOW_MS }]);
    });

    it('verarbeitet die Vereinigung aus gescannten Symbolen und Symbolen aus dem State', async () => {
      const { useCase, positionHistorySource } = createUseCase({
        scannedSymbols: ['BTC-USDT'],
        initialState: {
          [INCOME_SCAN_KEY]: NOW_MS - DAY_MS,
          [positionStateKey('ETH-USDT')]: NOW_MS - DAY_MS,
        },
      });

      await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      const fetchedSymbols = new Set(positionHistorySource.calls.map((call) => call.symbol));
      expect([...fetchedSymbols].sort()).toEqual(['BTC-USDT', 'ETH-USDT']);
    });

    it('fragt ein Symbol, das in Scan und State vorkommt, nur einmal pro Fenster ab', async () => {
      const { useCase, positionHistorySource } = createUseCase({
        scannedSymbols: ['BTC-USDT'],
        initialState: { [positionStateKey('BTC-USDT')]: NOW_MS - DAY_MS },
      });

      await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(positionHistorySource.callsFor('BTC-USDT')).toHaveLength(1);
    });

    it('setzt income_scan_until erst am Ende auf now', async () => {
      const { useCase, importStateRepository } = createUseCase({
        scannedSymbols: ['BTC-USDT', 'ETH-USDT'],
        fetchHandler: ({ symbol, toMs }) => [positionRecord(symbol, toMs)],
      });

      await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(importStateRepository.valuesByKey.get(INCOME_SCAN_KEY)).toBe(NOW_MS);
      expect(importStateRepository.setCalls.at(-1)).toEqual({
        key: INCOME_SCAN_KEY,
        epochMs: NOW_MS,
      });
    });

    it('bricht ab und setzt keinen State, wenn der Income-Scan fehlschlägt', async () => {
      const { useCase, importStateRepository, positionHistorySource } = createUseCase({
        scanFailure: new Error('Scan fehlgeschlagen'),
      });

      await expect(useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS })).rejects.toThrow(
        'Scan fehlgeschlagen',
      );
      expect(importStateRepository.setCalls).toEqual([]);
      expect(positionHistorySource.calls).toEqual([]);
    });
  });

  describe('State pro Symbol', () => {
    it('setzt den State eines Symbols mit Positionen auf now', async () => {
      const { useCase, importStateRepository } = createUseCase({
        scannedSymbols: ['BTC-USDT'],
        fetchHandler: ({ symbol, toMs }) => [positionRecord(symbol, toMs)],
      });

      await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(importStateRepository.valuesByKey.get(positionStateKey('BTC-USDT'))).toBe(NOW_MS);
    });

    it('setzt den State eines Symbols nur aus dem State (nicht im Scan) ohne Positionen auf now', async () => {
      const { useCase, importStateRepository } = createUseCase({
        scannedSymbols: [],
        initialState: { [positionStateKey('ETH-USDT')]: NOW_MS - DAY_MS },
      });

      const result = await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(result.suspiciousSymbols).toEqual([]);
      expect(importStateRepository.valuesByKey.get(positionStateKey('ETH-USDT'))).toBe(NOW_MS);
    });
  });

  describe('Verdacht auf still leere Antwort', () => {
    it('meldet KEINEN Verdacht, wenn die einzige Position im 2. Fenster liegt', async () => {
      const secondWindowStart = FULL_IMPORT_START_MS + WINDOW_SIZE_MS + 1;
      const { useCase, importStateRepository } = createUseCase({
        scannedSymbols: ['BTC-USDT'],
        fetchHandler: ({ symbol, fromMs }) =>
          fromMs === secondWindowStart ? [positionRecord(symbol, secondWindowStart + DAY_MS)] : [],
      });

      const result = await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(result.suspiciousSymbols).toEqual([]);
      expect(importStateRepository.valuesByKey.get(positionStateKey('BTC-USDT'))).toBe(NOW_MS);
    });

    it('meldet gescannte Symbole ohne jede Position als verdächtig (sortiert)', async () => {
      const { useCase } = createUseCase({
        scannedSymbols: ['XRP-USDT', 'BTC-USDT', 'ETH-USDT'],
        fetchHandler: ({ symbol, toMs }) =>
          symbol === 'BTC-USDT' ? [positionRecord(symbol, toMs)] : [],
      });

      const result = await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(result.suspiciousSymbols).toEqual(['ETH-USDT', 'XRP-USDT']);
      expect(result.symbols).toContainEqual({ symbol: 'ETH-USDT', positionCount: 0 });
    });

    it('lässt den State eines verdächtigen Symbols beim Vollimport auf fullImportStartMs', async () => {
      const { useCase, importStateRepository } = createUseCase({ scannedSymbols: ['ETH-USDT'] });

      await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(importStateRepository.valuesByKey.get(positionStateKey('ETH-USDT'))).toBe(
        FULL_IMPORT_START_MS,
      );
    });

    it('lässt einen vorhandenen State eines verdächtigen Symbols unverändert', async () => {
      const lastImportMs = NOW_MS - 5 * DAY_MS;
      const { useCase, importStateRepository } = createUseCase({
        scannedSymbols: ['ETH-USDT'],
        initialState: { [positionStateKey('ETH-USDT')]: lastImportMs },
      });

      await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(importStateRepository.valuesByKey.get(positionStateKey('ETH-USDT'))).toBe(
        lastImportMs,
      );
      expect(
        importStateRepository.setCalls.filter((call) => call.key === positionStateKey('ETH-USDT')),
      ).toEqual([]);
    });
  });

  describe('Verdacht bleibt über mehrere Läufe erhalten', () => {
    it('Regression: meldet den Verdacht auch im nächsten Lauf, wenn der Income-Scan das Symbol nicht mehr findet', async () => {
      const { useCase, positionHistorySource, positionRepository, importStateRepository } =
        createUseCase({ scannedSymbols: ['ETH-USDT'] });
      const firstResult = await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });
      expect(firstResult.suspiciousSymbols).toEqual(['ETH-USDT']);

      const secondRun = new ImportPositionsUseCase({
        positionHistorySource,
        tradedSymbolSource: { listSymbolsWithRealizedPnl: async () => new Set<string>() },
        positionRepository,
        importStateRepository,
        clock: fixedClock,
        windowSizeMs: WINDOW_SIZE_MS,
        overlapMs: OVERLAP_MS,
      });
      const secondResult = await secondRun.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(secondResult.suspiciousSymbols).toEqual(['ETH-USDT']);
      expect(importStateRepository.valuesByKey.get(positionStateKey('ETH-USDT'))).toBe(
        FULL_IMPORT_START_MS,
      );
    });

    it('hebt den Verdacht auf, sobald Positionen kommen, und rückt dann vor', async () => {
      const { useCase, importStateRepository } = createUseCase({
        scannedSymbols: [],
        initialState: {
          [positionStateKey('ETH-USDT')]: FULL_IMPORT_START_MS,
          [suspiciousKey('ETH-USDT')]: NOW_MS - DAY_MS,
        },
        fetchHandler: ({ symbol, toMs }) => [positionRecord(symbol, toMs)],
      });

      const result = await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(result.suspiciousSymbols).toEqual([]);
      expect(importStateRepository.valuesByKey.has(suspiciousKey('ETH-USDT'))).toBe(false);
      expect(importStateRepository.valuesByKey.get(positionStateKey('ETH-USDT'))).toBe(NOW_MS);
    });
  });

  describe('Fehler bei einzelnen Symbolen', () => {
    it('macht mit den anderen Symbolen weiter, ohne das fehlerhafte vorzurücken', async () => {
      const { useCase, importStateRepository, positionRepository } = createUseCase({
        scannedSymbols: ['BTC-USDT', 'ETH-USDT', 'XRP-USDT'],
        fetchHandler: ({ symbol, toMs }) => {
          if (symbol === 'ETH-USDT') throw new Error('Netzwerk kaputt');
          return [positionRecord(symbol, toMs)];
        },
      });

      const result = await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(result.failedSymbols).toEqual([{ symbol: 'ETH-USDT', reason: 'Netzwerk kaputt' }]);
      // Neues Symbol ohne State: State auf rangeStart, damit es bekannt bleibt.
      expect(importStateRepository.valuesByKey.get(positionStateKey('ETH-USDT'))).toBe(
        FULL_IMPORT_START_MS,
      );
      expect(importStateRepository.valuesByKey.get(positionStateKey('BTC-USDT'))).toBe(NOW_MS);
      expect(importStateRepository.valuesByKey.get(positionStateKey('XRP-USDT'))).toBe(NOW_MS);
      const storedSymbols = new Set(
        [...positionRepository.storedById.values()].map((record) => record.position.symbol),
      );
      expect(storedSymbols.has('BTC-USDT')).toBe(true);
      expect(storedSymbols.has('XRP-USDT')).toBe(true);
      expect(result.suspiciousSymbols).not.toContain('ETH-USDT');
      expect(importStateRepository.valuesByKey.get(INCOME_SCAN_KEY)).toBe(NOW_MS);
    });

    it('lässt einen vorhandenen State des fehlerhaften Symbols unverändert', async () => {
      const lastImportMs = NOW_MS - 3 * DAY_MS;
      const { useCase, importStateRepository } = createUseCase({
        scannedSymbols: ['ETH-USDT'],
        initialState: { [positionStateKey('ETH-USDT')]: lastImportMs },
        fetchHandler: () => {
          throw new Error('Rate-Limit');
        },
      });

      const result = await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(result.failedSymbols).toEqual([{ symbol: 'ETH-USDT', reason: 'Rate-Limit' }]);
      expect(importStateRepository.valuesByKey.get(positionStateKey('ETH-USDT'))).toBe(
        lastImportMs,
      );
      expect(
        importStateRepository.setCalls.filter((call) => call.key === positionStateKey('ETH-USDT')),
      ).toEqual([]);
    });
  });

  describe('Robustheit über mehrere Läufe', () => {
    it('Regression: ein neues Symbol, das beim ersten Lauf scheitert, wird im nächsten Lauf erneut versucht', async () => {
      let failEth = true;
      const { useCase, positionHistorySource, positionRepository, importStateRepository } =
        createUseCase({
          scannedSymbols: ['ETH-USDT'],
          fetchHandler: ({ symbol, toMs }) => {
            if (failEth) throw new Error('code 100410');
            return [positionRecord(symbol, toMs)];
          },
        });

      await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });
      expect(importStateRepository.valuesByKey.get(INCOME_SCAN_KEY)).toBe(NOW_MS);

      // Zweiter Lauf: der Income-Scan (ab now − Überlappung) findet ETH nicht mehr.
      failEth = false;
      const secondRun = new ImportPositionsUseCase({
        positionHistorySource,
        tradedSymbolSource: { listSymbolsWithRealizedPnl: async () => new Set<string>() },
        positionRepository,
        importStateRepository,
        clock: fixedClock,
        windowSizeMs: WINDOW_SIZE_MS,
        overlapMs: OVERLAP_MS,
      });
      positionHistorySource.calls.length = 0;

      const result = await secondRun.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(result.failedSymbols).toEqual([]);
      expect(positionHistorySource.callsFor('ETH-USDT')[0]?.fromMs).toBe(FULL_IMPORT_START_MS);
      expect(importStateRepository.valuesByKey.get(positionStateKey('ETH-USDT'))).toBe(NOW_MS);
    });

    it('schreibt den State eines Symbols, bevor das nächste Symbol abgefragt wird', async () => {
      const stateKeysSeenAtEthStart: string[] = [];
      const { useCase, importStateRepository } = createUseCase({
        scannedSymbols: ['BTC-USDT', 'ETH-USDT'],
        fetchHandler: ({ symbol, toMs }) => {
          if (symbol === 'ETH-USDT' && stateKeysSeenAtEthStart.length === 0) {
            stateKeysSeenAtEthStart.push(...importStateRepository.valuesByKey.keys(), '-');
          }
          return [positionRecord(symbol, toMs)];
        },
      });

      await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(stateKeysSeenAtEthStart).toContain(positionStateKey('BTC-USDT'));
      expect(stateKeysSeenAtEthStart).not.toContain(INCOME_SCAN_KEY);
    });
  });

  describe('Ergebnis', () => {
    it('liefert bei leerer Historie ein leeres Ergebnis ohne Fehler', async () => {
      const { useCase } = createUseCase();

      const result = await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(result).toEqual({
        savedPositionCount: 0,
        symbols: [],
        suspiciousSymbols: [],
        failedSymbols: [],
      });
    });

    it('summiert die gespeicherten Positionen über alle Symbole und Fenster', async () => {
      const { useCase } = createUseCase({
        scannedSymbols: ['BTC-USDT', 'ETH-USDT'],
        fetchHandler: ({ symbol, toMs }) => [positionRecord(symbol, toMs)],
      });

      const result = await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      // 3 Fenster pro Symbol, je 1 Position
      expect(result.savedPositionCount).toBe(6);
      expect(
        [...result.symbols].sort((left, right) => left.symbol.localeCompare(right.symbol)),
      ).toEqual([
        { symbol: 'BTC-USDT', positionCount: 3 },
        { symbol: 'ETH-USDT', positionCount: 3 },
      ]);
    });

    it('zählt eine Position, die in zwei Abfragen auftaucht, nur einmal', async () => {
      const samePosition = positionRecord('BTC-USDT', FULL_IMPORT_START_MS + WINDOW_SIZE_MS);
      const { useCase } = createUseCase({
        scannedSymbols: ['BTC-USDT'],
        fetchHandler: () => [samePosition],
      });

      const result = await useCase.execute({ fullImportStartMs: FULL_IMPORT_START_MS });

      expect(result.savedPositionCount).toBe(1);
      expect(result.symbols).toEqual([{ symbol: 'BTC-USDT', positionCount: 1 }]);
    });
  });
});
