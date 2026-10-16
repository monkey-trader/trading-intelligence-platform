import type { OpenPositionsSnapshot } from '@/domain/position/entities/OpenPositionsSnapshot';
import { OpenPositionsUnavailableError } from '@/domain/position/errors/OpenPositionsUnavailableError';
import type { OpenPositionSource } from '@/domain/position/interfaces/OpenPositionSource';

/** Ersatz-Source, wenn BX_KEY/BX_SECRET fehlen: der Server startet, der Endpunkt meldet not-configured. */
export class UnconfiguredOpenPositionSource implements OpenPositionSource {
  async fetchOpenPositions(): Promise<OpenPositionsSnapshot> {
    throw new OpenPositionsUnavailableError('not-configured');
  }
}
