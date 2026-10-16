import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import type { OpenPositionSource } from '@/domain/position/interfaces/OpenPositionSource';
import { liquidationDistanceRatio } from '@/domain/position/services/liquidationDistanceRatio';
import { summarizeOpenPositions } from '@/domain/position/services/summarizeOpenPositions';

/** Offene Position, ergänzt um den relativen Abstand zum Liquidationspreis (null ohne Liquidation). */
export type OpenPositionWithDistance = OpenPosition & { liquidationDistanceRatio: string | null };

/** Ergebnis des Use Cases: sortierte Positionen, exakte PnL-Summe und Abfragezeitpunkt. */
export type OpenPositionsOverview = {
  positions: OpenPositionWithDistance[];
  totalUnrealizedProfit: string;
  fetchedAt: number;
};

/**
 * Liefert die offenen Positionen, neueste zuerst. Bei gleicher Eröffnungszeit nach Symbol,
 * dann LONG vor SHORT (Hedge-Modus), dann nach positionId, damit die Reihenfolge immer fest ist.
 * Abstand zur Liquidation und Summe kommen aus den Domain-Services.
 */
export class GetOpenPositionsUseCase {
  constructor(private readonly openPositionSource: OpenPositionSource) {}

  async execute(): Promise<OpenPositionsOverview> {
    const snapshot = await this.openPositionSource.fetchOpenPositions();
    const positions = [...snapshot.positions]
      .sort(compareNewestFirstThenSymbol)
      .map(withLiquidationDistance);
    const { totalUnrealizedProfit } = summarizeOpenPositions(positions);
    return { positions, totalUnrealizedProfit, fetchedAt: snapshot.fetchedAt };
  }
}

function withLiquidationDistance(position: OpenPosition): OpenPositionWithDistance {
  return {
    ...position,
    liquidationDistanceRatio: liquidationDistanceRatio(
      position.side,
      position.markPrice,
      position.liquidationPrice,
    ),
  };
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
