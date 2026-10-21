import styles from '@/presentation/shared/components/AppHeader.module.css';
import { useTheme, type ThemeChoice } from '@/presentation/shared/hooks/useTheme';

const THEME_OPTIONS: ReadonlyArray<{ value: ThemeChoice; label: string }> = [
  { value: 'system', label: 'System' },
  { value: 'light', label: 'Hell' },
  { value: 'dark', label: 'Dunkel' },
];

/** Kopfzeile der Anwendung mit dem App-Titel und der Design-Wahl (System, Hell, Dunkel). */
export function AppHeader() {
  const { theme, setTheme } = useTheme();

  return (
    <header className={styles.header}>
      <div className={styles.titlebar} aria-hidden="true" />
      <span className={styles.title}>Trading Intelligence Platform</span>
      <label className={styles.themeControl}>
        <span className={styles.themeLabel}>Design</span>
        <select
          className={styles.themeSelect}
          value={theme}
          onChange={(event) => setTheme(event.target.value as ThemeChoice)}
        >
          {THEME_OPTIONS.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </label>
    </header>
  );
}
