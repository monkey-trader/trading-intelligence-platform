import { act, renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import {
  COLUMN_ORDER_STORAGE_KEY,
  DEFAULT_COLUMN_ORDER,
} from '@/presentation/position/columnOrder';
import { useColumnOrder } from '@/presentation/position/hooks/useColumnOrder';

function fakeStorage(initial?: string) {
  const data = new Map<string, string>();
  if (initial !== undefined) data.set(COLUMN_ORDER_STORAGE_KEY, initial);
  return {
    data,
    getItem: vi.fn((key: string) => data.get(key) ?? null),
    setItem: vi.fn((key: string, value: string) => void data.set(key, value)),
    removeItem: vi.fn((key: string) => void data.delete(key)),
  };
}

describe('useColumnOrder', () => {
  it('nutzt den v2-Storage-Key', () => {
    expect(COLUMN_ORDER_STORAGE_KEY).toBe('tip_open_positions_column_order_v2');
  });
  it('liefert ohne Eintrag die Standardreihenfolge', () => {
    const { result } = renderHook(() => useColumnOrder(fakeStorage()));
    expect(result.current.order).toEqual([...DEFAULT_COLUMN_ORDER]);
    expect(result.current.isDefault).toBe(true);
  });
  it('liest gespeicherte Reihenfolge', () => {
    const stored = [
      'symbol',
      'entrySl',
      ...DEFAULT_COLUMN_ORDER.filter((k) => k !== 'symbol' && k !== 'entrySl'),
    ];
    const { result } = renderHook(() => useColumnOrder(fakeStorage(JSON.stringify(stored))));
    expect(result.current.order.slice(0, 2)).toEqual(['symbol', 'entrySl']);
    expect(result.current.isDefault).toBe(false);
  });
  it('kaputtes JSON -> Standard, kein Absturz', () => {
    const { result } = renderHook(() => useColumnOrder(fakeStorage('{kaputt')));
    expect(result.current.order).toEqual([...DEFAULT_COLUMN_ORDER]);
  });
  it('wirft getItem -> Standard', () => {
    const storage = fakeStorage();
    storage.getItem.mockImplementation(() => {
      throw new Error('gesperrt');
    });
    const { result } = renderHook(() => useColumnOrder(storage));
    expect(result.current.order).toEqual([...DEFAULT_COLUMN_ORDER]);
  });
  it('move speichert als JSON-Array', () => {
    const storage = fakeStorage();
    const { result } = renderHook(() => useColumnOrder(storage));
    act(() => result.current.move('unrealizedProfit', 'valueMargin'));
    expect(result.current.order.slice(0, 3)).toEqual(['symbol', 'unrealizedProfit', 'valueMargin']);
    const saved: unknown = JSON.parse(storage.data.get(COLUMN_ORDER_STORAGE_KEY) ?? 'null');
    expect(saved).toEqual(result.current.order);
  });
  it('move bei wirfendem setItem: State ändert sich trotzdem, kein Absturz', () => {
    const storage = fakeStorage();
    storage.setItem.mockImplementation(() => {
      throw new Error('voll');
    });
    const { result } = renderHook(() => useColumnOrder(storage));
    act(() => result.current.move('unrealizedProfit', 'valueMargin'));
    expect(result.current.order[1]).toBe('unrealizedProfit');
  });
  it('reset stellt Standard her und löscht den Eintrag', () => {
    const storage = fakeStorage();
    const { result } = renderHook(() => useColumnOrder(storage));
    act(() => result.current.move('unrealizedProfit', 'valueMargin'));
    act(() => result.current.reset());
    expect(result.current.order).toEqual([...DEFAULT_COLUMN_ORDER]);
    expect(result.current.isDefault).toBe(true);
    expect(storage.removeItem).toHaveBeenCalledWith(COLUMN_ORDER_STORAGE_KEY);
    expect(storage.data.has(COLUMN_ORDER_STORAGE_KEY)).toBe(false);
  });
  it('reset bei wirfendem removeItem: kein Absturz', () => {
    const storage = fakeStorage();
    storage.removeItem.mockImplementation(() => {
      throw new Error('gesperrt');
    });
    const { result } = renderHook(() => useColumnOrder(storage));
    act(() => result.current.move('unrealizedProfit', 'valueMargin'));
    act(() => result.current.reset());
    expect(result.current.isDefault).toBe(true);
  });
});
