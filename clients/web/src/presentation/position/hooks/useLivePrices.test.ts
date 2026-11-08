import { act, renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type {
  PositionPrice,
  PriceSnapshot,
  PriceStreamGateway,
  PriceStreamHandlers,
} from '@/domain/price/PriceStream';
import { useLivePrices } from '@/presentation/position/hooks/useLivePrices';

const price = (positionId: string, value: string): PositionPrice => ({
  positionId,
  symbol: 'SUI-USDT',
  price: value,
  change24hPercent: '1.3',
  distanceToBreakEvenPercent: '2',
  distanceToTp1Percent: '5',
});
const snapshot = (value: string, stale = false): PriceSnapshot => ({
  prices: [price('p1', value)],
  fetchedAt: 1,
  stale,
});

function fakeGateway() {
  let handlers: PriceStreamHandlers | undefined;
  const unsubscribe = vi.fn();
  const subscribe = vi.fn((given: PriceStreamHandlers) => {
    handlers = given;
    return unsubscribe;
  });
  const gateway: PriceStreamGateway = { subscribe };
  const need = () => {
    if (!handlers) throw new Error('nicht abonniert');
    return handlers;
  };
  return {
    gateway,
    subscribe,
    unsubscribe,
    push: (value: PriceSnapshot) => act(() => need().onSnapshot(value)),
    status: (value: 'live' | 'ausgefallen') => act(() => need().onStatus(value)),
  };
}

describe('useLivePrices', () => {
  it('abonniert beim Mount genau einmal und beendet beim Unmount', () => {
    const fake = fakeGateway();
    const { unmount, rerender } = renderHook(() => useLivePrices(fake.gateway));
    rerender();
    expect(fake.subscribe).toHaveBeenCalledTimes(1);
    unmount();
    expect(fake.unsubscribe).toHaveBeenCalledTimes(1);
  });

  it('liefert ohne Daten keinen Kurs und ist nicht veraltet', () => {
    const fake = fakeGateway();
    const { result } = renderHook(() => useLivePrices(fake.gateway));
    expect(result.current.prices['p1']).toBeUndefined();
    expect(result.current.stale).toBe(false);
  });

  it('liefert den Kurs je positionId', async () => {
    const fake = fakeGateway();
    const { result } = renderHook(() => useLivePrices(fake.gateway));
    await fake.push(snapshot('1.5'));
    expect(result.current.prices['p1']?.price).toBe('1.5');
  });

  it('markiert veraltet, wenn PriceSnapshot.stale gesetzt ist', async () => {
    const fake = fakeGateway();
    const { result } = renderHook(() => useLivePrices(fake.gateway));
    await fake.push(snapshot('1.5', true));
    expect(result.current.stale).toBe(true);
  });

  it('behält bei Ausfall den letzten Kurs und markiert veraltet', async () => {
    const fake = fakeGateway();
    const { result } = renderHook(() => useLivePrices(fake.gateway));
    await fake.push(snapshot('1.5'));
    await fake.status('ausgefallen');
    expect(result.current.prices['p1']?.price).toBe('1.5');
    expect(result.current.stale).toBe(true);
  });

  it('wird nach Erholung (live plus frischer Snapshot) wieder aktuell', async () => {
    const fake = fakeGateway();
    const { result } = renderHook(() => useLivePrices(fake.gateway));
    await fake.status('ausgefallen');
    await fake.status('live');
    await fake.push(snapshot('2'));
    expect(result.current.stale).toBe(false);
  });

  it('Ticks ändern den Ansagetext nicht, nur der Wechsel Daten/veraltet', async () => {
    const fake = fakeGateway();
    const { result } = renderHook(() => useLivePrices(fake.gateway));
    await fake.push(snapshot('1.5'));
    const live = result.current.announcement;
    await fake.push(snapshot('1.6'));
    await fake.push(snapshot('1.7'));
    expect(result.current.announcement).toBe(live);
    await fake.status('ausgefallen');
    expect(result.current.announcement).not.toBe(live);
    expect(result.current.announcement).toMatch(/veraltet/i);
  });
});
