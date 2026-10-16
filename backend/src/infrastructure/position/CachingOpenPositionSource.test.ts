import { describe, expect, it } from 'vitest';
import { CachingOpenPositionSource } from '@/infrastructure/position/CachingOpenPositionSource';
import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import type { OpenPositionsSnapshot } from '@/domain/position/entities/OpenPositionsSnapshot';
import type { OpenPositionSource } from '@/domain/position/interfaces/OpenPositionSource';
import type { Clock } from '@/domain/shared/interfaces/Clock';
import { OpenPositionsUnavailableError } from '@/domain/position/errors/OpenPositionsUnavailableError';

const TTL_MS = 5_000;
const RATE_LIMIT_HOLD_MS = 30_000;
const START_MS = 1_791_000_000_000;

const suiPosition: OpenPosition = {
  positionId: '9100000000000000001',
  symbol: 'SUI-USDT',
  side: 'LONG',
  isolated: true,
  positionAmount: '500',
  avgOpenPrice: '0.9600',
  markPrice: '1.0100',
  leverage: 10,
  initialMargin: '48.0000',
  margin: '60.5000',
  positionValue: '505',
  unrealizedProfit: '25.0000',
  realisedProfit: '-0.2400',
  pnlRatio: '0.5208',
  liquidationPrice: '0.865',
  openTime: 1791500000000,
  updateTime: 1791600000000,
};

/** Fake-Clock: Zeit wird nur über advance() verändert, keine echten Timer. */
function createFakeClock(startMs: number = START_MS) {
  let currentMs = startMs;
  const clock: Clock = { now: () => currentMs };
  return {
    clock,
    advance: (deltaMs: number): void => {
      currentMs += deltaMs;
    },
  };
}

type SourceOutcome =
  { kind: 'resolve'; snapshot: OpenPositionsSnapshot } | { kind: 'reject'; error: unknown };

/**
 * Fake-Source: zählt die Aufrufe und arbeitet eine Liste vorbereiteter Ergebnisse ab.
 * Das letzte Ergebnis wiederholt sich. fetchedAt kommt aus der Fake-Clock zum Aufrufzeitpunkt.
 */
function createCountingSource(clock: Clock, outcomes: readonly ('ok' | Error)[]) {
  let callCount = 0;
  const source: OpenPositionSource = {
    async fetchOpenPositions(): Promise<OpenPositionsSnapshot> {
      const outcome = outcomes[Math.min(callCount, outcomes.length - 1)] ?? 'ok';
      callCount += 1;
      if (outcome !== 'ok') throw outcome;
      return { positions: [suiPosition], fetchedAt: clock.now() };
    },
  };
  return { source, getCallCount: () => callCount };
}

/** Fake-Source, deren Antwort der Test erst später auflöst (für gleichzeitige Aufrufe). */
function createDeferredSource() {
  let callCount = 0;
  let settle: ((outcome: SourceOutcome) => void) | undefined;
  const source: OpenPositionSource = {
    fetchOpenPositions(): Promise<OpenPositionsSnapshot> {
      callCount += 1;
      return new Promise<OpenPositionsSnapshot>((resolve, reject) => {
        settle = (outcome) => {
          if (outcome.kind === 'resolve') resolve(outcome.snapshot);
          else reject(outcome.error);
        };
      });
    },
  };
  return {
    source,
    getCallCount: () => callCount,
    settle: (outcome: SourceOutcome): void => {
      if (settle === undefined) throw new Error('Die Source wurde noch nicht aufgerufen.');
      settle(outcome);
    },
  };
}

function createCache(source: OpenPositionSource, clock: Clock): CachingOpenPositionSource {
  return new CachingOpenPositionSource(source, clock, {
    ttlMs: TTL_MS,
    rateLimitHoldMs: RATE_LIMIT_HOLD_MS,
  });
}

/** Lässt ausstehende Microtasks laufen, damit die Source sicher aufgerufen wurde. */
function flushPendingWork(): Promise<void> {
  return new Promise((resolve) => setImmediate(resolve));
}

async function captureRejection(action: () => Promise<unknown>): Promise<unknown> {
  try {
    await action();
  } catch (error) {
    return error;
  }
  throw new Error('Es wurde ein Fehler erwartet, aber keiner geworfen.');
}

describe('CachingOpenPositionSource', () => {
  it('implementiert den Port OpenPositionSource', () => {
    const { clock } = createFakeClock();
    const { source } = createCountingSource(clock, ['ok']);

    const port: OpenPositionSource = createCache(source, clock);

    expect(typeof port.fetchOpenPositions).toBe('function');
  });

  it('reicht den Snapshot der Source beim ersten Aufruf durch', async () => {
    const { clock } = createFakeClock();
    const { source, getCallCount } = createCountingSource(clock, ['ok']);
    const cache = createCache(source, clock);

    const snapshot = await cache.fetchOpenPositions();

    expect(snapshot).toEqual({ positions: [suiPosition], fetchedAt: START_MS });
    expect(getCallCount()).toBe(1);
  });

  it('fragt innerhalb der TTL nicht erneut und behält das ursprüngliche fetchedAt', async () => {
    const { clock, advance } = createFakeClock();
    const { source, getCallCount } = createCountingSource(clock, ['ok']);
    const cache = createCache(source, clock);

    const firstSnapshot = await cache.fetchOpenPositions();
    advance(TTL_MS - 1);
    const secondSnapshot = await cache.fetchOpenPositions();

    expect(getCallCount()).toBe(1);
    expect(secondSnapshot.fetchedAt).toBe(START_MS);
    expect(secondSnapshot).toEqual(firstSnapshot);
  });

  it('fragt nach Ablauf der TTL erneut und liefert das neue fetchedAt', async () => {
    const { clock, advance } = createFakeClock();
    const { source, getCallCount } = createCountingSource(clock, ['ok']);
    const cache = createCache(source, clock);

    await cache.fetchOpenPositions();
    advance(TTL_MS + 1);
    const refreshedSnapshot = await cache.fetchOpenPositions();

    expect(getCallCount()).toBe(2);
    expect(refreshedSnapshot.fetchedAt).toBe(START_MS + TTL_MS + 1);
  });

  it('cacht einen allgemeinen Fehler nicht: der nächste Aufruf fragt erneut', async () => {
    const { clock } = createFakeClock();
    const failure = new Error('Programmierfehler');
    const { source, getCallCount } = createCountingSource(clock, [failure, 'ok']);
    const cache = createCache(source, clock);

    await expect(cache.fetchOpenPositions()).rejects.toBe(failure);
    const snapshot = await cache.fetchOpenPositions();

    expect(getCallCount()).toBe(2);
    expect(snapshot.positions).toEqual([suiPosition]);
  });

  it('cacht OpenPositionsUnavailableError mit anderem reason als rate-limited nicht', async () => {
    const { clock } = createFakeClock();
    const unreachable = new OpenPositionsUnavailableError('unreachable');
    const { source, getCallCount } = createCountingSource(clock, [unreachable, 'ok']);
    const cache = createCache(source, clock);

    await expect(cache.fetchOpenPositions()).rejects.toBe(unreachable);
    await expect(cache.fetchOpenPositions()).resolves.toEqual({
      positions: [suiPosition],
      fetchedAt: START_MS,
    });
    expect(getCallCount()).toBe(2);
  });

  it('wirft rate-limited während rateLimitHoldMs erneut, ohne die Source zu fragen', async () => {
    const { clock, advance } = createFakeClock();
    const rateLimited = new OpenPositionsUnavailableError('rate-limited');
    const { source, getCallCount } = createCountingSource(clock, [rateLimited, 'ok']);
    const cache = createCache(source, clock);

    await expect(cache.fetchOpenPositions()).rejects.toBe(rateLimited);
    advance(1);
    const heldAtStart = await captureRejection(() => cache.fetchOpenPositions());
    advance(RATE_LIMIT_HOLD_MS - 2);
    const heldAtEnd = await captureRejection(() => cache.fetchOpenPositions());

    expect(getCallCount()).toBe(1);
    for (const held of [heldAtStart, heldAtEnd]) {
      expect(held).toBeInstanceOf(OpenPositionsUnavailableError);
      expect(held instanceof OpenPositionsUnavailableError ? held.reason : undefined).toBe(
        'rate-limited',
      );
    }
  });

  it('fragt nach Ablauf von rateLimitHoldMs wieder bei der Source', async () => {
    const { clock, advance } = createFakeClock();
    const rateLimited = new OpenPositionsUnavailableError('rate-limited');
    const { source, getCallCount } = createCountingSource(clock, [rateLimited, 'ok']);
    const cache = createCache(source, clock);

    await expect(cache.fetchOpenPositions()).rejects.toBeInstanceOf(OpenPositionsUnavailableError);
    advance(RATE_LIMIT_HOLD_MS + 1);
    const snapshot = await cache.fetchOpenPositions();

    expect(getCallCount()).toBe(2);
    expect(snapshot).toEqual({
      positions: [suiPosition],
      fetchedAt: START_MS + RATE_LIMIT_HOLD_MS + 1,
    });
  });

  it('teilt zwei gleichzeitige Aufrufe auf einen Source-Aufruf', async () => {
    const { clock } = createFakeClock();
    const { source, getCallCount, settle } = createDeferredSource();
    const cache = createCache(source, clock);
    const expectedSnapshot: OpenPositionsSnapshot = {
      positions: [suiPosition],
      fetchedAt: START_MS,
    };

    const firstCall = cache.fetchOpenPositions();
    const secondCall = cache.fetchOpenPositions();
    await flushPendingWork();
    settle({ kind: 'resolve', snapshot: expectedSnapshot });

    await expect(firstCall).resolves.toEqual(expectedSnapshot);
    await expect(secondCall).resolves.toEqual(expectedSnapshot);
    expect(getCallCount()).toBe(1);
  });

  it('teilt bei gleichzeitigen Aufrufen auch dieselbe Rejection und gibt danach frei', async () => {
    const { clock } = createFakeClock();
    const { source, getCallCount, settle } = createDeferredSource();
    const cache = createCache(source, clock);
    const failure = new OpenPositionsUnavailableError('upstream-error');

    const firstCall = captureRejection(() => cache.fetchOpenPositions());
    const secondCall = captureRejection(() => cache.fetchOpenPositions());
    await flushPendingWork();
    settle({ kind: 'reject', error: failure });

    expect(await firstCall).toBe(failure);
    expect(await secondCall).toBe(failure);
    expect(getCallCount()).toBe(1);

    const thirdCall = cache.fetchOpenPositions();
    await flushPendingWork();
    expect(getCallCount()).toBe(2);
    settle({ kind: 'resolve', snapshot: { positions: [], fetchedAt: START_MS } });
    await expect(thirdCall).resolves.toEqual({ positions: [], fetchedAt: START_MS });
  });

  it('löscht den rate-limited-Hold nach einem Erfolg, ein späteres allgemeines Problem wird nicht zurückgehalten', async () => {
    const { clock, advance } = createFakeClock();
    const rateLimited = new OpenPositionsUnavailableError('rate-limited');
    const upstreamError = new OpenPositionsUnavailableError('upstream-error');
    const { source, getCallCount } = createCountingSource(clock, [
      rateLimited,
      'ok',
      upstreamError,
      'ok',
    ]);
    const cache = createCache(source, clock);

    await expect(cache.fetchOpenPositions()).rejects.toBe(rateLimited);
    advance(RATE_LIMIT_HOLD_MS + 1);
    await cache.fetchOpenPositions();
    advance(TTL_MS + 1);
    await expect(cache.fetchOpenPositions()).rejects.toBe(upstreamError);
    await cache.fetchOpenPositions();

    expect(getCallCount()).toBe(4);
  });

  it('teilt einen laufenden Fetch auch mit einem Aufruf nach Ablauf der TTL', async () => {
    const { clock, advance } = createFakeClock();
    const { source, getCallCount, settle } = createDeferredSource();
    const cache = createCache(source, clock);

    const firstCall = cache.fetchOpenPositions();
    await flushPendingWork();
    advance(TTL_MS + 1);
    const lateCall = cache.fetchOpenPositions();
    settle({ kind: 'resolve', snapshot: { positions: [suiPosition], fetchedAt: START_MS } });

    await expect(firstCall).resolves.toEqual({ positions: [suiPosition], fetchedAt: START_MS });
    await expect(lateCall).resolves.toEqual({ positions: [suiPosition], fetchedAt: START_MS });
    expect(getCallCount()).toBe(1);
  });
});
