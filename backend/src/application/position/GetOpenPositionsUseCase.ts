import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import type { OpenPositionSource } from '@/domain/position/interfaces/OpenPositionSource';

/**
 * Liefert die offenen Positionen, neueste zuerst. Bei gleicher Eröffnungszeit nach Symbol,
 * dann LONG vor SHORT (Hedge-Modus), dann nach positionId, damit die Reihenfolge immer fest ist.
 */
export class GetOpenPositionsUseCase {
  constructor(private readonly openPositionSource: OpenPositionSource) {}

  async execute(): Promise<OpenPosition[]> {
    const positions = await this.openPositionSource.fetchOpenPositions();
    return [...positions].sort(compareNewestFirstThenSymbol);
  }
}

function compareNewestFirstThenSymbol(left: OpenPosition, right: OpenPosition): number {
  if (left.openTime !== right.openTime) return right.openTime - left.openTime;
  return (
    compareText(left.symbol, right.symbol) ||
    compareText(left.side, right.side) ||
    compareText(left.positionId, right.positionId)
  );
}

function compareText(left: string, right: string): number {
  if (left === right) return 0;
  return left < right ? -1 : 1;
}
