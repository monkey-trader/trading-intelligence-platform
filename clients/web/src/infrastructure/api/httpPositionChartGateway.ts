import {
  CHART_INTERVALS,
  type Candle,
  type ChartInterval,
  type ChartMarker,
  type PositionChart,
  type ProtectiveOrder,
  type RLevel,
} from '@/domain/position/PositionChart';
import type { PositionChartGateway } from '@/domain/position/PositionChartGateway';
import type {
  PositionChartProblemType,
  PositionChartResult,
} from '@/domain/position/PositionChartResult';

/** Ohne Antwort in dieser Zeit gilt das Backend als nicht erreichbar. */
const DEFAULT_TIMEOUT_MS = 10_000;
const PROBLEM_CONTENT_TYPE = 'application/problem+json';
const DECIMAL_PATTERN = /^-?\d+(\.\d+)?$/;

export type HttpPositionChartGatewayOptions = Readonly<{ timeoutMs?: number }>;

/** Problem-Typen, die das Backend liefern darf (laut API-Vertrag). */
const BACKEND_PROBLEM_TYPES: readonly PositionChartProblemType[] = [
  'bad-request',
  'not-found',
  'forbidden-host',
  'not-configured',
  'rate-limited',
  'unreachable',
  'upstream-error',
  'invalid-response',
  'internal',
];

/** Gateway auf GET /positions/{id}/chart des eigenen Backends. */
export function createHttpPositionChartGateway(
  fetchFn: typeof fetch = fetch,
  baseUrl = '/api',
  options: HttpPositionChartGatewayOptions = {},
): PositionChartGateway {
  const timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  return {
    async fetchChart(positionId, interval, limit, signal): Promise<PositionChartResult> {
      const url = `${baseUrl}/positions/${encodeURIComponent(positionId)}/chart?interval=${interval}&limit=${limit}`;
      let response: Response;
      try {
        response = await fetchFn(url, {
          method: 'GET',
          headers: { Accept: 'application/json' },
          signal: AbortSignal.any([signal, AbortSignal.timeout(timeoutMs)]),
        });
      } catch (error) {
        // Nur ein Abbruch durch den Aufrufer wird weitergeworfen; ein Timeout heißt "nicht erreichbar".
        if (signal.aborted && isAbortError(error)) throw error;
        return problem('backend-unreachable');
      }

      if (response.ok) return readChart(response);
      if (isProblemResponse(response)) return readProblem(response);
      return problem('backend-unreachable');
    },
  };
}

async function readChart(response: Response): Promise<PositionChartResult> {
  const body = await readJson(response);
  if (body === INVALID_JSON) return problem('invalid-body');
  const chart = parseChart(body);
  return chart ? { kind: 'ok', chart } : problem('invalid-body');
}

async function readProblem(response: Response): Promise<PositionChartResult> {
  const body = await readJson(response);
  if (body === INVALID_JSON) return problem('invalid-body');
  const problemType = isRecord(body)
    ? BACKEND_PROBLEM_TYPES.find((knownType) => knownType === body['type'])
    : undefined;
  if (!problemType) return problem('invalid-body');
  const retryAfterSeconds = parseRetryAfterSeconds(response.headers.get('Retry-After'));
  return retryAfterSeconds === undefined
    ? problem(problemType)
    : { kind: 'problem', problemType, retryAfterSeconds };
}

const INVALID_JSON = Symbol('invalid-json');

async function readJson(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch (error) {
    if (isAbortError(error)) throw error;
    return INVALID_JSON;
  }
}

function problem(problemType: PositionChartProblemType): PositionChartResult {
  return { kind: 'problem', problemType };
}

function isProblemResponse(response: Response): boolean {
  const contentType = response.headers.get('Content-Type') ?? '';
  return contentType.toLowerCase().startsWith(PROBLEM_CONTENT_TYPE);
}

function parseRetryAfterSeconds(headerValue: string | null): number | undefined {
  if (headerValue === null || !/^\d+$/.test(headerValue.trim())) return undefined;
  return Number.parseInt(headerValue.trim(), 10);
}

function parseChart(body: unknown): PositionChart | undefined {
  if (!isRecord(body)) return undefined;
  const { positionId, symbol, side, interval, entry, fetchedAt } = body;
  const candles = parseList(body['candles'], parseCandle);
  const stopLoss = parseList(body['stopLoss'], parseProtectiveOrder);
  const takeProfit = parseList(body['takeProfit'], parseProtectiveOrder);
  const markers = parseList(body['markers'], parseMarker);
  const levels = parseLevels(body['levels']);
  const parsedEntry = parseEntry(entry);
  if (
    typeof positionId !== 'string' ||
    positionId === '' ||
    typeof symbol !== 'string' ||
    (side !== 'LONG' && side !== 'SHORT') ||
    !isChartInterval(interval) ||
    !candles ||
    !stopLoss ||
    !takeProfit ||
    !markers ||
    !parsedEntry ||
    !isFiniteNumber(fetchedAt)
  ) {
    return undefined;
  }
  return {
    positionId,
    symbol,
    side,
    interval,
    candles,
    entry: parsedEntry,
    stopLoss,
    takeProfit,
    markers,
    levels,
    fetchedAt,
  };
}

function parseList<Item>(
  candidate: unknown,
  parseItem: (item: unknown) => Item | undefined,
): Item[] | undefined {
  if (!Array.isArray(candidate)) return undefined;
  const items: Item[] = [];
  for (const rawItem of candidate) {
    const item = parseItem(rawItem);
    if (item === undefined) return undefined;
    items.push(item);
  }
  return items;
}

/** Defensiv: fehlende oder kaputte Level-Liste heißt "keine Level", nie ein Fehler. */
function parseLevels(candidate: unknown): RLevel[] | undefined {
  if (!Array.isArray(candidate)) return undefined;
  const levels: RLevel[] = [];
  for (const rawLevel of candidate) {
    const level = parseLevel(rawLevel);
    if (level) levels.push(level);
  }
  return levels.length > 0 ? levels : undefined;
}

function parseLevel(candidate: unknown): RLevel | undefined {
  if (!isRecord(candidate)) return undefined;
  const { kind, rMultiple, price, percent, coincidesWithTakeProfit } = candidate;
  const multiple = typeof rMultiple === 'number' ? String(rMultiple) : rMultiple;
  return (kind === 'TWO_R' || kind === 'THREE_R') &&
    isDecimal(multiple) &&
    isDecimal(price) &&
    isDecimal(percent)
    ? {
        kind,
        rMultiple: multiple,
        price,
        percent,
        ...(typeof coincidesWithTakeProfit === 'boolean' ? { coincidesWithTakeProfit } : {}),
      }
    : undefined;
}

function parseEntry(candidate: unknown): PositionChart['entry'] | undefined {
  if (!isRecord(candidate)) return undefined;
  const { price, time } = candidate;
  return isDecimal(price) && isFiniteNumber(time) ? { price, time } : undefined;
}

function parseCandle(candidate: unknown): Candle | undefined {
  if (!isRecord(candidate)) return undefined;
  const { time, open, high, low, close, volume } = candidate;
  return isFiniteNumber(time) &&
    isDecimal(open) &&
    isDecimal(high) &&
    isDecimal(low) &&
    isDecimal(close) &&
    isDecimal(volume)
    ? { time, open, high, low, close, volume }
    : undefined;
}

function parseProtectiveOrder(candidate: unknown): ProtectiveOrder | undefined {
  if (!isRecord(candidate)) return undefined;
  const { orderId, triggerPrice, quantity, distanceRatio } = candidate;
  return typeof orderId === 'string' &&
    isDecimal(triggerPrice) &&
    isDecimal(quantity) &&
    isDecimal(distanceRatio)
    ? { orderId, triggerPrice, quantity, distanceRatio }
    : undefined;
}

function parseMarker(candidate: unknown): ChartMarker | undefined {
  if (!isRecord(candidate)) return undefined;
  const { time, kind, price } = candidate;
  return isFiniteNumber(time) && kind === 'ENTRY' && isDecimal(price)
    ? { time, kind, price }
    : undefined;
}

function isChartInterval(value: unknown): value is ChartInterval {
  return CHART_INTERVALS.some((knownInterval) => knownInterval === value);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isDecimal(value: unknown): value is string {
  return typeof value === 'string' && DECIMAL_PATTERN.test(value);
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value);
}

function isAbortError(error: unknown): boolean {
  return (
    typeof error === 'object' && error !== null && 'name' in error && error.name === 'AbortError'
  );
}
