import type { OpenPosition, OpenPositionsSnapshot } from '@/domain/position/OpenPosition';
import type { OpenPositionsGateway } from '@/domain/position/OpenPositionsGateway';
import type {
  BackendProblemType,
  OpenPositionsProblemType,
  OpenPositionsResult,
} from '@/domain/position/OpenPositionsResult';
import type { components } from '@/infrastructure/api/generated';

type OpenPositionsSnapshotDto = components['schemas']['OpenPositionsSnapshot'];
type OpenPositionDto = components['schemas']['OpenPosition'];
type RLevelDto = components['schemas']['RLevel'];
type ProblemDto = components['schemas']['Problem'];

const OPEN_POSITIONS_PATH = '/positions/open';
/** Ohne Antwort in dieser Zeit gilt das Backend als nicht erreichbar, damit das Polling nie hängt. */
const DEFAULT_TIMEOUT_MS = 10_000;

export type HttpOpenPositionsGatewayOptions = Readonly<{ timeoutMs?: number }>;
const PROBLEM_CONTENT_TYPE = 'application/problem+json';
const DECIMAL_PATTERN = /^-?\d+(\.\d+)?$/;

const BACKEND_PROBLEM_TYPES: readonly BackendProblemType[] = [
  'forbidden-host',
  'not-configured',
  'rate-limited',
  'unreachable',
  'upstream-error',
  'invalid-response',
  'internal',
] satisfies readonly ProblemDto['type'][];

/**
 * Gateway auf GET /positions/open des eigenen Backends.
 * Standard-Basis ist /api (Vite-Proxy im Dev-Betrieb).
 */
export function createHttpOpenPositionsGateway(
  fetchFn: typeof fetch = fetch,
  baseUrl = '/api',
  options: HttpOpenPositionsGatewayOptions = {},
): OpenPositionsGateway {
  const timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  return {
    async fetchOpenPositions(signal: AbortSignal): Promise<OpenPositionsResult> {
      let response: Response;
      try {
        response = await fetchFn(`${baseUrl}${OPEN_POSITIONS_PATH}`, {
          method: 'GET',
          headers: { Accept: 'application/json' },
          signal: AbortSignal.any([signal, AbortSignal.timeout(timeoutMs)]),
        });
      } catch (error) {
        // Nur ein Abbruch durch den Aufrufer wird weitergeworfen; ein Timeout heißt "nicht erreichbar".
        if (signal.aborted && isAbortError(error)) throw error;
        return problem('backend-unreachable');
      }

      if (response.ok) return readSnapshot(response);
      if (isProblemResponse(response)) return readProblem(response);
      // Fehler ohne problem+json kommt nicht von unserem Backend (z.B. Vite-Proxy ohne Backend).
      return problem('backend-unreachable');
    },
  };
}

async function readSnapshot(response: Response): Promise<OpenPositionsResult> {
  const body = await readJson(response);
  if (body === INVALID_JSON) return problem('invalid-body');
  const snapshotDto = parseSnapshotDto(body);
  if (!snapshotDto) return problem('invalid-body');
  return { kind: 'ok', snapshot: toSnapshot(snapshotDto) };
}

async function readProblem(response: Response): Promise<OpenPositionsResult> {
  const body = await readJson(response);
  if (body === INVALID_JSON) return problem('invalid-body');
  const problemType = parseProblemType(body);
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

function problem(problemType: OpenPositionsProblemType): OpenPositionsResult {
  return { kind: 'problem', problemType };
}

function isProblemResponse(response: Response): boolean {
  const contentType = response.headers.get('Content-Type') ?? '';
  return contentType.toLowerCase().startsWith(PROBLEM_CONTENT_TYPE);
}

/** Nur die Sekunden-Variante von Retry-After; ein HTTP-Datum oder Unsinn wird ignoriert. */
function parseRetryAfterSeconds(headerValue: string | null): number | undefined {
  if (headerValue === null || !/^\d+$/.test(headerValue.trim())) return undefined;
  return Number.parseInt(headerValue.trim(), 10);
}

function parseProblemType(body: unknown): BackendProblemType | undefined {
  if (!isRecord(body)) return undefined;
  const problemType = body['type'];
  return BACKEND_PROBLEM_TYPES.find((knownType) => knownType === problemType);
}

function parseSnapshotDto(body: unknown): OpenPositionsSnapshotDto | undefined {
  if (!isRecord(body)) return undefined;
  const { positions, totalUnrealizedProfit, fetchedAt } = body;
  if (!Array.isArray(positions) || !isDecimal(totalUnrealizedProfit) || !isFiniteNumber(fetchedAt))
    return undefined;
  const positionDtos: OpenPositionDto[] = [];
  for (const candidate of positions) {
    const positionDto = parseOpenPositionDto(candidate);
    if (!positionDto) return undefined;
    positionDtos.push(positionDto);
  }
  return { positions: positionDtos, totalUnrealizedProfit, fetchedAt };
}

function parseOpenPositionDto(candidate: unknown): OpenPositionDto | undefined {
  if (!isRecord(candidate)) return undefined;
  const {
    positionId,
    symbol,
    side,
    isolated,
    positionAmount,
    avgOpenPrice,
    markPrice,
    leverage,
    initialMargin,
    margin,
    positionValue,
    unrealizedProfit,
    realisedProfit,
    pnlRatio,
    stopLoss,
    takeProfit,
    breakEven,
    openTime,
    updateTime,
  } = candidate;
  const { initialStopLoss, riskDistance, currentRMultiple, breakEvenTriggerReached, reached2RAt } =
    candidate;
  const matchesContract =
    typeof positionId === 'string' &&
    positionId !== '' &&
    typeof symbol === 'string' &&
    (side === 'LONG' || side === 'SHORT') &&
    typeof isolated === 'boolean' &&
    isDecimal(positionAmount) &&
    isDecimal(avgOpenPrice) &&
    isDecimal(markPrice) &&
    isFiniteNumber(leverage) &&
    isDecimal(initialMargin) &&
    isDecimal(margin) &&
    isDecimal(positionValue) &&
    isDecimal(unrealizedProfit) &&
    isDecimal(realisedProfit) &&
    isDecimal(pnlRatio) &&
    (stopLoss === null || isDecimal(stopLoss)) &&
    (takeProfit === null || isDecimal(takeProfit)) &&
    (breakEven === null || typeof breakEven === 'boolean') &&
    isFiniteNumber(openTime) &&
    isFiniteNumber(updateTime);
  if (!matchesContract) return undefined;

  return {
    positionId,
    symbol,
    side,
    isolated,
    positionAmount,
    avgOpenPrice,
    markPrice,
    leverage,
    initialMargin,
    margin,
    positionValue,
    unrealizedProfit,
    realisedProfit,
    pnlRatio,
    stopLoss,
    takeProfit,
    breakEven,
    openTime,
    updateTime,
    initialStopLoss: isDecimal(initialStopLoss) ? initialStopLoss : null,
    initialStopLossSource:
      candidate['initialStopLossSource'] === 'AUTO' ||
      candidate['initialStopLossSource'] === 'MANUAL'
        ? candidate['initialStopLossSource']
        : null,
    riskDistance: isDecimal(riskDistance) ? riskDistance : null,
    currentRMultiple: isDecimal(currentRMultiple) ? currentRMultiple : null,
    breakEvenTriggerReached:
      typeof breakEvenTriggerReached === 'boolean' ? breakEvenTriggerReached : null,
    reached2RAt: isFiniteNumber(reached2RAt) ? reached2RAt : null,
    ...withLevels(parseLevels(candidate['levels'])),
  };
}

function withLevels(levels: RLevelDto[] | undefined): { levels?: RLevelDto[] } {
  return levels ? { levels } : {};
}

/** Defensiv: alles Unpassende (fehlend, leer, ungültig) ergibt undefined. */
function parseLevels(value: unknown): RLevelDto[] | undefined {
  if (!Array.isArray(value) || value.length === 0) return undefined;
  const levels: RLevelDto[] = [];
  for (const entry of value) {
    if (!isRecord(entry)) return undefined;
    const { kind, rMultiple, price, percent } = entry;
    if (kind === 'ONE_R') continue;
    if (
      (kind !== 'TWO_R' && kind !== 'THREE_R') ||
      (rMultiple !== 2 && rMultiple !== 3) ||
      !isDecimal(price) ||
      !isDecimal(percent)
    )
      return undefined;
    levels.push({ kind, rMultiple, price, percent });
  }
  return levels;
}

function toSnapshot(snapshotDto: OpenPositionsSnapshotDto): OpenPositionsSnapshot {
  return {
    positions: snapshotDto.positions.map(toOpenPosition),
    totalUnrealizedProfit: snapshotDto.totalUnrealizedProfit,
    fetchedAt: snapshotDto.fetchedAt,
  };
}

function toOpenPosition(positionDto: OpenPositionDto): OpenPosition {
  return {
    positionId: positionDto.positionId,
    symbol: positionDto.symbol,
    side: positionDto.side,
    isolated: positionDto.isolated,
    positionAmount: positionDto.positionAmount,
    avgOpenPrice: positionDto.avgOpenPrice,
    markPrice: positionDto.markPrice,
    leverage: positionDto.leverage,
    initialMargin: positionDto.initialMargin,
    margin: positionDto.margin,
    positionValue: positionDto.positionValue,
    unrealizedProfit: positionDto.unrealizedProfit,
    realisedProfit: positionDto.realisedProfit,
    pnlRatio: positionDto.pnlRatio,
    stopLoss: positionDto.stopLoss,
    takeProfit: positionDto.takeProfit,
    breakEven: positionDto.breakEven,
    openTime: positionDto.openTime,
    updateTime: positionDto.updateTime,
    initialStopLoss: positionDto.initialStopLoss ?? null,
    initialStopLossSource: positionDto.initialStopLossSource ?? null,
    riskDistance: positionDto.riskDistance ?? null,
    currentRMultiple: positionDto.currentRMultiple ?? null,
    breakEvenTriggerReached: positionDto.breakEvenTriggerReached ?? null,
    reached2RAt: positionDto.reached2RAt ?? null,
    ...(positionDto.levels && positionDto.levels.length > 0 ? { levels: positionDto.levels } : {}),
  };
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

/** Abbruch über das AbortSignal (DOMException oder Error mit name AbortError). */
function isAbortError(error: unknown): boolean {
  return (
    typeof error === 'object' && error !== null && 'name' in error && error.name === 'AbortError'
  );
}
