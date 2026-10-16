export type BingXConfig = { apiKey: string; apiSecret: string; baseUrl: string };

type Environment = Readonly<Record<string, string | undefined>>;

const DEFAULT_BASE_URL = 'https://open-api.bingx.com';

/**
 * Liest die BingX-Zugangsdaten. Fehlen BX_KEY oder BX_SECRET (oder sind sie leer), kommt
 * `undefined` zurück, ohne zu werfen. Ob das ein Fehler ist, entscheidet der Aufrufer.
 */
export function readBingXConfig(environment: Environment): BingXConfig | undefined {
  const apiKey = nonEmpty(environment.BX_KEY);
  const apiSecret = nonEmpty(environment.BX_SECRET);
  if (apiKey === undefined || apiSecret === undefined) return undefined;
  return { apiKey, apiSecret, baseUrl: nonEmpty(environment.BX_BASE) ?? DEFAULT_BASE_URL };
}

/** Namen der fehlenden Pflichtvariablen, für Warnungen. Gibt nie Werte zurück. */
export function missingBingXVariables(environment: Environment): string[] {
  return (['BX_KEY', 'BX_SECRET'] as const).filter(
    (variableName) => nonEmpty(environment[variableName]) === undefined,
  );
}

function nonEmpty(value: string | undefined): string | undefined {
  return value === undefined || value.trim() === '' ? undefined : value.trim();
}
