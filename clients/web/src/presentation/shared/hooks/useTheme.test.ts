import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import { THEME_STORAGE_KEY, useTheme } from '@/presentation/shared/hooks/useTheme';

describe('useTheme', () => {
  beforeEach(() => {
    window.localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  it('startet ohne Wahl mit "system" und setzt kein data-theme', () => {
    const { result } = renderHook(() => useTheme());

    expect(result.current.theme).toBe('system');
    expect(document.documentElement.hasAttribute('data-theme')).toBe(false);
  });

  it('setzt data-theme und speichert die Wahl', () => {
    const { result } = renderHook(() => useTheme());

    act(() => result.current.setTheme('dark'));

    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
    expect(window.localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark');
  });

  it('liest die gespeicherte Wahl beim Start', () => {
    window.localStorage.setItem(THEME_STORAGE_KEY, 'light');

    const { result } = renderHook(() => useTheme());

    expect(result.current.theme).toBe('light');
    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
  });

  it('ignoriert ungültige gespeicherte Werte', () => {
    window.localStorage.setItem(THEME_STORAGE_KEY, 'neon');

    const { result } = renderHook(() => useTheme());

    expect(result.current.theme).toBe('system');
  });

  it('"system" entfernt data-theme wieder', () => {
    const { result } = renderHook(() => useTheme());
    act(() => result.current.setTheme('dark'));

    act(() => result.current.setTheme('system'));

    expect(document.documentElement.hasAttribute('data-theme')).toBe(false);
  });
});
