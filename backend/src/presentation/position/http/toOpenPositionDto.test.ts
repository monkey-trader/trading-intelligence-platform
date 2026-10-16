import { describe, expect, it } from 'vitest';
import { toOpenPositionDto } from '@/presentation/position/http/toOpenPositionDto';
import type { OpenPosition } from '@/domain/position/entities/OpenPosition';

/** Feldliste des OpenAPI-Schemas `OpenPosition` (docs/openapi.yaml). */
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
  'openTime',
  'updateTime',
] as const;

const samplePosition: OpenPosition = {
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
  openTime: 1791700000000,
  updateTime: 1791710000000,
};

describe('toOpenPositionDto', () => {
  it('bildet alle Felder unverändert ab', () => {
    expect(toOpenPositionDto(samplePosition)).toEqual({ ...samplePosition });
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
