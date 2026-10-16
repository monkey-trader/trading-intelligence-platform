import { describe, expect, it } from 'vitest';
import { mapOpenPositionsResponse } from '@/infrastructure/position/bingx/openPositionsMapper';
import { UnexpectedBingXResponseError } from '@/infrastructure/bingx/errors';
import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';
import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import openPositionsFixture from '@/infrastructure/position/bingx/fixtures/openPositions.json';
import openPositionsEmptyFixture from '@/infrastructure/position/bingx/fixtures/openPositions.empty.json';

function captureError(action: () => unknown): unknown {
  try {
    action();
  } catch (error) {
    return error;
  }
  throw new Error('Es wurde ein Fehler erwartet, aber keiner geworfen.');
}

function requireDefined<T>(value: T | undefined): T {
  if (value === undefined) throw new Error('Fixture unvollständig');
  return value;
}

function isMappingError(caught: unknown): boolean {
  return caught instanceof PositionDomainError || caught instanceof UnexpectedBingXResponseError;
}

const firstRawPosition = requireDefined(openPositionsFixture.data[0]);

function withLiquidationPrice(liquidationPrice: unknown): unknown[] {
  return [{ ...firstRawPosition, liquidationPrice }];
}

describe('mapOpenPositionsResponse', () => {
  it('mappt die Fixture auf vollständige offene Positionen', () => {
    const positions = mapOpenPositionsResponse(openPositionsFixture.data);

    const expected: OpenPosition[] = [
      {
        positionId: '9100000000000000001',
        symbol: 'SUI-USDT',
        side: 'LONG',
        isolated: true,
        positionAmount: '500',
        avgOpenPrice: '0.9600',
        markPrice: '1.0100',
        leverage: 10,
        initialMargin: '48.0000',
        margin: '60.5000',
        positionValue: '505',
        unrealizedProfit: '25.0000',
        realisedProfit: '-0.2400',
        pnlRatio: '0.5208',
        liquidationPrice: '0.865',
        openTime: 1791500000000,
        updateTime: 1791600000000,
      },
      {
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
      },
    ];
    expect(positions).toEqual(expected);
  });

  it('übernimmt keine zusätzlichen BingX-Felder (currency, riskRate, availableAmt …)', () => {
    const positions = mapOpenPositionsResponse(openPositionsFixture.data);

    for (const position of positions) {
      expect(position).not.toHaveProperty('currency');
      expect(position).not.toHaveProperty('riskRate');
      expect(position).not.toHaveProperty('availableAmt');
      expect(position).not.toHaveProperty('maxMarginReduction');
      expect(position).not.toHaveProperty('minIncreaseMargin');
      expect(position).not.toHaveProperty('onlyOnePosition');
    }
  });

  it('liefert für die empty-Fixture eine leere Liste', () => {
    expect(mapOpenPositionsResponse(openPositionsEmptyFixture.data)).toEqual([]);
  });

  it('liefert für data = null eine leere Liste', () => {
    expect(mapOpenPositionsResponse(null)).toEqual([]);
  });

  it.each([
    ['undefined', undefined],
    ['ein leeres Objekt', {}],
    ['ein String', 'positions'],
    ['eine Zahl', 0],
  ])('wirft UnexpectedBingXResponseError für %s (nie still [])', (_description, data) => {
    expect(() => mapOpenPositionsResponse(data)).toThrow(UnexpectedBingXResponseError);
  });

  it('wirft UnexpectedBingXResponseError, wenn ein Eintrag kein Objekt ist', () => {
    expect(() => mapOpenPositionsResponse([firstRawPosition, null])).toThrow(
      UnexpectedBingXResponseError,
    );
  });

  it('übernimmt liquidationPrice als String unverändert', () => {
    const positions = mapOpenPositionsResponse(withLiquidationPrice('0.86500'));

    expect(positions[0]?.liquidationPrice).toBe('0.86500');
  });

  it('wandelt liquidationPrice als number 0 in "0" um', () => {
    const positions = mapOpenPositionsResponse(withLiquidationPrice(0));

    expect(positions[0]?.liquidationPrice).toBe('0');
  });

  it('wandelt liquidationPrice als number 4410.5 in "4410.5" um', () => {
    const positions = mapOpenPositionsResponse(withLiquidationPrice(4410.5));

    expect(positions[0]?.liquidationPrice).toBe('4410.5');
  });

  it.each([
    [1e-7, '0.0000001'],
    [5e-7, '0.0000005'],
    [1.5e-8, '0.000000015'],
    [1e21, '1000000000000000000000'],
  ])('wandelt liquidationPrice in Exponentenform %s verlustfrei in "%s" um', (raw, expected) => {
    const positions = mapOpenPositionsResponse(withLiquidationPrice(raw));

    expect(positions[0]?.liquidationPrice).toBe(expected);
  });

  it('nennt den Endpunkt in der Fehlermeldung eines fehlenden Felds', () => {
    const withoutMarkPrice = Object.fromEntries(
      Object.entries(firstRawPosition).filter(([fieldName]) => fieldName !== 'markPrice'),
    );

    expect(() => mapOpenPositionsResponse([withoutMarkPrice])).toThrow(
      /user\/positions.*markPrice/,
    );
  });

  it.each([
    ['NaN', Number.NaN],
    ['Infinity', Number.POSITIVE_INFINITY],
    ['-Infinity', Number.NEGATIVE_INFINITY],
    ['null', null],
    ['ein Boolean', true],
  ])('wirft einen Fehler für liquidationPrice als %s', (_description, liquidationPrice) => {
    const caught = captureError(() =>
      mapOpenPositionsResponse(withLiquidationPrice(liquidationPrice)),
    );

    expect(isMappingError(caught)).toBe(true);
  });

  it('wirft einen Fehler, wenn liquidationPrice fehlt', () => {
    const withoutLiquidationPrice = Object.fromEntries(
      Object.entries(firstRawPosition).filter(([fieldName]) => fieldName !== 'liquidationPrice'),
    );

    const caught = captureError(() => mapOpenPositionsResponse([withoutLiquidationPrice]));

    expect(isMappingError(caught)).toBe(true);
  });

  it('lehnt positionId als number ab (Präzisionsverlust)', () => {
    const data = [{ ...firstRawPosition, positionId: 9100000000000000000 }];

    const caught = captureError(() => mapOpenPositionsResponse(data));

    expect(isMappingError(caught)).toBe(true);
  });

  it('wirft einen Fehler, wenn ein Pflichtfeld fehlt', () => {
    const withoutMarkPrice = Object.fromEntries(
      Object.entries(firstRawPosition).filter(([fieldName]) => fieldName !== 'markPrice'),
    );

    const caught = captureError(() => mapOpenPositionsResponse([withoutMarkPrice]));

    expect(isMappingError(caught)).toBe(true);
  });

  it('reicht Domain-Fehler der Factory durch (z.B. ungültige Seite)', () => {
    const data = [{ ...firstRawPosition, positionSide: 'BOTH' }];

    const caught = captureError(() => mapOpenPositionsResponse(data));

    expect(caught).toBeInstanceOf(PositionDomainError);
  });
});
