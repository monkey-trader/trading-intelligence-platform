// Zweiter Composition Root: baut die Adapter für den Positions-Import und startet einen Lauf.
// Nach dem Import läuft einmalig (fortsetzbar) die Suche nach Symbolen älter als ~90 Tage.
// Exit-Codes: 0 = ok, 2 = Verdachtsfälle (still leere Antworten), 1 = Fehler.
// Konfiguration, Key und Secret werden nie ausgegeben.
import { mkdirSync } from 'node:fs';
import { dirname } from 'node:path';
import { DiscoverHistoricalSymbolsUseCase } from '@/application/position/DiscoverHistoricalSymbolsUseCase';
import { ImportPositionsUseCase } from '@/application/position/ImportPositionsUseCase';
import { BingXClient } from '@/infrastructure/bingx/BingXClient';
import { readImportConfig } from '@/infrastructure/config/importConfig';
import { openDatabase } from '@/infrastructure/persistence/sqlite/openDatabase';
import { SqliteImportStateRepository } from '@/infrastructure/persistence/sqlite/SqliteImportStateRepository';
import { BingXContractSymbolSource } from '@/infrastructure/position/BingXContractSymbolSource';
import { BingXPositionHistorySource } from '@/infrastructure/position/BingXPositionHistorySource';
import { BingXTradedSymbolSource } from '@/infrastructure/position/BingXTradedSymbolSource';
import { SqlitePositionRepository } from '@/infrastructure/position/SqlitePositionRepository';
import { systemClock } from '@/infrastructure/shared/SystemClock';
import { formatDiscoveryResult } from '@/presentation/position/cli/formatDiscoveryResult';
import { formatImportResult } from '@/presentation/position/cli/formatImportResult';

const DAY_MS = 86_400_000;
// BingX erlaubt maximal 90 Tage pro Abfrage, 89 lassen Luft.
const WINDOW_SIZE_MS = 89 * DAY_MS;
const OVERLAP_MS = 3_600_000;
// Die Suche nach alten Symbolen endet hier. Jüngere findet der Income-Scan (~90 Tage), 60 lassen Luft.
const DISCOVERY_MIN_AGE_MS = 60 * DAY_MS;
const PROGRESS_EVERY_SYMBOLS = 50;

const EXIT_OK = 0;
const EXIT_FAILURE = 1;
const EXIT_SUSPICIOUS = 2;

async function run(): Promise<number> {
  const config = readImportConfig(process.env);
  mkdirSync(dirname(config.databasePath), { recursive: true });
  const database = openDatabase(config.databasePath);

  try {
    const client = new BingXClient({
      apiKey: config.apiKey,
      apiSecret: config.apiSecret,
      baseUrl: config.baseUrl,
    });
    const positionHistorySource = new BingXPositionHistorySource(client);
    const positionRepository = new SqlitePositionRepository(database);
    const importStateRepository = new SqliteImportStateRepository(database);
    const fullImportStartMs = systemClock.now() - config.fullImportDays * DAY_MS;

    const importResult = await new ImportPositionsUseCase({
      positionHistorySource,
      tradedSymbolSource: new BingXTradedSymbolSource(client),
      positionRepository,
      importStateRepository,
      clock: systemClock,
      windowSizeMs: WINDOW_SIZE_MS,
      overlapMs: OVERLAP_MS,
    }).execute({ fullImportStartMs });
    process.stdout.write(`${formatImportResult(importResult)}\n`);

    const discoveryResult = await new DiscoverHistoricalSymbolsUseCase({
      contractSymbolSource: new BingXContractSymbolSource(client),
      positionHistorySource,
      positionRepository,
      importStateRepository,
      windowSizeMs: WINDOW_SIZE_MS,
    }).execute({
      fullImportStartMs,
      discoveryUntilMs: systemClock.now() - DISCOVERY_MIN_AGE_MS,
      onProgress: (checkedCount, totalCount, symbol) => {
        if (checkedCount % PROGRESS_EVERY_SYMBOLS === 0 || checkedCount === totalCount) {
          process.stderr.write(`  Suche alte Symbole: ${checkedCount}/${totalCount} (${symbol})\n`);
        }
      },
    });
    process.stdout.write(`\n${formatDiscoveryResult(discoveryResult)}\n`);

    if (importResult.failedSymbols.length > 0 || discoveryResult.failedSymbols.length > 0) {
      return EXIT_FAILURE;
    }
    if (importResult.suspiciousSymbols.length > 0) return EXIT_SUSPICIOUS;
    return EXIT_OK;
  } finally {
    database.close();
  }
}

try {
  process.exitCode = await run();
} catch (error: unknown) {
  // Nur die Meldung ausgeben: unsere Fehlertexte enthalten weder Key noch Secret noch signierte URL.
  const message = error instanceof Error ? error.message : String(error);
  process.stderr.write(`Positions-Import fehlgeschlagen: ${message}\n`);
  process.exitCode = EXIT_FAILURE;
}
