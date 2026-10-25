import type { OpenPosition } from '@/domain/position/OpenPosition';
import { formatDecimal } from '@/presentation/position/format/formatDecimal';
import {
  formatRatioAsPercent,
  MISSING_VALUE,
} from '@/presentation/position/format/formatRatioAsPercent';
import type { AssetIcon } from '@/domain/asset/AssetIcon';
import { CoinIcon } from '@/presentation/position/components/CoinIcon';
import { pnlTone } from '@/presentation/position/format/pnlTone';
import {
  Fragment,
  useEffect,
  useMemo,
  useRef,
  useState,
  type DragEvent,
  type KeyboardEvent,
  type ReactNode,
} from 'react';
import type { ColumnKey } from '@/presentation/position/columnOrder';
import { useColumnOrder } from '@/presentation/position/hooks/useColumnOrder';
import type { InitialStopLossGateway } from '@/domain/position/InitialStopLossGateway';
import { InitialStopLossControl } from '@/presentation/position/components/InitialStopLossControl';
import { BreakEvenTargetInfo } from '@/presentation/position/components/BreakEvenTargetInfo';
import { useTableSort } from '@/presentation/position/hooks/useTableSort';
import { sortPositions, type SortKey } from '@/presentation/position/sorting/sortPositions';
import { InfoDialog } from '@/presentation/shared/components/InfoDialog';
import styles from '@/presentation/position/components/OpenPositionsTable.module.css';

const COLUMN_LABELS = {
  symbol: 'Symbol',
  valueMargin: 'Wert / Margin',
  unrealizedProfit: 'Unreal. PnL',
  realisedProfit: 'Real. PnL',
  entrySl: 'Einstieg / SL',
  be: 'BE (2R)',
  tp1: 'TP1 (3R)',
} as const;

const SORT_LABELS: Readonly<Record<SortKey, string>> = {
  symbol: 'Symbol',
  value: 'Wert',
  margin: 'Margin',
  unrealizedProfit: 'Unreal. PnL',
  realisedProfit: 'Real. PnL',
  entry: 'Einstieg',
  stopLoss: 'SL',
  be: 'BE (2R)',
  tp1: 'TP1 (3R)',
};

/** Sortierbare Werte je Spalte; gestapelte Spalten haben zwei. */
const COLUMN_SORT_KEYS: Readonly<Record<ColumnKey, readonly SortKey[]>> = {
  symbol: ['symbol'],
  valueMargin: ['value', 'margin'],
  unrealizedProfit: ['unrealizedProfit'],
  realisedProfit: ['realisedProfit'],
  entrySl: ['entry', 'stopLoss'],
  be: ['be'],
  tp1: ['tp1'],
};

const SORT_KEY_ORDER: readonly SortKey[] = [
  'symbol',
  'value',
  'margin',
  'unrealizedProfit',
  'realisedProfit',
  'entry',
  'stopLoss',
  'be',
  'tp1',
];

const PLACEHOLDER_ROW_COUNT = 3;

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
  /** Lädt gerade: Sortieren gesperrt, Platzhalterzeilen statt leerer Tabelle. */
  isLoading?: boolean | undefined;
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
  isLoading = false,
}: OpenPositionsTableProps) {
  const { order, move, reset, isDefault } = useColumnOrder();
  const { sort, cycle, select, toggleDirection } = useTableSort();
  const [announcement, setAnnouncement] = useState('');
  const focusHandleKey = useRef<ColumnKey | null>(null);
  const focusButtonIndex = useRef(0);
  const scrollRef = useRef<HTMLDivElement>(null);
  const sortedPositions = useMemo(() => sortPositions(positions, sort), [positions, sort]);
  const isSortDisabled = isLoading || positions.length === 0;

  // Nach dem Verschieben bleibt der Fokus am Griff, auch wenn React die Kopfzelle umhängt.
  useEffect(() => {
    const columnKey = focusHandleKey.current;
    if (columnKey === null) return;
    focusHandleKey.current = null;
    scrollRef.current
      ?.querySelectorAll<HTMLElement>(`th[data-column-key="${columnKey}"] button`)
      [focusButtonIndex.current]?.focus();
  }, [order]);

  const describeSort = (key: SortKey, direction: 'asc' | 'desc' | null): string =>
    direction === null
      ? 'Sortierung aufgehoben, Reihenfolge der API'
      : `Sortiert nach ${SORT_LABELS[key]}, ${direction === 'asc' ? 'aufsteigend' : 'absteigend'}`;

  const handleCycle = (key: SortKey) => {
    cycle(key);
    const direction = sort?.key === key ? (sort.direction === 'asc' ? 'desc' : null) : 'asc';
    setAnnouncement(describeSort(key, direction));
  };

  const moveColumnAndAnnounce = (fromKey: ColumnKey, toKey: ColumnKey) => {
    move(fromKey, toKey);
    setAnnouncement(
      `Spalte ${COLUMN_LABELS[fromKey]} verschoben, von Stelle ${order.indexOf(fromKey) + 1} auf Stelle ${order.indexOf(toKey) + 1} von ${order.length}`,
    );
  };
  const [draggedKey, setDraggedKey] = useState<ColumnKey | null>(null);
  const [dropTargetKey, setDropTargetKey] = useState<ColumnKey | null>(null);
  const [isBeTargetInfoOpen, setIsBeTargetInfoOpen] = useState(false);

  const endDrag = () => {
    setDraggedKey(null);
    setDropTargetKey(null);
  };

  const handleDrop = (event: DragEvent, targetKey: ColumnKey) => {
    event.preventDefault();
    if (draggedKey !== null) moveColumnAndAnnounce(draggedKey, targetKey);
    endDrag();
  };

  const handleHeaderKeyDown = (event: KeyboardEvent<HTMLElement>, columnKey: ColumnKey) => {
    if (!event.altKey) return;
    const buttons = Array.from(event.currentTarget.querySelectorAll('button'));
    const focusedButton = event.target instanceof HTMLButtonElement ? event.target : undefined;
    focusButtonIndex.current = Math.max(
      0,
      focusedButton === undefined ? 0 : buttons.indexOf(focusedButton),
    );
    const index = order.indexOf(columnKey);
    const neighbourKey =
      event.key === 'ArrowRight'
        ? order[index + 1]
        : event.key === 'ArrowLeft'
          ? order[index - 1]
          : undefined;
    if (neighbourKey === undefined) return;
    event.preventDefault();
    focusHandleKey.current = columnKey;
    moveColumnAndAnnounce(columnKey, neighbourKey);
  };

  return (
    <>
      {/* Immer gerendert, damit der Tabellenkopf nicht springt. Mobil: Sortierauswahl statt Kopf. */}
      <div className={styles.toolbar}>
        <div className={styles.mobileSort}>
          <label htmlFor="positions-sort-select" className={styles.mobileSortLabel}>
            Sortieren
          </label>
          <select
            id="positions-sort-select"
            className={styles.mobileSortSelect}
            value={sort?.key ?? ''}
            disabled={isSortDisabled}
            onChange={(event) => {
              const key = SORT_KEY_ORDER.find((candidate) => candidate === event.target.value);
              select(key ?? null);
              setAnnouncement(describeSort(key ?? 'symbol', key === undefined ? null : 'asc'));
            }}
          >
            <option value="">Reihenfolge der API</option>
            {SORT_KEY_ORDER.map((key) => (
              <option key={key} value={key}>
                {SORT_LABELS[key]}
              </option>
            ))}
          </select>
          <button
            type="button"
            className={styles.mobileSortDirection}
            aria-label="Richtung wechseln"
            disabled={isSortDisabled || sort === null}
            onClick={() => {
              toggleDirection();
              if (sort !== null) {
                setAnnouncement(describeSort(sort.key, sort.direction === 'asc' ? 'desc' : 'asc'));
              }
            }}
          >
            {sort?.direction === 'desc' ? '↓' : '↑'}
          </button>
        </div>
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
      <p role="status" className={styles.visuallyHidden}>
        {announcement}
      </p>
      {/* Ab 768 px passt die Tabelle in die Breite (auch bei 1280 px); der Container fängt nur
          Ausnahmen (sehr breite Werte) ab. tabIndex, damit man dann per Tastatur scrollen kann. */}
      <div
        ref={scrollRef}
        className={styles.scrollContainer}
        role="region"
        aria-label="Offene Positionen, horizontal scrollbar"
        tabIndex={0}
      >
        <table
          className={styles.table}
          aria-label="Offene Positionen"
          aria-busy={isLoading || undefined}
        >
          <thead>
            <tr>
              {order.map((columnKey) => {
                const label = COLUMN_LABELS[columnKey];
                const sortKeys = COLUMN_SORT_KEYS[columnKey];
                const activeKey = sortKeys.find((key) => key === sort?.key);
                const ariaSort =
                  activeKey === undefined
                    ? 'none'
                    : sort?.direction === 'asc'
                      ? 'ascending'
                      : 'descending';
                const sortButtons = (
                  <span className={styles.sortGroup}>
                    {sortKeys.map((key, index) => (
                      <Fragment key={key}>
                        {index > 0 && <span className={styles.sortSeparator}>/</span>}
                        <button
                          type="button"
                          className={styles.sortButton}
                          disabled={isSortDisabled}
                          aria-label={`${SORT_LABELS[key]} sortieren: ${
                            sort?.key !== key
                              ? 'aufsteigend'
                              : sort.direction === 'asc'
                                ? 'absteigend'
                                : 'aus'
                          }`}
                          onClick={() => handleCycle(key)}
                        >
                          {SORT_LABELS[key]}
                          {sort?.key === key && (
                            <span aria-hidden="true" className={styles.sortArrow}>
                              {sort.direction === 'asc' ? '↑' : '↓'}
                            </span>
                          )}
                        </button>
                      </Fragment>
                    ))}
                  </span>
                );
                if (columnKey === 'symbol') {
                  return (
                    <th
                      key={columnKey}
                      scope="col"
                      data-column-key={columnKey}
                      aria-sort={ariaSort}
                      className={styles.headerCell}
                    >
                      {sortButtons}
                    </th>
                  );
                }
                const dropClass =
                  dropTargetKey === columnKey && draggedKey !== columnKey
                    ? ` ${styles.dropTarget}`
                    : '';
                return (
                  <th
                    key={columnKey}
                    scope="col"
                    data-column-key={columnKey}
                    aria-sort={ariaSort}
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
                    onKeyDown={(event) => handleHeaderKeyDown(event, columnKey)}
                  >
                    {sortButtons}
                    {columnKey === 'tp1' && (
                      <button
                        type="button"
                        className={styles.infoButton}
                        aria-label="Erklärung zu den TPs"
                        onClick={() => setIsBeTargetInfoOpen(true)}
                      ></button>
                    )}
                    <button
                      type="button"
                      data-drag-handle
                      className={styles.dragHandle}
                      aria-label={`Spalte verschieben: ${label}`}
                    />
                  </th>
                );
              })}
            </tr>
          </thead>
          <tbody>
            {isLoading &&
              positions.length === 0 &&
              Array.from({ length: PLACEHOLDER_ROW_COUNT }, (_, index) => (
                <tr
                  key={index}
                  className={styles.placeholderRow}
                  data-testid="position-placeholder-row"
                >
                  {order.map((columnKey) => (
                    <td key={columnKey} className={styles.placeholderCell} aria-hidden="true">
                      <span className={styles.placeholderBar} />
                    </td>
                  ))}
                </tr>
              ))}
            {sortedPositions.map((position) => (
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
        title="BE und TP1"
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
              <span className={`${styles.chip} ${styles.chipAction}`}>BE ziehen</span>
            )}
          </div>
        </div>
      </th>
    ),
    valueMargin: (
      <td data-label={COLUMN_LABELS.valueMargin} className={styles.cell}>
        <div className={styles.twoLine}>
          <span>
            {formatDecimal(position.positionValue, USDT_FORMAT)}{' '}
            <span className={styles.unit}>USDT</span>
          </span>
          <span className={styles.lineSub}>
            {formatDecimal(position.margin, USDT_FORMAT)} <span className={styles.unit}>USDT</span>
          </span>
        </div>
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
    entrySl: (
      <td data-label={COLUMN_LABELS.entrySl} className={styles.cell}>
        <div className={styles.twoLine}>
          <span>{formatDecimal(position.avgOpenPrice)}</span>
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
    be: (
      <td data-label={COLUMN_LABELS.be} className={styles.cell}>
        {secondLevel ? (
          formatDecimal(secondLevel.price)
        ) : (
          <span aria-label="Risiko unbekannt">{MISSING_VALUE}</span>
        )}
      </td>
    ),
    tp1: (
      <td data-label={COLUMN_LABELS.tp1} className={styles.cell}>
        {thirdLevel ? (
          formatDecimal(thirdLevel.price)
        ) : (
          <span aria-label="Risiko unbekannt">{MISSING_VALUE}</span>
        )}
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
