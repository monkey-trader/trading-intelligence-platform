import type { OpenPositionsGateway } from '@/domain/position/OpenPositionsGateway';
import { OpenPositionsHeader } from '@/presentation/position/components/OpenPositionsHeader';
import { OpenPositionsTable } from '@/presentation/position/components/OpenPositionsTable';
import { useOpenPositionsViewModel } from '@/presentation/position/hooks/useOpenPositionsViewModel';
import styles from '@/presentation/position/components/OpenPositionsPage.module.css';

type OpenPositionsPageProps = Readonly<{
  gateway: OpenPositionsGateway;
}>;

/** Seite "Offene Positionen": lädt über das injizierte Gateway und pollt. */
export function OpenPositionsPage({ gateway }: OpenPositionsPageProps) {
  const { status, snapshot, errorMessage, isStale, refresh } = useOpenPositionsViewModel(gateway);

  return (
    <main className={styles.page}>
      <h1 className={styles.title}>Offene Positionen</h1>

      {status === 'loading' && (
        <p className={styles.message} role="status">
          Lade offene Positionen …
        </p>
      )}

      {status === 'error' && (
        <div className={styles.errorPanel}>
          <p role="alert" className={styles.errorText}>
            {errorMessage}
          </p>
          <button type="button" className={styles.retryButton} onClick={refresh}>
            Erneut versuchen
          </button>
        </div>
      )}

      {snapshot && (
        <>
          <OpenPositionsHeader
            totalUnrealizedProfit={snapshot.totalUnrealizedProfit}
            positionCount={snapshot.positions.length}
            fetchedAt={snapshot.fetchedAt}
            onRefresh={refresh}
          />
          {isStale && errorMessage && (
            <p role="status" className={styles.staleWarning}>
              {`Daten möglicherweise veraltet. ${errorMessage}`}
            </p>
          )}
          {snapshot.positions.length > 0 ? (
            <OpenPositionsTable positions={snapshot.positions} />
          ) : (
            <p className={styles.message}>Keine offenen Positionen.</p>
          )}
        </>
      )}
    </main>
  );
}
