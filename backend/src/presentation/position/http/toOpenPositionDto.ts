import type {
  OpenPositionsOverview,
  OpenPositionWithDistance,
} from '@/application/position/GetOpenPositionsUseCase';

/** Eintrag der Antwort von GET /positions/open, entspricht `OpenPosition` in docs/openapi.yaml. */
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
  liquidationDistanceRatio: string | null;
  openTime: number;
  updateTime: number;
};

/** Antwort von GET /positions/open, entspricht `OpenPositionsSnapshot` in docs/openapi.yaml. */
export type OpenPositionsSnapshotDto = {
  positions: OpenPositionDto[];
  totalUnrealizedProfit: string;
  fetchedAt: number;
};

/** Explizite Abbildung, damit zusätzliche Felder einer Entity nie in die Antwort gelangen. */
export function toOpenPositionDto(position: OpenPositionWithDistance): OpenPositionDto {
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
    liquidationDistanceRatio: position.liquidationDistanceRatio,
    openTime: position.openTime,
    updateTime: position.updateTime,
  };
}

/** Explizite Abbildung der Hülle, zusätzliche Felder fallen weg. */
export function toOpenPositionsSnapshotDto(
  overview: OpenPositionsOverview,
): OpenPositionsSnapshotDto {
  return {
    positions: overview.positions.map(toOpenPositionDto),
    totalUnrealizedProfit: overview.totalUnrealizedProfit,
    fetchedAt: overview.fetchedAt,
  };
}
