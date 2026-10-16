import type { OpenPosition } from '@/domain/position/entities/OpenPosition';

/** Antwortform von GET /positions/open, entspricht dem Schema `OpenPosition` in docs/openapi.yaml. */
export type OpenPositionDto = {
  positionId: string;
  symbol: string;
  side: 'LONG' | 'SHORT';
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
};

/** Explizite Abbildung, damit zusätzliche Felder einer Entity nie in die Antwort gelangen. */
export function toOpenPositionDto(position: OpenPosition): OpenPositionDto {
  return {
    positionId: position.positionId,
    symbol: position.symbol,
    side: position.side,
    isolated: position.isolated,
    positionAmount: position.positionAmount,
    avgOpenPrice: position.avgOpenPrice,
    markPrice: position.markPrice,
    leverage: position.leverage,
    initialMargin: position.initialMargin,
    margin: position.margin,
    positionValue: position.positionValue,
    unrealizedProfit: position.unrealizedProfit,
    realisedProfit: position.realisedProfit,
    pnlRatio: position.pnlRatio,
    liquidationPrice: position.liquidationPrice,
    openTime: position.openTime,
    updateTime: position.updateTime,
  };
}
