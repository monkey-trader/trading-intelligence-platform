import type { BingXGetClient, BingXQueryParams } from '@/infrastructure/bingx/BingXGetClient';
import {
  BingXApiError,
  type BingXError,
  BingXHttpError,
  BingXNetworkError,
  BingXRateLimitError,
  UnexpectedBingXResponseError,
} from '@/infrastructure/bingx/errors';
import { signQuery } from '@/infrastructure/bingx/signing';

type FetchFunction = (input: string | URL | Request, init?: RequestInit) => Promise<Response>;

export type BingXClientOptions = Readonly<{
  apiKey: string;
  apiSecret: string;
  baseUrl: string;
  fetchFn?: FetchFunction;
  sleep?: (milliseconds: number) => Promise<void>;
  now?: () => number;
  /** Wiederholungen bei vorübergehenden Fehlern (zusätzlich zum ersten Versuch). */
  maxRetries?: number;
  /** Feste Pause nach jedem erfolgreichen Request, schont das Rate-Limit. */
  requestPauseMs?: number;
  /** Erste Wartezeit vor einem Retry, verdoppelt sich mit jedem Retry. */
  initialBackoffMs?: number;
  /** Abbruch eines einzelnen Requests nach dieser Zeit. */
  requestTimeoutMs?: number;
}>;

const HTTP_TOO_MANY_REQUESTS = 429;
const BINGX_RATE_LIMIT_CODE = 100410;
/** Vorübergehende BingX-Fehler bei HTTP 200: Rate-Limit, interner Fehler, Server ausgelastet. */
const RETRYABLE_API_CODES: ReadonlySet<number> = new Set([BINGX_RATE_LIMIT_CODE, 100500, 100503]);
const DEFAULT_REQUEST_TIMEOUT_MS = 15_000;
const DEFAULT_MAX_RETRIES = 5;
const DEFAULT_REQUEST_PAUSE_MS = 250;
const DEFAULT_INITIAL_BACKOFF_MS = 1_000;

const defaultSleep = (milliseconds: number): Promise<void> =>
  new Promise((resolve) => setTimeout(resolve, milliseconds));

/**
 * Minimaler BingX-Client: nur signierte GET-Requests.
 * Jeder Versuch bekommt einen neuen timestamp und eine neue Signatur.
 */
export class BingXClient implements BingXGetClient {
  private readonly fetchFn: FetchFunction;
  private readonly sleep: (milliseconds: number) => Promise<void>;
  private readonly now: () => number;
  private readonly maxRetries: number;
  private readonly requestPauseMs: number;
  private readonly initialBackoffMs: number;
  private readonly requestTimeoutMs: number;

  constructor(private readonly options: BingXClientOptions) {
    this.fetchFn = options.fetchFn ?? fetch;
    this.sleep = options.sleep ?? defaultSleep;
    this.now = options.now ?? Date.now;
    this.maxRetries = options.maxRetries ?? DEFAULT_MAX_RETRIES;
    this.requestPauseMs = options.requestPauseMs ?? DEFAULT_REQUEST_PAUSE_MS;
    this.initialBackoffMs = options.initialBackoffMs ?? DEFAULT_INITIAL_BACKOFF_MS;
    this.requestTimeoutMs = options.requestTimeoutMs ?? DEFAULT_REQUEST_TIMEOUT_MS;
  }

  async get(path: string, params: BingXQueryParams): Promise<unknown> {
    const attemptCount = this.maxRetries + 1;
    let lastFailure: BingXError | undefined;

    for (let attempt = 0; attempt < attemptCount; attempt += 1) {
      if (attempt > 0) await this.sleep(this.initialBackoffMs * 2 ** (attempt - 1));

      const outcome = await this.attempt(path, params, attemptCount);
      if (outcome.kind === 'success') {
        if (this.requestPauseMs > 0) await this.sleep(this.requestPauseMs);
        return outcome.data;
      }
      lastFailure = outcome.error;
    }
    throw lastFailure ?? new BingXRateLimitError(path, attemptCount);
  }

  /** Ein einzelner Versuch mit frischem timestamp und frischer Signatur. */
  private async attempt(
    path: string,
    params: BingXQueryParams,
    attemptCount: number,
  ): Promise<AttemptOutcome> {
    let response: Response;
    try {
      response = await this.fetchFn(this.signedUrl(path, params), {
        method: 'GET',
        headers: { 'X-BX-APIKEY': this.options.apiKey },
        signal: AbortSignal.timeout(this.requestTimeoutMs),
      });
    } catch (error: unknown) {
      const causeName = error instanceof Error ? error.name : 'Unbekannt';
      return { kind: 'retry', error: new BingXNetworkError(path, attemptCount, causeName) };
    }

    if (response.status === HTTP_TOO_MANY_REQUESTS) {
      return { kind: 'retry', error: new BingXRateLimitError(path, attemptCount) };
    }
    if (response.status >= 500) {
      return { kind: 'retry', error: new BingXHttpError(response.status, path) };
    }
    if (!response.ok) throw new BingXHttpError(response.status, path);

    try {
      return { kind: 'success', data: parseEnvelope(await readJson(response, path), path) };
    } catch (error: unknown) {
      if (error instanceof BingXApiError && RETRYABLE_API_CODES.has(error.code)) {
        const retryError =
          error.code === BINGX_RATE_LIMIT_CODE
            ? new BingXRateLimitError(path, attemptCount)
            : error;
        return { kind: 'retry', error: retryError };
      }
      throw error;
    }
  }

  private signedUrl(path: string, params: BingXQueryParams): string {
    const query = Object.entries({ ...params, timestamp: this.now() })
      .map(([name, value]) => `${encodeURIComponent(name)}=${encodeURIComponent(String(value))}`)
      .join('&');
    return `${this.options.baseUrl}${path}?${query}&signature=${signQuery(query, this.options.apiSecret)}`;
  }
}

type AttemptOutcome = { kind: 'success'; data: unknown } | { kind: 'retry'; error: BingXError };

async function readJson(response: Response, path: string): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    throw new UnexpectedBingXResponseError(`BingX lieferte kein gültiges JSON bei ${path}.`);
  }
}

/** Prüft die Hülle `{ code, msg, data }` und liefert `data`. */
function parseEnvelope(body: unknown, path: string): unknown {
  if (typeof body !== 'object' || body === null || !('code' in body)) {
    throw new UnexpectedBingXResponseError(`BingX-Antwort ohne Feld code bei ${path}.`);
  }
  const { code } = body;
  if (typeof code !== 'number') {
    throw new UnexpectedBingXResponseError(`BingX-Antwort mit ungültigem code bei ${path}.`);
  }
  if (code !== 0) {
    const apiMessage = 'msg' in body && typeof body.msg === 'string' ? body.msg : '';
    throw new BingXApiError(code, apiMessage, path);
  }
  return 'data' in body ? body.data : undefined;
}
