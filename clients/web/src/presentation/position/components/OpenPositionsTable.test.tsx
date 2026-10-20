import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { OpenPosition } from '@/domain/position/OpenPosition';
import {
  breakEvenPosition,
  goldShortPosition,
  noStopLossPosition,
  unknownOrdersPosition,
  positionWithoutLiquidation,
  suiLongPosition,
} from '@/test/fixtures/openPositions';
import { formatDateTime } from '@/presentation/position/format/formatTimestamp';
import { OpenPositionsTable } from '@/presentation/position/components/OpenPositionsTable';

const COLUMN_LABELS = [
  'Symbol',
  'Seite',
  'Menge',
  'Einstieg',
  'Mark',
  'Hebel',
  'Margin',
  'Unreal. PnL',
  'PnL %',
  'Real. PnL',
  'Liq.-Preis',
  'Liq.-Abstand',
  'SL',
  'TP',
  'Eröffnet',
] as const;

type ColumnLabel = (typeof COLUMN_LABELS)[number];

function renderTable(positions: readonly OpenPosition[]) {
  render(<OpenPositionsTable positions={positions} />);
  return screen.getByRole('table', { name: /offene positionen/i });
}

function bodyRows(table: HTMLElement): HTMLElement[] {
  const [, ...rows] = within(table).getAllByRole('row');
  return rows;
}

/** Datenzellen einer Zeile; die Symbol-Zelle darf auch ein Zeilenkopf (th) sein. */
function cellsOf(row: HTMLElement): HTMLElement[] {
  return Array.from(row.querySelectorAll<HTMLElement>('td, th'));
}

function cellOf(row: HTMLElement, columnLabel: ColumnLabel): HTMLElement {
  const cell = cellsOf(row)[COLUMN_LABELS.indexOf(columnLabel)];
  if (!cell) throw new Error(`Spalte ${columnLabel} fehlt`);
  return cell;
}

function rowFor(table: HTMLElement, symbol: string): HTMLElement {
  const row = bodyRows(table).find((candidate) => within(candidate).queryByText(symbol));
  if (!row) throw new Error(`Zeile für ${symbol} fehlt`);
  return row;
}

function hasTone(cell: HTMLElement, tone: 'positive' | 'negative' | 'neutral'): boolean {
  return cell.classList.contains(tone) || cell.querySelector(`.${tone}`) !== null;
}

describe('OpenPositionsTable', () => {
  it('ist eine semantische Tabelle mit den Spalten in fester Reihenfolge', () => {
    const table = renderTable([suiLongPosition]);

    const headers = within(table)
      .getAllByRole('columnheader')
      .map((header) => header.textContent?.trim());
    expect(headers).toEqual([...COLUMN_LABELS]);
  });

  it('zeigt eine Zeile pro Position in der gelieferten Reihenfolge', () => {
    const table = renderTable([goldShortPosition, suiLongPosition]);

    const rows = bodyRows(table);
    expect(rows).toHaveLength(2);
    expect(within(rows[0] as HTMLElement).getByText('NCCOGOLD2USD-USDT')).toBeInTheDocument();
    expect(within(rows[1] as HTMLElement).getByText('SUI-USDT')).toBeInTheDocument();
  });

  it('versieht jede Zelle mit data-label für die mobile Kartenansicht', () => {
    const table = renderTable([goldShortPosition, suiLongPosition]);

    for (const row of bodyRows(table)) {
      expect(cellsOf(row).map((cell) => cell.getAttribute('data-label'))).toEqual([
        ...COLUMN_LABELS,
      ]);
    }
  });

  it('formatiert die Werte einer LONG-Position', () => {
    const table = renderTable([suiLongPosition]);
    const row = rowFor(table, 'SUI-USDT');

    expect(cellOf(row, 'Seite')).toHaveTextContent(/long/i);
    expect(cellOf(row, 'Menge')).toHaveTextContent(/^500$/);
    expect(cellOf(row, 'Einstieg')).toHaveTextContent(/^0\.96$/);
    expect(cellOf(row, 'Mark')).toHaveTextContent(/^1\.01$/);
    expect(cellOf(row, 'Hebel')).toHaveTextContent(/^10\s?[x×]$/);
    expect(cellOf(row, 'Margin')).toHaveTextContent(/^60\.50$/);
    expect(cellOf(row, 'Unreal. PnL')).toHaveTextContent(/^\+25\.00$/);
    expect(cellOf(row, 'PnL %')).toHaveTextContent(/^\+52\.08\s?%$/);
    expect(cellOf(row, 'Real. PnL')).toHaveTextContent(/^[-−]0\.24$/);
    expect(cellOf(row, 'Liq.-Preis')).toHaveTextContent(/^0\.865$/);
    expect(cellOf(row, 'Liq.-Abstand')).toHaveTextContent(/^14\.36\s?%$/);
    expect(cellOf(row, 'Eröffnet')).toHaveTextContent(formatDateTime(suiLongPosition.openTime));
  });

  it('formatiert die Werte einer SHORT-Position mit Tausendertrennung und Minus', () => {
    const table = renderTable([goldShortPosition]);
    const row = rowFor(table, 'NCCOGOLD2USD-USDT');

    expect(cellOf(row, 'Seite')).toHaveTextContent(/short/i);
    expect(cellOf(row, 'Menge')).toHaveTextContent(/^0\.25$/);
    expect(cellOf(row, 'Einstieg')).toHaveTextContent(/^4['’]150$/);
    expect(cellOf(row, 'Mark')).toHaveTextContent(/^4['’]175$/);
    expect(cellOf(row, 'Hebel')).toHaveTextContent(/^15\s?[x×]$/);
    expect(cellOf(row, 'Unreal. PnL')).toHaveTextContent(/^[-−]6\.25$/);
    expect(cellOf(row, 'PnL %')).toHaveTextContent(/^[-−]9\.04\s?%$/);
    expect(cellOf(row, 'Real. PnL')).toHaveTextContent(/^[-−]0\.52$/);
    expect(cellOf(row, 'Liq.-Preis')).toHaveTextContent(/^4['’]410\.5$/);
    expect(cellOf(row, 'Liq.-Abstand')).toHaveTextContent(/^5\.64\s?%$/);
    expect(cellOf(row, 'Eröffnet')).toHaveTextContent(formatDateTime(goldShortPosition.openTime));
  });

  it('tönt die PnL-Zellen nach dem Vorzeichen', () => {
    const table = renderTable([goldShortPosition, suiLongPosition, positionWithoutLiquidation]);
    const winner = rowFor(table, 'SUI-USDT');
    const loser = rowFor(table, 'NCCOGOLD2USD-USDT');
    const flat = rowFor(table, 'BTC-USDT');

    expect(hasTone(cellOf(winner, 'Unreal. PnL'), 'positive')).toBe(true);
    expect(hasTone(cellOf(winner, 'PnL %'), 'positive')).toBe(true);
    expect(hasTone(cellOf(loser, 'Unreal. PnL'), 'negative')).toBe(true);
    expect(hasTone(cellOf(loser, 'PnL %'), 'negative')).toBe(true);
    expect(hasTone(cellOf(flat, 'Unreal. PnL'), 'neutral')).toBe(true);
    expect(hasTone(cellOf(flat, 'Unreal. PnL'), 'positive')).toBe(false);
    expect(hasTone(cellOf(flat, 'Unreal. PnL'), 'negative')).toBe(false);
  });

  it('zeigt positive PnL-Werte mit explizitem "+", Preise und Liq.-Abstand ohne "+"', () => {
    const profitablePosition: OpenPosition = {
      ...suiLongPosition,
      positionId: '9100000000000000004',
      symbol: 'ETH-USDT',
      positionAmount: '2',
      avgOpenPrice: '2500.5',
      markPrice: '3117.75',
      unrealizedProfit: '1234.5',
      realisedProfit: '12.3456',
      pnlRatio: '0.12345',
      liquidationPrice: '2100.25',
      liquidationDistanceRatio: '0.32634',
    };
    const table = renderTable([profitablePosition]);
    const row = rowFor(table, 'ETH-USDT');

    expect(cellOf(row, 'Unreal. PnL')).toHaveTextContent(/^\+1['’]234\.50$/);
    expect(cellOf(row, 'Real. PnL')).toHaveTextContent(/^\+12\.35$/);
    expect(cellOf(row, 'PnL %')).toHaveTextContent(/^\+12\.35\s?%$/);
    expect(cellOf(row, 'Menge')).toHaveTextContent(/^2$/);
    expect(cellOf(row, 'Einstieg')).toHaveTextContent(/^2['’]500\.5$/);
    expect(cellOf(row, 'Mark')).toHaveTextContent(/^3['’]117\.75$/);
    expect(cellOf(row, 'Liq.-Preis')).toHaveTextContent(/^2['’]100\.25$/);
    expect(cellOf(row, 'Liq.-Abstand')).toHaveTextContent(/^32\.63\s?%$/);
  });

  it('zeigt PnL-Werte, die gerundet 0 sind, ohne Vorzeichen', () => {
    const nearlyFlatPosition: OpenPosition = {
      ...positionWithoutLiquidation,
      unrealizedProfit: '0.004',
      realisedProfit: '-0.004',
      pnlRatio: '-0.00004',
    };
    const table = renderTable([nearlyFlatPosition]);
    const row = rowFor(table, 'BTC-USDT');

    expect(cellOf(row, 'Unreal. PnL')).toHaveTextContent(/^0\.00$/);
    expect(cellOf(row, 'Real. PnL')).toHaveTextContent(/^0\.00$/);
    expect(cellOf(row, 'PnL %')).toHaveTextContent(/^0\.00\s?%$/);
    expect(hasTone(cellOf(row, 'Real. PnL'), 'neutral')).toBe(true);
  });

  it('zeigt bei PnL genau 0 kein Vorzeichen', () => {
    const table = renderTable([positionWithoutLiquidation]);
    const row = rowFor(table, 'BTC-USDT');

    expect(cellOf(row, 'Unreal. PnL')).toHaveTextContent(/^0\.00$/);
    expect(cellOf(row, 'Real. PnL')).toHaveTextContent(/^0\.00$/);
    expect(cellOf(row, 'PnL %')).toHaveTextContent(/^0\.00\s?%$/);
  });

  it('tönt den realisierten PnL nach dem gerundeten Vorzeichen', () => {
    const realisedWinner: OpenPosition = { ...suiLongPosition, realisedProfit: '3.5' };
    const table = renderTable([realisedWinner, goldShortPosition]);

    expect(hasTone(cellOf(rowFor(table, 'SUI-USDT'), 'Real. PnL'), 'positive')).toBe(true);
    expect(hasTone(cellOf(rowFor(table, 'NCCOGOLD2USD-USDT'), 'Real. PnL'), 'negative')).toBe(true);
  });

  it('zeigt einen Gedankenstrich ohne Liquidationspreis und ohne Liq.-Abstand', () => {
    const table = renderTable([positionWithoutLiquidation]);
    const row = rowFor(table, 'BTC-USDT');

    expect(cellOf(row, 'Liq.-Abstand')).toHaveTextContent(/^–$/);
    expect(cellOf(row, 'Liq.-Preis')).toHaveTextContent(/^–$/);
  });
});

describe('OpenPositionsTable: Chart-Auswahl', () => {
  it('hat pro Zeile einen Chart-Button mit Symbol im Namen, ohne die Spalten zu ändern', () => {
    const table = renderTable([suiLongPosition, goldShortPosition]);

    expect(screen.getByRole('button', { name: /chart.*SUI-USDT/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /chart.*NCCOGOLD2USD-USDT/i })).toBeInTheDocument();
    expect(within(table).getAllByRole('columnheader')).toHaveLength(COLUMN_LABELS.length);
  });

  it('meldet die positionId beim Klick auf den Chart-Button', async () => {
    const user = userEvent.setup();
    const onSelectPosition = vi.fn();
    render(
      <OpenPositionsTable
        positions={[suiLongPosition, goldShortPosition]}
        onSelectPosition={onSelectPosition}
      />,
    );

    await user.click(screen.getByRole('button', { name: /chart.*NCCOGOLD2USD-USDT/i }));

    expect(onSelectPosition).toHaveBeenCalledWith(goldShortPosition.positionId);
  });

  it('markiert die gewählte Zeile per aria-expanded', () => {
    render(
      <OpenPositionsTable
        positions={[suiLongPosition, goldShortPosition]}
        selectedPositionId={suiLongPosition.positionId}
        onSelectPosition={() => {}}
      />,
    );

    expect(screen.getByRole('button', { name: /chart.*SUI-USDT/i })).toHaveAttribute(
      'aria-expanded',
      'true',
    );
    expect(screen.getByRole('button', { name: /chart.*NCCOGOLD2USD-USDT/i })).toHaveAttribute(
      'aria-expanded',
      'false',
    );
  });

  describe('SL, TP und Break-even', () => {
    it('zeigt SL und TP als Preis', () => {
      const row = rowFor(renderTable([goldShortPosition]), 'NCCOGOLD2USD-USDT');

      expect(cellOf(row, 'SL')).toHaveTextContent(/^4['’]200$/);
      expect(cellOf(row, 'TP')).toHaveTextContent(/^4['’]100$/);
    });

    it('zeigt – bei SL und TP null', () => {
      const row = rowFor(renderTable([unknownOrdersPosition]), 'SUI-USDT');

      expect(cellOf(row, 'SL')).toHaveTextContent(/^[–-]$/);
      expect(cellOf(row, 'TP')).toHaveTextContent(/^[–-]$/);
    });

    it('zeigt das Badge BE, wenn breakEven true ist', () => {
      const row = rowFor(renderTable([breakEvenPosition]), 'SUI-USDT');

      const badge = within(row).getByLabelText(/break-even/i);
      expect(badge).toHaveTextContent('BE');
    });

    it('zeigt kein BE-Badge bei breakEven false', () => {
      const row = rowFor(renderTable([goldShortPosition]), 'NCCOGOLD2USD-USDT');

      expect(within(row).queryByLabelText(/break-even/i)).not.toBeInTheDocument();
      expect(within(row).queryByText('BE')).not.toBeInTheDocument();
    });

    it('zeigt den Hinweis "kein SL", wenn Orders bekannt sind und kein SL existiert', () => {
      const row = rowFor(renderTable([noStopLossPosition]), 'SUI-USDT');

      expect(within(row).getByText('kein SL')).toBeInTheDocument();
      expect(within(row).queryByLabelText(/break-even/i)).not.toBeInTheDocument();
    });

    it('zeigt bei vorhandenem SL keinen Hinweis "kein SL"', () => {
      const row = rowFor(renderTable([goldShortPosition]), 'NCCOGOLD2USD-USDT');

      expect(within(row).queryByText('kein SL')).not.toBeInTheDocument();
    });

    it('zeigt bei breakEven null (Orders unbekannt) weder Badge noch "kein SL"', () => {
      const row = rowFor(renderTable([unknownOrdersPosition]), 'SUI-USDT');

      expect(within(row).queryByLabelText(/break-even/i)).not.toBeInTheDocument();
      expect(within(row).queryByText('BE')).not.toBeInTheDocument();
      expect(within(row).queryByText('kein SL')).not.toBeInTheDocument();
    });
  });
});
