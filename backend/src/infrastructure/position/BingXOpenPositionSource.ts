import type { OpenPositionsSnapshot } from '@/domain/position/entities/OpenPositionsSnapshot';
import {
  OpenPositionsUnavailableError,
  type OpenPositionsUnavailableReason,
} from '@/domain/position/errors/OpenPositionsUnavailableError';
import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';
import type { OpenPositionSource } from '@/domain/position/interfaces/OpenPositionSource';
import type { Clock } from '@/domain/shared/interfaces/Clock';
import type { BingXGetClient } from '@/infrastructure/bingx/BingXGetClient';
import {
  BingXApiError,
  BingXHttpError,
  BingXNetworkError,
  BingXRateLimitError,
  UnexpectedBingXResponseError,
} from '@/infrastructure/bingx/errors';
import { mapOpenPositionsResponse } from '@/infrastructure/position/bingx/openPositionsMapper';

const OPEN_POSITIONS_PATH = '/openApi/swap/v2/user/positions';

/**
 * Fragt die aktuell offenen Positionen live bei BingX ab und speichert nichts.
 * BingX-Fehler werden in OpenPositionsUnavailableError übersetzt, unbekannte Fehler gehen durch.
 * `fetchedAt` stammt aus der Clock, gelesen nach der erfolgreichen Antwort.
 */
export class BingXOpenPositionSource implements OpenPositionSource {
  constructor(
    private readonly client: BingXGetClient,
    private readonly clock: Clock,
  ) {}

  async fetchOpenPositions(): Promise<OpenPositionsSnapshot> {
    try {
      const data = await this.client.get(OPEN_POSITIONS_PATH, {});
      const positions = mapOpenPositionsResponse(data);
      return { positions, fetchedAt: this.clock.now() };
    } catch (error: unknown) {
      const reason = toUnavailableReason(error);
      if (reason === undefined) throw error;
      throw new OpenPositionsUnavailableError(reason, error);
    }
  }
}

function toUnavailableReason(error: unknown): OpenPositionsUnavailableReason | undefined {
  if (error instanceof BingXRateLimitError) return 'rate-limited';
  if (error instanceof BingXNetworkError) return 'unreachable';
  if (error instanceof BingXApiError || error instanceof BingXHttpError) return 'upstream-error';
  if (error instanceof UnexpectedBingXResponseError || error instanceof PositionDomainError) {
    return 'invalid-response';
  }
  return undefined;
}
