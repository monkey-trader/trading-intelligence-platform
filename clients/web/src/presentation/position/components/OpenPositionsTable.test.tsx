import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { OpenPosition } from '@/domain/position/OpenPosition';
import {
  goldShortPosition,
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
  'Unreal. PnL',
  'PnL %',
  'Liq.-Preis',
  'Liq.-Abstand',
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
    expect(cellOf(row, 'Unreal. PnL')).toHaveTextContent(/^\+?25\.00$/);
    expect(cellOf(row, 'PnL %')).toHaveTextContent(/^\+?52\.08\s?%$/);
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

  it('zeigt einen Gedankenstrich ohne Liquidationspreis und ohne Liq.-Abstand', () => {
    const table = renderTable([positionWithoutLiquidation]);
    const row = rowFor(table, 'BTC-USDT');

    expect(cellOf(row, 'Liq.-Abstand')).toHaveTextContent(/^–$/);
    expect(cellOf(row, 'Liq.-Preis')).toHaveTextContent(/^–$/);
  });
});
