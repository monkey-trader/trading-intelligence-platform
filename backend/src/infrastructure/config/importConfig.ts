export type ImportConfig = {
  apiKey: string;
  apiSecret: string;
  baseUrl: string;
  databasePath: string;
  fullImportDays: number;
};

type Environment = Readonly<Record<string, string | undefined>>;

const DEFAULT_BASE_URL = 'https://open-api.bingx.com';
const DEFAULT_DATABASE_PATH = 'data/journal.sqlite';
const DEFAULT_FULL_IMPORT_DAYS = 365;

/**
 * Liest die Konfiguration für den Positions-Import. Leere Werte gelten als nicht gesetzt.
 * Fehlertexte nennen nur den Variablennamen, nie einen Wert.
 */
export function readImportConfig(environment: Environment): ImportConfig {
  return {
    apiKey: required(environment, 'BX_KEY'),
    apiSecret: required(environment, 'BX_SECRET'),
    baseUrl: nonEmpty(environment.BX_BASE) ?? DEFAULT_BASE_URL,
    databasePath: nonEmpty(environment.JOURNAL_DB) ?? DEFAULT_DATABASE_PATH,
    fullImportDays: parseFullImportDays(nonEmpty(environment.IMPORT_FULL_DAYS)),
  };
}

function required(environment: Environment, variableName: string): string {
  const value = nonEmpty(environment[variableName]);
  if (value === undefined) {
    throw new Error(`Umgebungsvariable ${variableName} fehlt (in .env setzen).`);
  }
  return value;
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
