import type { OpenPositionsSnapshot } from '@/domain/position/entities/OpenPositionsSnapshot';
import { OpenPositionsUnavailableError } from '@/domain/position/errors/OpenPositionsUnavailableError';
import type { OpenPositionSource } from '@/domain/position/interfaces/OpenPositionSource';
import type { Clock } from '@/domain/shared/interfaces/Clock';

export type CachingOpenPositionSourceOptions = Readonly<{
  /** So lange wird ein erfolgreicher Snapshot unverändert ausgeliefert. */
  ttlMs: number;
  /** So lange wird ein rate-limited-Fehler erneut geworfen, ohne die Source zu fragen. */
  rateLimitHoldMs: number;
}>;

type CachedSnapshot = Readonly<{ snapshot: OpenPositionsSnapshot; expiresAtMs: number }>;
type HeldRateLimit = Readonly<{ error: OpenPositionsUnavailableError; expiresAtMs: number }>;

/**
 * Schützt BingX vor zu vielen Abfragen: cacht erfolgreiche Snapshots für `ttlMs` (das ursprüngliche
 * `fetchedAt` bleibt erhalten) und teilt gleichzeitige Aufrufe auf eine laufende Abfrage.
 * Fehler werden nicht gecacht, außer rate-limited: der wird `rateLimitHoldMs` lang erneut geworfen.
 */
export class CachingOpenPositionSource implements OpenPositionSource {
  private cachedSnapshot: CachedSnapshot | undefined;
  private heldRateLimit: HeldRateLimit | undefined;
  private inFlightFetch: Promise<OpenPositionsSnapshot> | undefined;

  constructor(
    private readonly source: OpenPositionSource,
    private readonly clock: Clock,
    private readonly options: CachingOpenPositionSourceOptions,
  ) {}

  fetchOpenPositions(): Promise<OpenPositionsSnapshot> {
    const nowMs = this.clock.now();
    if (this.cachedSnapshot !== undefined && nowMs < this.cachedSnapshot.expiresAtMs) {
      return Promise.resolve(this.cachedSnapshot.snapshot);
    }
    if (this.heldRateLimit !== undefined && nowMs < this.heldRateLimit.expiresAtMs) {
      return Promise.reject(this.heldRateLimit.error);
    }
    this.inFlightFetch ??= this.fetchAndRemember().finally(() => {
      this.inFlightFetch = undefined;
    });
    return this.inFlightFetch;
  }

  private async fetchAndRemember(): Promise<OpenPositionsSnapshot> {
    try {
      const snapshot = await this.source.fetchOpenPositions();
      this.cachedSnapshot = { snapshot, expiresAtMs: this.clock.now() + this.options.ttlMs };
      this.heldRateLimit = undefined;
      return snapshot;
    } catch (error: unknown) {
      if (error instanceof OpenPositionsUnavailableError && error.reason === 'rate-limited') {
        this.heldRateLimit = {
          error,
          expiresAtMs: this.clock.now() + this.options.rateLimitHoldMs,
        };
      }
      throw error;
    }
  }
}
