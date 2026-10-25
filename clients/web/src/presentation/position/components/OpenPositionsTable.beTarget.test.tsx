import { render, screen, within } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import type { OpenPosition } from '@/domain/position/OpenPosition';
import { suiLongPosition } from '@/test/fixtures/openPositions';
import { DEFAULT_COLUMN_ORDER } from '@/presentation/position/columnOrder';
import { OpenPositionsTable } from '@/presentation/position/components/OpenPositionsTable';
import { formatDecimal } from '@/presentation/position/format/formatDecimal';

// Spalten 'be' (Label "BE (2R)") und 'tp1' (Label "TP1 (3R)") am Ende; Zelle: reine Zahl ohne Präfix.

const levels = [
  { kind: 'TWO_R', rMultiple: 2, price: '1.0800', percent: '12.5' },
  { kind: 'THREE_R', rMultiple: 3, price: '1.1400', percent: '18.75' },
] as const;

function renderRow(position: OpenPosition) {
  render(<OpenPositionsTable positions={[position]} />);
  const table = screen.getByRole('table', { name: /offene positionen/i });
  return { table, row: within(table).getAllByRole('row')[1] as HTMLElement };
}

function levelCell(row: HTMLElement, label: string): HTMLElement {
  const cell = row.querySelector<HTMLElement>(`[data-label="${label}"]`);
  if (!cell) throw new Error(`${label}-Zelle fehlt`);
  return cell;
}

beforeEach(() => window.localStorage.clear());

describe('OpenPositionsTable Spalten BE und TP1', () => {
  it('be und tp1 stehen in der Standardreihenfolge am Ende', () => {
    expect(DEFAULT_COLUMN_ORDER.slice(-2)).toEqual(['be', 'tp1']);
  });

  it('Header tragen die Labels "BE (2R)" und "TP1 (3R)"', () => {
    const { table } = renderRow({ ...suiLongPosition, levels });
    expect(table.querySelector('th[data-column-key="be"]')).toHaveTextContent(/BE \(2R\)/);
    expect(table.querySelector('th[data-column-key="tp1"]')).toHaveTextContent(/TP1 \(3R\)/);
  });

  it('zeigt BE (2R) und TP1 (3R) als reine Zahl, formatiert wie der Einstieg', () => {
    const { row } = renderRow({ ...suiLongPosition, levels });
    expect(levelCell(row, 'BE (2R)')).toHaveTextContent(new RegExp(`^${formatDecimal('1.0800')}$`));
    expect(levelCell(row, 'TP1 (3R)')).toHaveTextContent(
      new RegExp(`^${formatDecimal('1.1400')}$`),
    );
    expect(row.textContent).not.toMatch(/TP[23]/);
  });

  it.each([
    ['levels fehlt', undefined],
    ['levels leer', []],
  ])('%s: "–" mit aria-label "Risiko unbekannt"', (_name, value) => {
    const { row } = renderRow({ ...suiLongPosition, levels: value });
    for (const label of ['BE (2R)', 'TP1 (3R)']) {
      const cell = levelCell(row, label);
      expect(cell).toHaveTextContent('–');
      expect(within(cell).getByLabelText('Risiko unbekannt')).toBeInTheDocument();
      expect(cell.textContent).not.toMatch(/(BE|TP\d) /);
    }
  });
});
