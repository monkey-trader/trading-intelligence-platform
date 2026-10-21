import { useCallback, useEffect, useState } from 'react';

export type ThemeChoice = 'system' | 'light' | 'dark';

export const THEME_STORAGE_KEY = 'tip_theme_v1';

const THEME_CHOICES: readonly ThemeChoice[] = ['system', 'light', 'dark'];

function isThemeChoice(value: unknown): value is ThemeChoice {
  return THEME_CHOICES.some((choice) => choice === value);
}

function readStoredTheme(): ThemeChoice {
  try {
    const stored = window.localStorage.getItem(THEME_STORAGE_KEY);
    return isThemeChoice(stored) ? stored : 'system';
  } catch {
    return 'system';
  }
}

function applyTheme(theme: ThemeChoice): void {
  if (theme === 'system') {
    document.documentElement.removeAttribute('data-theme');
  } else {
    document.documentElement.setAttribute('data-theme', theme);
  }
}

/** Design-Wahl (System, Hell, Dunkel): setzt data-theme am <html> und merkt sich die Wahl im Browser. */
export function useTheme(): { theme: ThemeChoice; setTheme: (theme: ThemeChoice) => void } {
  const [theme, setThemeState] = useState<ThemeChoice>(readStoredTheme);

  useEffect(() => {
    applyTheme(theme);
  }, [theme]);

  const setTheme = useCallback((next: ThemeChoice) => {
    try {
      window.localStorage.setItem(THEME_STORAGE_KEY, next);
    } catch {
      // Speicher nicht verfügbar: Wahl gilt nur für diese Sitzung.
    }
    setThemeState(next);
  }, []);

  return { theme, setTheme };
}
