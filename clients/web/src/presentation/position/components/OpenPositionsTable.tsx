import type { OpenPosition } from '@/domain/position/OpenPosition';
import { formatDecimal } from '@/presentation/position/format/formatDecimal';
import {
  formatRatioAsPercent,
  MISSING_VALUE,
} from '@/presentation/position/format/formatRatioAsPercent';
import type { AssetIcon } from '@/domain/asset/AssetIcon';
import { CoinIcon } from '@/presentation/position/components/CoinIcon';
import { pnlTone } from '@/presentation/position/format/pnlTone';
import { Fragment, useState, type DragEvent, type KeyboardEvent, type ReactNode } from 'react';
import type { ColumnKey } from '@/presentation/position/columnOrder';
import { useColumnOrder } from '@/presentation/position/hooks/useColumnOrder';
import type { InitialStopLossGateway } from '@/domain/position/InitialStopLossGateway';
import { InitialStopLossControl } from '@/presentation/position/components/InitialStopLossControl';
import { BreakEvenTargetInfo } from '@/presentation/position/components/BreakEvenTargetInfo';
import { InfoDialog } from '@/presentation/shared/components/InfoDialog';
import styles from '@/presentation/position/components/OpenPositionsTable.module.css';

const COLUMN_LABELS = {
  symbol: 'Symbol',
  positionValue: 'Positionswert',
  margin: 'Margin',
  unrealizedProfit: 'Unreal. PnL',
  realisedProfit: 'Real. PnL',
  entry: 'Einstieg',
  tpSl: 'TP / SL',
  beTarget: 'TPs (je 33 %)',
} as const;

const SIDE_LABELS: Readonly<Record<OpenPosition['side'], string>> = {
  LONG: 'Long',
  SHORT: 'Short',
};

type OpenPositionsTableProps = Readonly<{
  positions: readonly OpenPosition[];
  /** Position, deren Chart gerade offen ist. */
  selectedPositionId?: string | undefined;
  onSelectPosition?: ((positionId: string) => void) | undefined;
  /** Icons und Anzeigenamen je Symbol; fehlt ein Eintrag, gilt der Ticker als Name. */
  iconsBySymbol?: ReadonlyMap<string, AssetIcon> | undefined;
  /** Ohne Gateway gibt es kein manuelles Eintragen des Ursprungs-SL. */
  initialStopLossGateway?: InitialStopLossGateway | undefined;
  /** Wird nach erfolgreichem Speichern/Zurücksetzen aufgerufen (Liste neu laden). */
  onInitialStopLossChanged?: (() => void) | undefined;
}>;

/** Beträge in USDT: 2 Nachkommastellen. */
const USDT_FORMAT = { minimumFractionDigits: 2, maximumFractionDigits: 2 } as const;

/** PnL-Werte: 2 Nachkommastellen, positive mit "+", damit Gewinn nicht nur an der Farbe hängt. */
const PNL_FORMAT = {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
  signDisplay: 'exceptZero',
} as const;

/** Offene Positionen als Tabelle; auf schmalen Bildschirmen als Karten. */
export function OpenPositionsTable({
  positions,
  selectedPositionId,
  onSelectPosition,
  iconsBySymbol,
  initialStopLossGateway,
  onInitialStopLossChanged,
}: OpenPositionsTableProps) {
  const { order, move, reset, isDefault } = useColumnOrder();
  const [draggedKey, setDraggedKey] = useState<ColumnKey | null>(null);
  const [dropTargetKey, setDropTargetKey] = useState<ColumnKey | null>(null);
  const [isBeTargetInfoOpen, setIsBeTargetInfoOpen] = useState(false);

  const endDrag = () => {
    setDraggedKey(null);
    setDropTargetKey(null);
  };

  const handleDrop = (event: DragEvent, targetKey: ColumnKey) => {
    event.preventDefault();
    if (draggedKey !== null) move(draggedKey, targetKey);
    endDrag();
  };

  const handleHandleKeyDown = (event: KeyboardEvent, columnKey: ColumnKey) => {
    if (!event.altKey) return;
    const index = order.indexOf(columnKey);
    const neighbourKey =
      event.key === 'ArrowRight'
        ? order[index + 1]
        : event.key === 'ArrowLeft'
          ? order[index - 1]
          : undefined;
    if (neighbourKey === undefined) return;
    event.preventDefault();
    move(columnKey, neighbourKey);
  };

  return (
    <>
      {/* Immer gerendert (leer, wenn nicht nötig), damit der Tabellenkopf nicht springt. */}
      <div className={styles.toolbar}>
        <button
          type="button"
          className={`${styles.infoButton} ${styles.infoButtonToolbar}`}
          aria-label="Erklärung zu den TPs"
          onClick={() => setIsBeTargetInfoOpen(true)}
        ></button>
        {!isDefault && (
          <button type="button" className={styles.resetButton} onClick={reset}>
            Spalten zurücksetzen
          </button>
        )}
      </div>
      {/* Zwischen 768 px und voller Tabellenbreite scrollt die Tabelle horizontal im Panel,
          das Symbol bleibt dabei links stehen. tabIndex, damit man auch per Tastatur scrollen kann. */}
      <div
        className={styles.scrollContainer}
        role="region"
        aria-label="Offene Positionen, horizontal scrollbar"
        tabIndex={0}
      >
        <table className={styles.table} aria-label="Offene Positionen">
          <thead>
            <tr>
              {order.map((columnKey) => {
                const label = COLUMN_LABELS[columnKey];
                if (columnKey === 'symbol') {
                  return (
                    <th
                      key={columnKey}
                      scope="col"
                      data-column-key={columnKey}
                      className={styles.headerCell}
                    >
                      {label}
                    </th>
                  );
                }
                const labelId = `column-label-${columnKey}`;
                const dropClass =
                  dropTargetKey === columnKey && draggedKey !== columnKey
                    ? ` ${styles.dropTarget}`
                    : '';
                return (
                  <th
                    key={columnKey}
                    scope="col"
                    data-column-key={columnKey}
                    aria-labelledby={labelId}
                    draggable
                    className={`${styles.headerCell}${dropClass}`}
                    onDragStart={(event) => {
                      event.dataTransfer.effectAllowed = 'move';
                      event.dataTransfer.setData('text/plain', columnKey);
                      setDraggedKey(columnKey);
                    }}
                    onDragOver={(event) => {
                      event.preventDefault();
                      setDropTargetKey(columnKey);
                    }}
                    onDrop={(event) => handleDrop(event, columnKey)}
                    onDragEnd={endDrag}
                  >
                    <span id={labelId}>{label}</span>
                    {columnKey === 'beTarget' && (
                      <button
                        type="button"
                        className={styles.infoButton}
                        aria-label="Erklärung zu den TPs"
                        onClick={() => setIsBeTargetInfoOpen(true)}
                      ></button>
                    )}
                    <button
                      type="button"
                      className={styles.dragHandle}
                      aria-label={`Spalte verschieben: ${label}`}
                      onKeyDown={(event) => handleHandleKeyDown(event, columnKey)}
                    />
                  </th>
                );
              })}
            </tr>
          </thead>
          <tbody>
            {positions.map((position) => (
              <OpenPositionRow
                key={`${position.positionId}-${position.side}`}
                position={position}
                order={order}
                isSelected={position.positionId === selectedPositionId}
                onSelectPosition={onSelectPosition}
                assetIcon={iconsBySymbol?.get(position.symbol)}
                initialStopLossGateway={initialStopLossGateway}
                onInitialStopLossChanged={onInitialStopLossChanged}
              />
            ))}
          </tbody>
        </table>
      </div>
      <InfoDialog
        title="TP1 bis TP3"
        isOpen={isBeTargetInfoOpen}
        onClose={() => setIsBeTargetInfoOpen(false)}
      >
        <BreakEvenTargetInfo />
      </InfoDialog>
    </>
  );
}

function OpenPositionRow({
  position,
  order,
  isSelected,
  onSelectPosition,
  assetIcon,
  initialStopLossGateway,
  onInitialStopLossChanged,
}: Readonly<{
  initialStopLossGateway?: InitialStopLossGateway | undefined;
  onInitialStopLossChanged?: (() => void) | undefined;
  assetIcon?: AssetIcon | undefined;
  position: OpenPosition;
  order: readonly ColumnKey[];
  isSelected: boolean;
  onSelectPosition?: ((positionId: string) => void) | undefined;
}>) {
  const profitToneClass = styles[pnlTone(position.unrealizedProfit, 2)];
  const realisedToneClass = styles[pnlTone(position.realisedProfit, 2)];
  const isLong = position.side === 'LONG';
  const tickerFallback = (position.symbol.split('-')[0] ?? position.symbol).toUpperCase();
  const displayName = assetIcon?.displayName ?? tickerFallback;

  const firstLevel = position.levels?.find((level) => level.kind === 'ONE_R');
  const secondLevel = position.levels?.find((level) => level.kind === 'TWO_R');
  const thirdLevel = position.levels?.find((level) => level.kind === 'THREE_R');
  const cells: Record<ColumnKey, ReactNode> = {
    symbol: (
      <th
        scope="row"
        data-label={COLUMN_LABELS.symbol}
        className={`${styles.cell} ${styles.symbolCell} ${isLong ? styles.barLong : styles.barShort}`}
      >
        <div>
          <div className={styles.symbolMain}>
            <CoinIcon
              baseAsset={assetIcon?.baseAsset ?? tickerFallback}
              displayName={displayName}
              iconUrl={assetIcon?.iconUrl ?? null}
            />
            <span className={styles.symbolName}>{displayName}</span>
            <button
              type="button"
              className={styles.chartButton}
              aria-label={`Chart ${isSelected ? 'ausblenden' : 'anzeigen'}: ${displayName}`}
              aria-expanded={isSelected}
              onClick={() => onSelectPosition?.(position.positionId)}
            >
              Chart
            </button>
          </div>
          <div className={styles.chips}>
            <span className={`${styles.chip} ${isLong ? styles.chipLong : styles.chipShort}`}>
              {SIDE_LABELS[position.side]}
            </span>
            <span className={styles.chip}>{position.isolated ? 'Isolated' : 'Cross'}</span>
            <span className={styles.chip}>{`${position.leverage}X`}</span>
            {position.breakEvenTriggerReached === true && position.breakEven !== true && (
              <span className={`${styles.chip} ${styles.chipAction}`}>2R · BE ziehen</span>
            )}
          </div>
        </div>
      </th>
    ),
    positionValue: (
      <td data-label={COLUMN_LABELS.positionValue} className={styles.cell}>
        {formatDecimal(position.positionValue, USDT_FORMAT)}{' '}
        <span className={styles.unit}>USDT</span>
      </td>
    ),
    margin: (
      <td data-label={COLUMN_LABELS.margin} className={styles.cell}>
        {formatDecimal(position.margin, USDT_FORMAT)} <span className={styles.unit}>USDT</span>
      </td>
    ),
    unrealizedProfit: (
      <td
        data-label={COLUMN_LABELS.unrealizedProfit}
        className={`${styles.cell} ${profitToneClass}`}
      >
        <div className={styles.twoLine}>
          <span>{formatDecimal(position.unrealizedProfit, PNL_FORMAT)}</span>
          <span className={styles.lineSub}>
            ({formatRatioAsPercent(position.pnlRatio, { signDisplay: 'exceptZero' })})
          </span>
        </div>
      </td>
    ),
    realisedProfit: (
      <td
        data-label={COLUMN_LABELS.realisedProfit}
        className={`${styles.cell} ${realisedToneClass}`}
      >
        {formatDecimal(position.realisedProfit, PNL_FORMAT)}
      </td>
    ),
    entry: (
      <td data-label={COLUMN_LABELS.entry} className={styles.cell}>
        {formatDecimal(position.avgOpenPrice)}
      </td>
    ),
    tpSl: (
      <td data-label={COLUMN_LABELS.tpSl} className={styles.cell}>
        <div className={styles.twoLine}>
          <span>
            <span className={styles.lineMuted}>TP </span>
            {position.takeProfit === null ? MISSING_VALUE : formatDecimal(position.takeProfit)}
          </span>
          <span className={position.stopLoss === null ? undefined : styles.slLine}>
            <span className={styles.lineMuted}>SL </span>
            {position.stopLoss === null ? MISSING_VALUE : formatDecimal(position.stopLoss)}
            {position.breakEven === true && (
              <span className={styles.breakEvenBadge} role="img" aria-label="Break-even">
                BE
              </span>
            )}
            {position.breakEven === false && position.stopLoss === null && (
              <span className={styles.noStopLossHint}>kein SL</span>
            )}
          </span>
        </div>
      </td>
    ),
    beTarget: (
      <td data-label={COLUMN_LABELS.beTarget} className={styles.cell}>
        <div className={styles.beTargetStack}>
          {firstLevel && secondLevel && thirdLevel ? (
            <div className={styles.levelStack}>
              <span className={`${styles.levelPill} ${styles.levelFirst}`}>
                {`TP1 ${formatDecimal(firstLevel.price)}`}
              </span>
              <span className={`${styles.levelPill} ${styles.levelBe}`}>
                {`TP2 ${formatDecimal(secondLevel.price)} · BE`}
              </span>
              <span className={`${styles.levelPill} ${styles.levelTarget}`}>
                {`TP3 ${formatDecimal(thirdLevel.price)}`}
              </span>
            </div>
          ) : (
            <span aria-label="Risiko unbekannt">{MISSING_VALUE}</span>
          )}
          {initialStopLossGateway && (
            <InitialStopLossControl
              position={position}
              gateway={initialStopLossGateway}
              onChanged={onInitialStopLossChanged}
            />
          )}
        </div>
      </td>
    ),
  };

  return (
    <tr className={styles.row}>
      {order.map((columnKey) => (
        <Fragment key={columnKey}>{cells[columnKey]}</Fragment>
      ))}
    </tr>
  );
}
