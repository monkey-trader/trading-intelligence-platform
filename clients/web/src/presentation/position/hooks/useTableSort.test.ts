import { act, renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { useTableSort } from '@/presentation/position/hooks/useTableSort';

describe('useTableSort', () => {
  it('startet ohne Sortierung (Reihenfolge der API)', () => {
    const { result } = renderHook(() => useTableSort());
    expect(result.current.sort).toBeNull();
  });

  it('cycle läuft aufsteigend, absteigend, zurück zur API-Reihenfolge', () => {
    const { result } = renderHook(() => useTableSort());
    act(() => result.current.cycle('margin'));
    expect(result.current.sort).toEqual({ key: 'margin', direction: 'asc' });
    act(() => result.current.cycle('margin'));
    expect(result.current.sort).toEqual({ key: 'margin', direction: 'desc' });
    act(() => result.current.cycle('margin'));
    expect(result.current.sort).toBeNull();
  });

  it('select setzt Schlüssel aufsteigend; null hebt auf', () => {
    const { result } = renderHook(() => useTableSort());
    act(() => result.current.select('tp1'));
    expect(result.current.sort).toEqual({ key: 'tp1', direction: 'asc' });
    act(() => result.current.select(null));
    expect(result.current.sort).toBeNull();
  });

  it('toggleDirection wechselt die Richtung, ohne Sortierung wirkungslos', () => {
    const { result } = renderHook(() => useTableSort());
    act(() => result.current.toggleDirection());
    expect(result.current.sort).toBeNull();
    act(() => result.current.select('entry'));
    act(() => result.current.toggleDirection());
    expect(result.current.sort).toEqual({ key: 'entry', direction: 'desc' });
    act(() => result.current.toggleDirection());
    expect(result.current.sort).toEqual({ key: 'entry', direction: 'asc' });
  });

  it('Zustand bleibt über erneutes Rendern erhalten (Fehler/veraltet)', () => {
    const { result, rerender } = renderHook(() => useTableSort());
    act(() => result.current.cycle('symbol'));
    rerender();
    expect(result.current.sort).toEqual({ key: 'symbol', direction: 'asc' });
  });
});
