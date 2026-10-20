import styles from '@/presentation/shared/components/AppHeader.module.css';

/** Kopfzeile der Anwendung mit dem App-Titel. */
export function AppHeader() {
  return (
    <header className={styles.header}>
      <span className={styles.title}>Trading Intelligence Platform</span>
    </header>
  );
}
