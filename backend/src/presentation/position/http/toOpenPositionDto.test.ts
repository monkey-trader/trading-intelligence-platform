import { describe, expect, it } from 'vitest';
import {
  toOpenPositionDto,
  toOpenPositionsSnapshotDto,
} from '@/presentation/position/http/toOpenPositionDto';
import type { OpenPositionsOverview } from '@/application/position/GetOpenPositionsUseCase';

type OverviewPosition = OpenPositionsOverview['positions'][number];

/** Feldliste des OpenAPI-Schemas `OpenPosition` (docs/openapi.yaml 0.3.0). */
const OPEN_POSITION_SCHEMA_FIELDS = [
  'positionId',
  'symbol',
  'side',
  'isolated',
  'positionAmount',
  'avgOpenPrice',
  'markPrice',
  'leverage',
  'initialMargin',
  'margin',
  'positionValue',
  'unrealizedProfit',
  'realisedProfit',
  'pnlRatio',
  'liquidationPrice',
  'liquidationDistanceRatio',
  'openTime',
  'updateTime',
] as const;

/** Feldliste des OpenAPI-Schemas `OpenPositionsSnapshot`. */
const SNAPSHOT_SCHEMA_FIELDS = ['positions', 'totalUnrealizedProfit', 'fetchedAt'] as const;

const samplePosition: OverviewPosition = {
  positionId: '9100000000000000002',
  symbol: 'NCCOGOLD2USD-USDT',
  side: 'SHORT',
  isolated: true,
  positionAmount: '0.25',
  avgOpenPrice: '4150.00',
  markPrice: '4175.00',
  leverage: 15,
  initialMargin: '69.1667',
  margin: '65.0000',
  positionValue: '1043.75',
  unrealizedProfit: '-6.2500',
  realisedProfit: '-0.5187',
  pnlRatio: '-0.0904',
  liquidationPrice: '4410.5',
  liquidationDistanceRatio: '0.05640719',
  openTime: 1791700000000,
  updateTime: 1791710000000,
};

const crossPosition: OverviewPosition = {
  ...samplePosition,
  positionId: '9100000000000000003',
  isolated: false,
  liquidationPrice: '0',
  liquidationDistanceRatio: null,
};

describe('toOpenPositionDto', () => {
  it('bildet alle Felder unverändert ab', () => {
    expect(toOpenPositionDto(samplePosition)).toEqual({ ...samplePosition });
  });

  it('enthält liquidationDistanceRatio als String', () => {
    expect(toOpenPositionDto(samplePosition).liquidationDistanceRatio).toBe('0.05640719');
  });

  it('enthält liquidationDistanceRatio als null ohne Liquidationspreis', () => {
    const dto = toOpenPositionDto(crossPosition);

    expect(dto).toHaveProperty('liquidationDistanceRatio');
    expect(dto.liquidationDistanceRatio).toBeNull();
  });

  it('enthält genau die Felder des OpenAPI-Schemas', () => {
    const dto = toOpenPositionDto(samplePosition);

    expect(Object.keys(dto).sort()).toEqual([...OPEN_POSITION_SCHEMA_FIELDS].sort());
  });

  it('übernimmt keine zusätzlichen Felder einer Entity', () => {
    const positionWithExtras = {
      ...samplePosition,
      riskRate: '0.0120',
      rawPayload: '{"geheim":"xyz"}',
    };

    const dto = toOpenPositionDto(positionWithExtras);

    expect(dto).not.toHaveProperty('riskRate');
    expect(dto).not.toHaveProperty('rawPayload');
    expect(Object.keys(dto).sort()).toEqual([...OPEN_POSITION_SCHEMA_FIELDS].sort());
  });

  it('liefert ein neues Objekt statt der Entity selbst', () => {
    expect(toOpenPositionDto(samplePosition)).not.toBe(samplePosition);
  });
});

describe('toOpenPositionsSnapshotDto', () => {
  const overview: OpenPositionsOverview = {
    positions: [samplePosition, crossPosition],
    totalUnrealizedProfit: '-12.5',
    fetchedAt: 1791800000000,
  };

  it('bildet die Hülle mit Positionen, Summe und fetchedAt ab', () => {
    expect(toOpenPositionsSnapshotDto(overview)).toEqual({
      positions: [{ ...samplePosition }, { ...crossPosition }],
      totalUnrealizedProfit: '-12.5',
      fetchedAt: 1791800000000,
    });
  });

  it('enthält genau die Felder des OpenAPI-Schemas OpenPositionsSnapshot', () => {
    const dto = toOpenPositionsSnapshotDto(overview);

    expect(Object.keys(dto).sort()).toEqual([...SNAPSHOT_SCHEMA_FIELDS].sort());
  });

  it('übernimmt keine zusätzlichen Felder der Hülle oder der Positionen', () => {
    const overviewWithExtras = {
      ...overview,
      positions: [{ ...samplePosition, riskRate: '0.0120' }],
      debugInfo: 'intern',
    };

    const dto = toOpenPositionsSnapshotDto(overviewWithExtras);

    expect(dto).not.toHaveProperty('debugInfo');
    expect(Object.keys(dto.positions[0] ?? {}).sort()).toEqual(
      [...OPEN_POSITION_SCHEMA_FIELDS].sort(),
    );
  });

  it('behält die Reihenfolge der Positionen', () => {
    const dto = toOpenPositionsSnapshotDto(overview);

    expect(dto.positions.map((position) => position.positionId)).toEqual([
      '9100000000000000002',
      '9100000000000000003',
    ]);
  });

  it('bildet eine leere Liste mit Summe "0" ab', () => {
    expect(
      toOpenPositionsSnapshotDto({ positions: [], totalUnrealizedProfit: '0', fetchedAt: 1 }),
    ).toEqual({ positions: [], totalUnrealizedProfit: '0', fetchedAt: 1 });
  });
});
