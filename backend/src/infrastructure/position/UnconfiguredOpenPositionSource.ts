import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import { OpenPositionsUnavailableError } from '@/domain/position/errors/OpenPositionsUnavailableError';
import type { OpenPositionSource } from '@/domain/position/interfaces/OpenPositionSource';

/** Ersatz-Source, wenn BX_KEY/BX_SECRET fehlen: der Server startet, der Endpunkt meldet not-configured. */
export class UnconfiguredOpenPositionSource implements OpenPositionSource {
  async fetchOpenPositions(): Promise<OpenPosition[]> {
    throw new OpenPositionsUnavailableError('not-configured');
  }
}
