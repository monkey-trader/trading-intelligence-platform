import type { Position } from '@/domain/position/entities/Position';
import { CloseBeforeOpenError } from '@/domain/position/errors/CloseBeforeOpenError';
import { DecimalAmount } from '@/domain/position/valueObjects/DecimalAmount';
import { EpochMillis } from '@/domain/position/valueObjects/EpochMillis';
import { Leverage } from '@/domain/position/valueObjects/Leverage';
import { PositionId } from '@/domain/position/valueObjects/PositionId';
import { PositionSide } from '@/domain/position/valueObjects/PositionSide';
import { TradingSymbol } from '@/domain/position/valueObjects/TradingSymbol';

/** Primitive Eingabe für die Factory. Die Seite ist bewusst `string`, geprüft wird über das VO. */
export type PositionInput = Readonly<{
  positionId: string;
  symbol: string;
  side: string;
  isolated: boolean;
  openTime: number;
  closeTime: number;
  avgOpenPrice: string;
  avgClosePrice: string;
  realisedProfit: string;
  netProfit: string;
  positionAmount: string;
  closedAmount: string;
  commission: string;
  totalFunding: string;
  leverage: number;
  fullyClosed: boolean;
}>;

export const PositionFactory = {
  /** Validiert jedes Feld über sein Value Object und liefert eine Position mit primitiven Feldern. */
  create(input: PositionInput): Position {
    const openTime = EpochMillis.create(input.openTime).value;
    const closeTime = EpochMillis.create(input.closeTime).value;
    if (closeTime < openTime) {
      throw new CloseBeforeOpenError(
        `Schließzeit ${closeTime} liegt vor Eröffnungszeit ${openTime} (Position ${input.positionId}).`,
      );
    }

    return {
      positionId: PositionId.create(input.positionId).value,
      symbol: TradingSymbol.create(input.symbol).value,
      side: PositionSide.create(input.side).value,
      isolated: input.isolated,
      openTime,
      closeTime,
      avgOpenPrice: DecimalAmount.create(input.avgOpenPrice).value,
      avgClosePrice: DecimalAmount.create(input.avgClosePrice).value,
      realisedProfit: DecimalAmount.create(input.realisedProfit).value,
      netProfit: DecimalAmount.create(input.netProfit).value,
      positionAmount: DecimalAmount.create(input.positionAmount).value,
      closedAmount: DecimalAmount.create(input.closedAmount).value,
      commission: DecimalAmount.create(input.commission).value,
      totalFunding: DecimalAmount.create(input.totalFunding).value,
      leverage: Leverage.create(input.leverage).value,
      fullyClosed: input.fullyClosed,
    };
  },
};
