import type { PositionPrice, PriceSnapshot, PriceStreamGateway } from '@/domain/price/PriceStream';

/** Der Teil von EventSource, den das Gateway braucht (für Tests ersetzbar). */
export type EventSourceLike = {
  addEventListener(type: string, listener: EventListener): void;
  close(): void;
  onerror: ((event: Event) => void) | null;
  onopen: ((event: Event) => void) | null;
};
export type EventSourceFactory = (url: string) => EventSourceLike;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function stringOrNull(value: unknown): string | null | undefined {
  if (value === null) return null;
  return typeof value === 'string' ? value : undefined;
}

function toPositionPrice(value: unknown): PositionPrice | undefined {
  if (!isRecord(value)) return undefined;
  const { positionId, symbol, price, change24hPercent } = value;
  const distanceToBreakEvenPercent = stringOrNull(value['distanceToBreakEvenPercent']);
  const distanceToTp1Percent = stringOrNull(value['distanceToTp1Percent']);
  if (
    typeof positionId !== 'string' ||
    typeof symbol !== 'string' ||
    typeof price !== 'string' ||
    typeof change24hPercent !== 'string' ||
    distanceToBreakEvenPercent === undefined ||
    distanceToTp1Percent === undefined
  ) {
    return undefined;
  }
  return {
    positionId,
    symbol,
    price,
    change24hPercent,
    distanceToBreakEvenPercent,
    distanceToTp1Percent,
  };
}

/** Prüft die Nutzlast gegen PriceSnapshot; ein ungültiger Eintrag verwirft das ganze Event. */
function parseSnapshot(data: unknown): PriceSnapshot | undefined {
  if (typeof data !== 'string') return undefined;
  let parsed: unknown;
  try {
    parsed = JSON.parse(data);
  } catch {
    return undefined;
  }
  if (!isRecord(parsed)) return undefined;
  const { prices, fetchedAt, stale } = parsed;
  if (!Array.isArray(prices) || typeof fetchedAt !== 'number' || typeof stale !== 'boolean') {
    return undefined;
  }
  const entries = prices.map(toPositionPrice);
  const valid = entries.filter((entry): entry is PositionPrice => entry !== undefined);
  if (valid.length !== entries.length) return undefined;
  return { prices: valid, fetchedAt, stale };
}

/**
 * Kurs-Stream des Backends (GET {baseUrl}/prices/stream, Event "prices" = PriceSnapshot).
 * Verbinden nach einem Fehler übernimmt EventSource selbst; das Gateway meldet nur den Status.
 */
export function createHttpPriceStreamGateway(
  createEventSource: EventSourceFactory = (url) => new EventSource(url),
  baseUrl = '/api',
): PriceStreamGateway {
  return {
    subscribe: ({ onSnapshot, onStatus }) => {
      const source = createEventSource(`${baseUrl}/prices/stream`);
      source.onopen = () => onStatus('live');
      source.onerror = () => onStatus('ausgefallen');
      source.addEventListener('prices', (event) => {
        const snapshot = parseSnapshot(event instanceof MessageEvent ? event.data : undefined);
        if (snapshot) onSnapshot(snapshot);
      });
      return () => source.close();
    },
  };
}
