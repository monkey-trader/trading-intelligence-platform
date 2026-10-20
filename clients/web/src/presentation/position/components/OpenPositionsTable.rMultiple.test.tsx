import { render, screen, within } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import type { OpenPosition } from '@/domain/position/OpenPosition';
import { suiLongPosition } from '@/test/fixtures/openPositions';
import { DEFAULT_COLUMN_ORDER } from '@/presentation/position/columnOrder';
import { OpenPositionsTable } from '@/presentation/position/components/OpenPositionsTable';

const withR = (overrides: Partial<OpenPosition>): OpenPosition => ({
  ...suiLongPosition,
  initialStopLoss: '0.90',
  riskDistance: '0.06',
  currentRMultiple: '1.4',
  breakEvenTriggerReached: false,
  reached2RAt: null,
  ...overrides,
});

function renderRow(position: OpenPosition) {
  render(<OpenPositionsTable positions={[position]} />);
  const table = screen.getByRole('table', { name: /offene positionen/i });
  return { table, row: within(table).getAllByRole('row')[1] as HTMLElement };
}
const rCell = (row: HTMLElement) => {
  const cell = row.querySelector<HTMLElement>('[data-label="R"]');
  if (!cell) throw new Error('R-Zelle fehlt');
  return cell;
};

beforeEach(() => window.localStorage.clear());

describe('OpenPositionsTable R-Spalte', () => {
  it('Spalte rMultiple steht direkt nach unrealizedProfit', () => {
    const i = DEFAULT_COLUMN_ORDER.indexOf('unrealizedProfit');
    expect(DEFAULT_COLUMN_ORDER[i + 1]).toBe('rMultiple');
  });

  it('Header trägt Label "R"', () => {
    const { table } = renderRow(withR({}));
    expect(table.querySelector('th[data-column-key="rMultiple"]')).toHaveTextContent(/^R$/);
  });

  it.each([
    ['1.4', '+1.4R', 'positive'],
    ['-0.5', '-0.5R', 'negative'],
  ])('zeigt %s als %s mit pnlTone-Klasse', (value, text, tone) => {
    const { row } = renderRow(withR({ currentRMultiple: value }));
    const cell = rCell(row);
    expect(cell.textContent?.replace('−', '-')).toBe(text);
    expect(cell.innerHTML).toMatch(new RegExp(tone, 'i'));
  });

  it('null: "–" mit aria-label "Risiko unbekannt"', () => {
    const { row } = renderRow(withR({ currentRMultiple: null, riskDistance: null }));
    const cell = rCell(row);
    expect(cell).toHaveTextContent('–');
    expect(within(cell).getByLabelText('Risiko unbekannt')).toBeInTheDocument();
  });
});

describe('OpenPositionsTable Aktionschip 2R', () => {
  const chipName = /2R · BE ziehen/;

  it('Chip bei breakEvenTriggerReached true und breakEven false', () => {
    const { row } = renderRow(withR({ breakEvenTriggerReached: true, breakEven: false }));
    expect(within(row).getByText(chipName)).toBeInTheDocument();
    expect(row.querySelector('[data-label="Symbol"]')).toHaveTextContent(chipName);
  });

  it('Chip auch bei breakEven null (Orders unbekannt)', () => {
    const { row } = renderRow(withR({ breakEvenTriggerReached: true, breakEven: null }));
    expect(within(row).getByText(chipName)).toBeInTheDocument();
  });

  it('kein Chip bei breakEven true; stattdessen Break-even-Badge', () => {
    const { row } = renderRow(withR({ breakEvenTriggerReached: true, breakEven: true }));
    expect(within(row).queryByText(chipName)).not.toBeInTheDocument();
    expect(within(row).getByRole('img', { name: 'Break-even' })).toBeInTheDocument();
  });

  it.each([false, null])('kein Chip bei breakEvenTriggerReached %s', (flag) => {
    const { row } = renderRow(withR({ breakEvenTriggerReached: flag, breakEven: false }));
    expect(within(row).queryByText(chipName)).not.toBeInTheDocument();
  });
});
