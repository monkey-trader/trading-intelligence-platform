import { useMemo, useState } from 'react';
import type { AssetIconsGateway } from '@/domain/asset/AssetIconsGateway';
import { useAssetIcons } from '@/presentation/position/hooks/useAssetIcons';
import type { PositionChartGateway } from '@/domain/position/PositionChartGateway';
import type { ChartRendererFactory } from '@/presentation/position/chart/ChartRenderer';
import { PositionChartPanel } from '@/presentation/position/components/PositionChartPanel';
import type { OpenPositionsGateway } from '@/domain/position/OpenPositionsGateway';
import { OpenPositionsHeader } from '@/presentation/position/components/OpenPositionsHeader';
import { OpenPositionsTable } from '@/presentation/position/components/OpenPositionsTable';
import { useOpenPositionsViewModel } from '@/presentation/position/hooks/useOpenPositionsViewModel';
import type { InitialStopLossGateway } from '@/domain/position/InitialStopLossGateway';
import styles from '@/presentation/position/components/OpenPositionsPage.module.css';

type OpenPositionsPageProps = Readonly<{
  gateway: OpenPositionsGateway;
  /** Ohne Chart-Gateway und Renderer gibt es keine Chart-Auswahl. */
  chartGateway?: PositionChartGateway;
  createChartRenderer?: ChartRendererFactory;
  /** Ohne Gateway zeigt die Tabelle Ticker statt Icons. */
  assetIconsGateway?: AssetIconsGateway;
  /** Ohne Gateway gibt es kein manuelles Eintragen des Ursprungs-SL. */
  initialStopLossGateway?: InitialStopLossGateway;
}>;

/** Seite "Offene Positionen": lädt über das injizierte Gateway und pollt. */
export function OpenPositionsPage({
  gateway,
  chartGateway,
  createChartRenderer,
  assetIconsGateway,
  initialStopLossGateway,
}: OpenPositionsPageProps) {
  const { status, snapshot, errorMessage, isStale, refresh } = useOpenPositionsViewModel(gateway);
  const symbols = useMemo(
    () => snapshot?.positions.map((position) => position.symbol) ?? [],
    [snapshot],
  );
  const iconsBySymbol = useAssetIcons(assetIconsGateway, symbols);

  const [chartPositionId, setChartPositionId] = useState<string>();
  const toggleChart = (positionId: string) =>
    setChartPositionId((current) => (current === positionId ? undefined : positionId));

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
            <OpenPositionsTable
              positions={snapshot.positions}
              iconsBySymbol={iconsBySymbol}
              initialStopLossGateway={initialStopLossGateway}
              onInitialStopLossChanged={refresh}
              selectedPositionId={chartPositionId}
              onSelectPosition={chartGateway && createChartRenderer ? toggleChart : undefined}
            />
          ) : (
            <p className={styles.message}>Keine offenen Positionen.</p>
          )}
          {chartPositionId && chartGateway && createChartRenderer && (
            <PositionChartPanel
              positionId={chartPositionId}
              gateway={chartGateway}
              createRenderer={createChartRenderer}
              onClose={() => setChartPositionId(undefined)}
            />
          )}
        </>
      )}
    </main>
  );
}
