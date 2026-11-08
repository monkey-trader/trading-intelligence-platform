import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { PriceStreamHandlers } from '@/domain/price/PriceStream';
import { createHttpPriceStreamGateway } from '@/infrastructure/api/httpPriceStreamGateway';
import { FakeEventSource } from '@/test/fakes/fakeEventSource';

const payload = {
  prices: [
    {
      positionId: '1234567890123456789',
      symbol: 'SUI-USDT',
      price: '0.0000123',
      change24hPercent: '-1.3',
      distanceToBreakEvenPercent: '2.5',
      distanceToTp1Percent: null,
    },
  ],
  fetchedAt: 1760000000000,
  stale: false,
};

function setup() {
  const handlers = { onSnapshot: vi.fn(), onStatus: vi.fn() } satisfies PriceStreamHandlers;
  const gateway = createHttpPriceStreamGateway((url) => new FakeEventSource(url), '/api');
  const unsubscribe = gateway.subscribe(handlers);
  return { handlers, unsubscribe, source: FakeEventSource.last() };
}

beforeEach(() => FakeEventSource.reset());

describe('httpPriceStreamGateway', () => {
  it('öffnet genau eine EventSource auf /api/prices/stream', () => {
    const { source } = setup();
    expect(FakeEventSource.instances).toHaveLength(1);
    expect(source.url).toBe('/api/prices/stream');
  });

  it('mappt das Event "prices" auf den PriceSnapshot, Dezimal-Strings unverändert', () => {
    const { handlers, source } = setup();
    source.emit('prices', JSON.stringify(payload));
    expect(handlers.onSnapshot).toHaveBeenCalledWith(payload);
  });

  it('meldet nach Öffnen und Snapshot den Status live', () => {
    const { handlers, source } = setup();
    source.open();
    expect(handlers.onStatus).toHaveBeenLastCalledWith('live');
  });

  it('ignoriert unbekannte Felder', () => {
    const { handlers, source } = setup();
    source.emit('prices', JSON.stringify({ ...payload, extra: 1 }));
    expect(handlers.onSnapshot).toHaveBeenCalledWith(payload);
  });

  it.each(['{kaputt', 'null', '{"prices":"x","fetchedAt":1,"stale":false}', ''])(
    'ignoriert ungültige Daten (%s) ohne Absturz und ohne Snapshot',
    (data) => {
      const { handlers, source } = setup();
      expect(() => source.emit('prices', data)).not.toThrow();
      expect(handlers.onSnapshot).not.toHaveBeenCalled();
    },
  );

  it('ignoriert andere Event-Typen', () => {
    const { handlers, source } = setup();
    source.emit('message', JSON.stringify(payload));
    expect(handlers.onSnapshot).not.toHaveBeenCalled();
  });

  it('meldet bei Fehler den Status ausgefallen, ohne selbst neu zu verbinden', () => {
    const { handlers, source } = setup();
    source.fail();
    expect(handlers.onStatus).toHaveBeenLastCalledWith('ausgefallen');
    expect(FakeEventSource.instances).toHaveLength(1);
    expect(source.closed).toBe(false);
  });

  it('unsubscribe schließt die EventSource', () => {
    const { unsubscribe, source } = setup();
    unsubscribe();
    expect(source.closed).toBe(true);
  });
});
