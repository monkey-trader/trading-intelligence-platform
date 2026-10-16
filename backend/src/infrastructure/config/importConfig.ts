import { type BingXConfig, readBingXConfig } from '@/infrastructure/config/bingxConfig';

export type ImportConfig = {
  apiKey: string;
  apiSecret: string;
  baseUrl: string;
  databasePath: string;
  fullImportDays: number;
};

type Environment = Readonly<Record<string, string | undefined>>;

const DEFAULT_DATABASE_PATH = 'data/journal.sqlite';
const DEFAULT_FULL_IMPORT_DAYS = 365;

/**
 * Liest die Konfiguration für den Positions-Import. Leere Werte gelten als nicht gesetzt.
 * Fehlertexte nennen nur den Variablennamen, nie einen Wert.
 */
export function readImportConfig(environment: Environment): ImportConfig {
  const bingXConfig = readBingXConfig(environment) ?? failOnMissingKeys(environment);
  return {
    ...bingXConfig,
    databasePath: nonEmpty(environment.JOURNAL_DB) ?? DEFAULT_DATABASE_PATH,
    fullImportDays: parseFullImportDays(nonEmpty(environment.IMPORT_FULL_DAYS)),
  };
}

/** Der Import braucht die Keys zwingend. Die Meldung nennt die fehlende Variable, nie einen Wert. */
function failOnMissingKeys(environment: Environment): BingXConfig {
  for (const variableName of ['BX_KEY', 'BX_SECRET']) {
    if (nonEmpty(environment[variableName]) === undefined) {
      throw new Error(`Umgebungsvariable ${variableName} fehlt (in .env setzen).`);
    }
  }
  throw new Error('BingX-Konfiguration unvollständig (BX_KEY/BX_SECRET in .env setzen).');
}

function nonEmpty(value: string | undefined): string | undefined {
  return value === undefined || value.trim() === '' ? undefined : value.trim();
}

function parseFullImportDays(rawDays: string | undefined): number {
  if (rawDays === undefined) return DEFAULT_FULL_IMPORT_DAYS;
  const days = Number(rawDays);
  if (!/^\d+$/.test(rawDays) || days < 1) {
    throw new Error(`Ungültiges IMPORT_FULL_DAYS: "${rawDays}" (erwartet ganze Zahl > 0).`);
  }
  return days;
}
