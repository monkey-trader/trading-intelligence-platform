import { formatDecimal } from '@/presentation/position/format/formatDecimal';
import { formatTimestamp } from '@/presentation/position/format/formatTimestamp';
import { pnlTone } from '@/presentation/position/format/pnlTone';
import styles from '@/presentation/position/components/OpenPositionsHeader.module.css';

type OpenPositionsHeaderProps = Readonly<{
  totalUnrealizedProfit: string;
  positionCount: number;
  fetchedAt: number;
  onRefresh: () => void;
}>;

/** Kopf mit PnL-Summe, Anzahl Positionen, Zeitpunkt der Daten und Aktualisieren-Knopf. */
export function OpenPositionsHeader({
  totalUnrealizedProfit,
  positionCount,
  fetchedAt,
  onRefresh,
}: OpenPositionsHeaderProps) {
  const formattedTotal = formatDecimal(totalUnrealizedProfit, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
    signDisplay: 'exceptZero',
  });
  const positionCountLabel = positionCount === 1 ? '1 Position' : `${positionCount} Positionen`;

  return (
    <section className={styles.header} aria-label="Zusammenfassung">
      <div className={styles.summary}>
        <span className={styles.label}>Unrealisierter PnL</span>
        <span className={`${styles.total} ${styles[pnlTone(totalUnrealizedProfit, 2)]}`}>
          {formattedTotal}
        </span>
        <div className={styles.meta}>
          <span>{positionCountLabel}</span>
          <span>{`Zuletzt aktualisiert ${formatTimestamp(fetchedAt)}`}</span>
        </div>
      </div>
      <button type="button" className={styles.refreshButton} onClick={onRefresh}>
        Aktualisieren
      </button>
    </section>
  );
}
