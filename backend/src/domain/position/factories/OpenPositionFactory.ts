import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import { UpdateBeforeOpenError } from '@/domain/position/errors/UpdateBeforeOpenError';
import { DecimalAmount } from '@/domain/position/valueObjects/DecimalAmount';
import { EpochMillis } from '@/domain/position/valueObjects/EpochMillis';
import { Leverage } from '@/domain/position/valueObjects/Leverage';
import { PositionId } from '@/domain/position/valueObjects/PositionId';
import { PositionSide } from '@/domain/position/valueObjects/PositionSide';
import { TradingSymbol } from '@/domain/position/valueObjects/TradingSymbol';

/** Primitive Eingabe für die Factory. Die Seite ist bewusst `string`, geprüft wird über das VO. */
export type OpenPositionInput = Readonly<{
  positionId: string;
  symbol: string;
  side: string;
  isolated: boolean;
  positionAmount: string;
  avgOpenPrice: string;
  markPrice: string;
  leverage: number;
  initialMargin: string;
  margin: string;
  positionValue: string;
  unrealizedProfit: string;
  realisedProfit: string;
  pnlRatio: string;
  liquidationPrice: string;
  openTime: number;
  updateTime: number;
}>;

export const OpenPositionFactory = {
  /** Validiert jedes Feld über sein Value Object und liefert eine offene Position mit primitiven Feldern. */
  create(input: OpenPositionInput): OpenPosition {
    const openTime = EpochMillis.create(input.openTime).value;
    const updateTime = EpochMillis.create(input.updateTime).value;
    if (updateTime < openTime) {
      throw new UpdateBeforeOpenError(
        `Änderungszeit ${updateTime} liegt vor Eröffnungszeit ${openTime} (Position ${input.positionId}).`,
      );
    }

    return {
      positionId: PositionId.create(input.positionId).value,
      symbol: TradingSymbol.create(input.symbol).value,
      side: PositionSide.create(input.side).value,
      isolated: input.isolated,
      positionAmount: DecimalAmount.create(input.positionAmount).value,
      avgOpenPrice: DecimalAmount.create(input.avgOpenPrice).value,
      markPrice: DecimalAmount.create(input.markPrice).value,
      leverage: Leverage.create(input.leverage).value,
      initialMargin: DecimalAmount.create(input.initialMargin).value,
      margin: DecimalAmount.create(input.margin).value,
      positionValue: DecimalAmount.create(input.positionValue).value,
      unrealizedProfit: DecimalAmount.create(input.unrealizedProfit).value,
      realisedProfit: DecimalAmount.create(input.realisedProfit).value,
      pnlRatio: DecimalAmount.create(input.pnlRatio).value,
      liquidationPrice: DecimalAmount.create(input.liquidationPrice).value,
      openTime,
      updateTime,
    };
  },
};
