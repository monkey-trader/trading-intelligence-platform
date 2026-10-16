import { describe, expect, it } from 'vitest';
import { mapPositionHistoryResponse } from '@/infrastructure/position/bingx/positionHistoryMapper';
import { UnexpectedBingXResponseError } from '@/infrastructure/bingx/errors';
import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';
import type { Position } from '@/domain/position/entities/Position';
import page0Fixture from '@/infrastructure/position/bingx/fixtures/positionHistory.page0.json';
import shortFixture from '@/infrastructure/position/bingx/fixtures/positionHistory.short.json';
import emptyFixture from '@/infrastructure/position/bingx/fixtures/positionHistory.empty.json';

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

const firstRawPosition = requireDefined(page0Fixture.data.positionHistory[0]);

describe('mapPositionHistoryResponse', () => {
  it('mappt die page0-Fixture auf vollständige Positionen', () => {
    const records = mapPositionHistoryResponse(page0Fixture.data);

    expect(records).toHaveLength(3);
    const expectedFirst: Position = {
      positionId: '9000000000000000000',
      symbol: 'ONDO-USDT',
      side: 'LONG',
      isolated: true,
      openTime: 1780000000000,
      closeTime: 1780018017000,
      avgOpenPrice: '0.4120',
      avgClosePrice: '0.4188',
      realisedProfit: '1.7000',
      netProfit: '1.5123',
      positionAmount: '250.00',
      closedAmount: '250.00',
      commission: '-0.05121506352000001',
      totalFunding: '-0.1356',
      leverage: 7,
      fullyClosed: true,
    };
    expect(records[0]?.position).toEqual(expectedFirst);
    expect(records.map((record) => record.position.positionId)).toEqual([
      '9000000000000000000',
      '9000000000000000001',
      '9000000000000000002',
    ]);
  });

  it('mappt die short-Fixture (SHORT, langes Symbol, Verlust)', () => {
    const records = mapPositionHistoryResponse(shortFixture.data);

    expect(records).toHaveLength(1);
    expect(records[0]?.position).toEqual({
      positionId: '9000000000000000099',
      symbol: 'NCCOGOLD2USD-USDT',
      side: 'SHORT',
      isolated: true,
      openTime: 1780300000000,
      closeTime: 1780310017000,
      avgOpenPrice: '3412.55',
      avgClosePrice: '3398.10',
      realisedProfit: '-2.1675',
      netProfit: '-2.4019',
      positionAmount: '0.15',
      closedAmount: '0.15',
      commission: '-0.1024',
      totalFunding: '-0.132',
      leverage: 15,
      fullyClosed: true,
    } satisfies Position);
  });

  it('speichert als rawPayload das JSON des jeweiligen Einzelobjekts', () => {
    const records = mapPositionHistoryResponse(page0Fixture.data);

    expect(JSON.parse(records[0]?.rawPayload ?? 'null')).toEqual(firstRawPosition);
    expect(JSON.parse(records[2]?.rawPayload ?? 'null')).toEqual(
      page0Fixture.data.positionHistory[2],
    );
  });

  it('liefert für die empty-Fixture eine leere Liste', () => {
    expect(mapPositionHistoryResponse(emptyFixture.data)).toEqual([]);
  });

  it.each([
    ['ein leeres Objekt', {}],
    ['eine Liste unter anderem Schlüssel', { list: page0Fixture.data.positionHistory }],
    ['positionHistory als Objekt statt Array', { positionHistory: {} }],
    ['positionHistory als null', { positionHistory: null }],
    ['null', null],
    ['undefined', undefined],
    ['ein String', 'positionHistory'],
  ])('wirft UnexpectedBingXResponseError für %s (nie still [])', (_description, data) => {
    expect(() => mapPositionHistoryResponse(data)).toThrow(UnexpectedBingXResponseError);
  });

  it('wirft UnexpectedBingXResponseError, wenn ein Eintrag kein Objekt ist', () => {
    expect(() => mapPositionHistoryResponse({ positionHistory: [firstRawPosition, null] })).toThrow(
      UnexpectedBingXResponseError,
    );
  });

  it('lehnt positionId als number ab (Präzisionsverlust)', () => {
    const data = { positionHistory: [{ ...firstRawPosition, positionId: 9000000000000000000 }] };

    const caught = captureError(() => mapPositionHistoryResponse(data));

    expect(
      caught instanceof PositionDomainError || caught instanceof UnexpectedBingXResponseError,
    ).toBe(true);
  });

  it('nennt den Endpunkt in der Fehlermeldung eines falsch typisierten Felds', () => {
    const data = {
      positionHistory: [{ ...page0Fixture.data.positionHistory[0], leverage: 'sieben' }],
    };

    expect(() => mapPositionHistoryResponse(data)).toThrow(/positionHistory.*leverage/);
  });

  it('wirft einen Fehler, wenn ein Pflichtfeld fehlt', () => {
    const withoutNetProfit = Object.fromEntries(
      Object.entries(firstRawPosition).filter(([fieldName]) => fieldName !== 'netProfit'),
    );
    const data = { positionHistory: [withoutNetProfit] };

    const caught = captureError(() => mapPositionHistoryResponse(data));

    expect(
      caught instanceof PositionDomainError || caught instanceof UnexpectedBingXResponseError,
    ).toBe(true);
  });

  it('übernimmt Beträge als String, ohne sie in number umzuwandeln', () => {
    const records = mapPositionHistoryResponse(page0Fixture.data);

    for (const record of records) {
      expect(typeof record.position.commission).toBe('string');
      expect(typeof record.position.netProfit).toBe('string');
    }
  });
});
