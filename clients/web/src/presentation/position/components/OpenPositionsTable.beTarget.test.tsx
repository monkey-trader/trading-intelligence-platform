import { render, screen, within } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import type { OpenPosition } from '@/domain/position/OpenPosition';
import { suiLongPosition } from '@/test/fixtures/openPositions';
import { DEFAULT_COLUMN_ORDER } from '@/presentation/position/columnOrder';
import { OpenPositionsTable } from '@/presentation/position/components/OpenPositionsTable';
import { formatDecimal } from '@/presentation/position/format/formatDecimal';

// Erwartet: OpenPosition.levels?: readonly { kind: 'ONE_R'|'TWO_R'|'THREE_R'; rMultiple: number; price: string; percent: string }[]
// Spalte 'beTarget' (Label "BE / Ziel") am Ende; Zelle: Zeile "BE <2R-Preis>" und Zeile "Ziel <3R-Preis>".

const levels = [
  { kind: 'ONE_R', rMultiple: 1, price: '1.0200', percent: '6.25' },
  { kind: 'TWO_R', rMultiple: 2, price: '1.0800', percent: '12.5' },
  { kind: 'THREE_R', rMultiple: 3, price: '1.1400', percent: '18.75' },
] as const;

function renderRow(position: OpenPosition) {
  render(<OpenPositionsTable positions={[position]} />);
  const table = screen.getByRole('table', { name: /offene positionen/i });
  return { table, row: within(table).getAllByRole('row')[1] as HTMLElement };
}

function beTargetCell(row: HTMLElement): HTMLElement {
  const cell = row.querySelector<HTMLElement>('[data-label="BE / Ziel"]');
  if (!cell) throw new Error('BE / Ziel-Zelle fehlt');
  return cell;
}

beforeEach(() => window.localStorage.clear());

describe('OpenPositionsTable Spalte BE / Ziel', () => {
  it('beTarget steht in der Standardreihenfolge am Ende', () => {
    expect(DEFAULT_COLUMN_ORDER[DEFAULT_COLUMN_ORDER.length - 1]).toBe('beTarget');
  });

  it('Header trägt das Label "BE / Ziel"', () => {
    const { table } = renderRow({ ...suiLongPosition, levels });
    expect(table.querySelector('th[data-column-key="beTarget"]')).toHaveTextContent(/^BE \/ Ziel$/);
  });

  it('zeigt 2R-Preis als BE und 3R-Preis als Ziel, formatiert wie der Einstieg', () => {
    const { row } = renderRow({ ...suiLongPosition, levels });
    const cell = beTargetCell(row);
    expect(within(cell).getByText(`BE ${formatDecimal('1.0800')}`)).toBeInTheDocument();
    expect(within(cell).getByText(`Ziel ${formatDecimal('1.1400')}`)).toBeInTheDocument();
    expect(cell.textContent).not.toContain(formatDecimal('1.0200'));
  });

  it.each([
    ['levels fehlt', undefined],
    ['levels leer', []],
  ])('%s: "–" mit aria-label "Risiko unbekannt"', (_name, value) => {
    const { row } = renderRow({ ...suiLongPosition, levels: value });
    const cell = beTargetCell(row);
    expect(cell).toHaveTextContent('–');
    expect(within(cell).getByLabelText('Risiko unbekannt')).toBeInTheDocument();
    expect(cell.textContent).not.toMatch(/BE |Ziel /);
  });
});
