import type { OpenPosition } from '@/domain/position/OpenPosition';
import { formatDecimal } from '@/presentation/position/format/formatDecimal';
import {
  formatRatioAsPercent,
  MISSING_VALUE,
} from '@/presentation/position/format/formatRatioAsPercent';
import { formatDateTime } from '@/presentation/position/format/formatTimestamp';
import { pnlTone } from '@/presentation/position/format/pnlTone';
import { Fragment, useState, type DragEvent, type KeyboardEvent, type ReactNode } from 'react';
import type { ColumnKey } from '@/presentation/position/columnOrder';
import { useColumnOrder } from '@/presentation/position/hooks/useColumnOrder';
import styles from '@/presentation/position/components/OpenPositionsTable.module.css';

const COLUMN_LABELS = {
  symbol: 'Symbol',
  side: 'Seite',
  amount: 'Menge',
  entry: 'Einstieg',
  mark: 'Mark',
  leverage: 'Hebel',
  margin: 'Margin',
  unrealizedProfit: 'Unreal. PnL',
  pnlRatio: 'PnL %',
  realisedProfit: 'Real. PnL',
  stopLoss: 'SL',
  takeProfit: 'TP',
  openTime: 'Eröffnet',
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
}>;

/** Margin in USDT: 2 Nachkommastellen. */
const MARGIN_FORMAT = { minimumFractionDigits: 2, maximumFractionDigits: 2 } as const;

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
}: OpenPositionsTableProps) {
  const { order, move, reset, isDefault } = useColumnOrder();
  const [draggedKey, setDraggedKey] = useState<ColumnKey | null>(null);
  const [dropTargetKey, setDropTargetKey] = useState<ColumnKey | null>(null);

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
              />
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}

function OpenPositionRow({
  position,
  order,
  isSelected,
  onSelectPosition,
}: Readonly<{
  position: OpenPosition;
  order: readonly ColumnKey[];
  isSelected: boolean;
  onSelectPosition?: ((positionId: string) => void) | undefined;
}>) {
  // Tönung passend zur Anzeige: PnL mit 2 Stellen, Verhältnis als Prozent mit 2 Stellen (= 4 Stellen).
  const profitToneClass = styles[pnlTone(position.unrealizedProfit, 2)];
  const ratioToneClass = styles[pnlTone(position.pnlRatio, 4)];
  const realisedToneClass = styles[pnlTone(position.realisedProfit, 2)];
  const sideClass = position.side === 'LONG' ? styles.sideLong : styles.sideShort;

  const cells: Record<ColumnKey, ReactNode> = {
    symbol: (
      <th
        scope="row"
        data-label={COLUMN_LABELS.symbol}
        className={`${styles.cell} ${styles.symbolCell}`}
      >
        {position.symbol}
        <button
          type="button"
          className={styles.chartButton}
          aria-label={`Chart ${isSelected ? 'ausblenden' : 'anzeigen'}: ${position.symbol}`}
          aria-expanded={isSelected}
          onClick={() => onSelectPosition?.(position.positionId)}
        >
          Chart
        </button>
      </th>
    ),
    side: (
      <td data-label={COLUMN_LABELS.side} className={`${styles.cell} ${sideClass}`}>
        {SIDE_LABELS[position.side]}
      </td>
    ),
    amount: (
      <td data-label={COLUMN_LABELS.amount} className={styles.cell}>
        {formatDecimal(position.positionAmount)}
      </td>
    ),
    entry: (
      <td data-label={COLUMN_LABELS.entry} className={styles.cell}>
        {formatDecimal(position.avgOpenPrice)}
      </td>
    ),
    mark: (
      <td data-label={COLUMN_LABELS.mark} className={styles.cell}>
        {formatDecimal(position.markPrice)}
      </td>
    ),
    leverage: (
      <td data-label={COLUMN_LABELS.leverage} className={styles.cell}>
        {`${position.leverage}×`}
      </td>
    ),
    margin: (
      <td data-label={COLUMN_LABELS.margin} className={styles.cell}>
        {formatDecimal(position.margin, MARGIN_FORMAT)}
      </td>
    ),
    unrealizedProfit: (
      <td
        data-label={COLUMN_LABELS.unrealizedProfit}
        className={`${styles.cell} ${profitToneClass}`}
      >
        {formatDecimal(position.unrealizedProfit, PNL_FORMAT)}
      </td>
    ),
    pnlRatio: (
      <td data-label={COLUMN_LABELS.pnlRatio} className={`${styles.cell} ${ratioToneClass}`}>
        {formatRatioAsPercent(position.pnlRatio, { signDisplay: 'exceptZero' })}
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
    stopLoss: (
      <td data-label={COLUMN_LABELS.stopLoss} className={styles.cell}>
        <span>
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
      </td>
    ),
    takeProfit: (
      <td data-label={COLUMN_LABELS.takeProfit} className={styles.cell}>
        {position.takeProfit === null ? MISSING_VALUE : formatDecimal(position.takeProfit)}
      </td>
    ),
    openTime: (
      <td data-label={COLUMN_LABELS.openTime} className={styles.cell}>
        {formatDateTime(position.openTime)}
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
