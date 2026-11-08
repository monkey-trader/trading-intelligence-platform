import { useMemo, useState } from 'react';
import type { AccountSummaryGateway } from '@/domain/account/AccountSummaryGateway';
import { useAccountSummary } from '@/presentation/account/hooks/useAccountSummary';
import type { AssetIconsGateway } from '@/domain/asset/AssetIconsGateway';
import { useAssetIcons } from '@/presentation/position/hooks/useAssetIcons';
import type { PositionChartGateway } from '@/domain/position/PositionChartGateway';
import type { ChartRendererFactory } from '@/presentation/position/chart/ChartRenderer';
import { PositionChartPanel } from '@/presentation/position/components/PositionChartPanel';
import type { OpenPositionsGateway } from '@/domain/position/OpenPositionsGateway';
import { OpenPositionsHeader } from '@/presentation/position/components/OpenPositionsHeader';
import { OpenPositionsTable } from '@/presentation/position/components/OpenPositionsTable';
import { useOpenPositionsViewModel } from '@/presentation/position/hooks/useOpenPositionsViewModel';
import type { PriceStreamGateway } from '@/domain/price/PriceStream';
import { useLivePrices } from '@/presentation/position/hooks/useLivePrices';
import type { InitialStopLossGateway } from '@/domain/position/InitialStopLossGateway';
import type { SwingLongGateway } from '@/domain/position/SwingLongGateway';
import { PositionTabs, type PositionTab } from '@/presentation/position/components/PositionTabs';
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
  /** Ohne Gateway zeigt der Kopf keinen Kontostand. */
  accountSummaryGateway?: AccountSummaryGateway;
  /** Ohne Gateway zeigt die Tabelle keine Live-Kurse. */
  priceStreamGateway?: PriceStreamGateway;
  /** Ohne Gateway gibt es keine Swing-Long-Schalter. */
  swingLongGateway?: SwingLongGateway | undefined;
}>;

const NO_PRICE_STREAM: PriceStreamGateway = { subscribe: () => () => undefined };

/** Seite "Offene Positionen": lädt über das injizierte Gateway und pollt. */
export function OpenPositionsPage({
  gateway,
  chartGateway,
  createChartRenderer,
  assetIconsGateway,
  initialStopLossGateway,
  accountSummaryGateway,
  priceStreamGateway,
  swingLongGateway,
}: OpenPositionsPageProps) {
  const { status, snapshot, errorMessage, isStale, refresh } = useOpenPositionsViewModel(gateway);
  const account = useAccountSummary(accountSummaryGateway);
  const livePrices = useLivePrices(priceStreamGateway ?? NO_PRICE_STREAM);
  const priceProps = priceStreamGateway
    ? {
        prices: livePrices.prices,
        pricesStale: livePrices.stale,
        pricesAnnouncement: livePrices.announcement,
      }
    : {};
  const symbols = useMemo(
    () => snapshot?.positions.map((position) => position.symbol) ?? [],
    [snapshot],
  );
  const iconsBySymbol = useAssetIcons(assetIconsGateway, symbols);

  const [activeTab, setActiveTab] = useState<PositionTab>('trades');
  const visiblePositions = useMemo(
    () =>
      snapshot?.positions.filter(
        (position) => position.swingLong === (activeTab === 'swing-long'),
      ) ?? [],
    [snapshot, activeTab],
  );
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

      {status === 'loading' && !snapshot && (
        <OpenPositionsTable
          positions={[]}
          iconsBySymbol={iconsBySymbol}
          isLoading
          initialStopLossGateway={initialStopLossGateway}
          onInitialStopLossChanged={refresh}
          swingLongGateway={swingLongGateway}
          onSwingLongChanged={refresh}
        />
      )}

      {snapshot && (
        <>
          <OpenPositionsHeader
            totalUnrealizedProfit={snapshot.totalUnrealizedProfit}
            swingLongUnrealizedProfit={snapshot.swingLongUnrealizedProfit}
            positionCount={visiblePositions.length}
            view={activeTab}
            fetchedAt={snapshot.fetchedAt}
            onRefresh={refresh}
            account={account}
          />
          {isStale && errorMessage && (
            <p role="status" className={styles.staleWarning}>
              {`Daten möglicherweise veraltet. ${errorMessage}`}
            </p>
          )}
          <PositionTabs active={activeTab} onChange={setActiveTab} />
          {visiblePositions.length > 0 ? (
            <OpenPositionsTable
              positions={visiblePositions}
              iconsBySymbol={iconsBySymbol}
              isLoading={status === 'loading'}
              initialStopLossGateway={initialStopLossGateway}
              onInitialStopLossChanged={refresh}
              swingLongGateway={swingLongGateway}
              onSwingLongChanged={refresh}
              selectedPositionId={chartPositionId}
              onSelectPosition={chartGateway && createChartRenderer ? toggleChart : undefined}
              {...priceProps}
            />
          ) : (
            <p className={styles.message}>
              {activeTab === 'swing-long'
                ? 'Keine Swing-Long-Positionen.'
                : 'Keine offenen Positionen.'}
            </p>
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
