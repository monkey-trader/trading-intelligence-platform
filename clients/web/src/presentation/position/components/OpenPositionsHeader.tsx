import { useState } from 'react';
import type { AccountSummary, PeriodKey } from '@/domain/account/AccountSummary';
import { formatDecimal } from '@/presentation/position/format/formatDecimal';
import { formatTimestamp } from '@/presentation/position/format/formatTimestamp';
import { pnlTone } from '@/presentation/position/format/pnlTone';
import styles from '@/presentation/position/components/OpenPositionsHeader.module.css';

type OpenPositionsHeaderProps = Readonly<{
  totalUnrealizedProfit: string;
  /** Unrealisierter PnL der Swing-Long-Positionen, getrennt von der Handels-Summe. */
  swingLongUnrealizedProfit?: string | undefined;
  positionCount: number;
  /** Ansicht des aktiven Reiters; im Swing-Long-Reiter ist der Swing-Long-PnL der Hauptwert. */
  view?: 'trades' | 'swing-long' | undefined;
  fetchedAt: number;
  onRefresh: () => void;
  /** Kontostand und Monats-%; ohne Angabe (Kontostand nicht verfügbar) bleibt der Bereich weg. */
  account?: AccountSummary | undefined;
}>;

const MONEY_FORMAT = { minimumFractionDigits: 2, maximumFractionDigits: 2 } as const;
const SIGNED_MONEY_FORMAT = { ...MONEY_FORMAT, signDisplay: 'exceptZero' } as const;

const PERIOD_LABELS: Readonly<Record<PeriodKey, string>> = {
  today: 'Heute',
  '7d': '7T',
  '30d': '30T',
};

/** Kopf mit PnL-Summe, Kontostand, Monats-%, Anzahl Positionen, Zeitpunkt der Daten und Aktualisieren-Knopf. */
export function OpenPositionsHeader({
  totalUnrealizedProfit,
  swingLongUnrealizedProfit,
  positionCount,
  view = 'trades',
  fetchedAt,
  onRefresh,
  account,
}: OpenPositionsHeaderProps) {
  const isSwingView = view === 'swing-long' && swingLongUnrealizedProfit !== undefined;
  const mainTotal = isSwingView ? swingLongUnrealizedProfit : totalUnrealizedProfit;
  const formattedTotal = formatDecimal(mainTotal, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
    signDisplay: 'exceptZero',
  });
  const [selectedPeriod, setSelectedPeriod] = useState<PeriodKey>('today');
  const periodPnl = account?.periods.find((entry) => entry.period === selectedPeriod);
  const positionCountLabel = positionCount === 1 ? '1 Position' : `${positionCount} Positionen`;

  return (
    <section className={styles.header} aria-label="Zusammenfassung">
      <div className={styles.summary}>
        <span className={styles.label}>
          {isSwingView ? 'Unrealisierter PnL Swing Long' : 'Unrealisierter PnL'}
        </span>
        <span className={`${styles.total} ${styles[pnlTone(mainTotal, 2)]}`}>
          {formattedTotal}
          {!isSwingView && account?.unrealizedPnlPercent !== undefined && (
            <span className={styles.percent}>
              {` (${formatDecimal(account.unrealizedPnlPercent, SIGNED_MONEY_FORMAT)} %)`}
            </span>
          )}
        </span>
        <div className={styles.meta}>
          <span>{positionCountLabel}</span>
          <span>{`Zuletzt aktualisiert ${formatTimestamp(fetchedAt)}`}</span>
        </div>
      </div>
      {account && (
        <dl className={styles.account} aria-label="Konto">
          <div className={styles.accountItem}>
            <dt className={styles.label}>Kontostand</dt>
            <dd className={styles.accountValue}>{formatDecimal(account.equity, MONEY_FORMAT)}</dd>
          </div>
          <div className={styles.accountItem}>
            <dt className={styles.label}>
              <span role="group" aria-label="Zeitraum" className={styles.periods}>
                {account.periods.map((entry) => (
                  <button
                    key={entry.period}
                    type="button"
                    className={styles.periodButton}
                    aria-pressed={entry.period === selectedPeriod}
                    onClick={() => setSelectedPeriod(entry.period)}
                  >
                    {PERIOD_LABELS[entry.period]}
                  </button>
                ))}
              </span>
            </dt>
            {periodPnl && (
              <dd className={`${styles.accountValue} ${styles[pnlTone(periodPnl.realizedPnl, 2)]}`}>
                {formatDecimal(periodPnl.realizedPnl, SIGNED_MONEY_FORMAT)}
                {periodPnl.pnlPercent !== undefined && (
                  <span className={styles.percent}>
                    {` (${formatDecimal(periodPnl.pnlPercent, SIGNED_MONEY_FORMAT)} %)`}
                  </span>
                )}
              </dd>
            )}
          </div>
        </dl>
      )}
      {swingLongUnrealizedProfit !== undefined && (
        <div role="group" aria-label="Swing Long" className={styles.swingLong}>
          <span className={styles.swingLongTitle}>Swing Long</span>
          {!isSwingView && (
            <div className={styles.accountItem}>
              <span className={styles.label}>Unrealisiert</span>
              <span
                className={`${styles.accountValue} ${styles[pnlTone(swingLongUnrealizedProfit, 2)]}`}
              >
                {formatDecimal(swingLongUnrealizedProfit, SIGNED_MONEY_FORMAT)}
              </span>
            </div>
          )}
          {account && (
            <div className={styles.accountItem}>
              <span className={styles.label}>Realisiert (Monat)</span>
              <span
                className={`${styles.accountValue} ${styles[pnlTone(account.swingLongRealizedPnl, 2)]}`}
              >
                {formatDecimal(account.swingLongRealizedPnl, SIGNED_MONEY_FORMAT)}
              </span>
            </div>
          )}
        </div>
      )}
      <button type="button" className={styles.refreshButton} onClick={onRefresh}>
        Aktualisieren
      </button>
    </section>
  );
}
