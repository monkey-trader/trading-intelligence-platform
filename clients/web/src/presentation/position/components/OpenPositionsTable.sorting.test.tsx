import { fireEvent, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { OpenPosition } from '@/domain/position/OpenPosition';
import { suiLongPosition } from '@/test/fixtures/openPositions';
import { OpenPositionsTable } from '@/presentation/position/components/OpenPositionsTable';
import { formatDecimal } from '@/presentation/position/format/formatDecimal';

// Vertrag (TASK-7): Spalten-Schlüssel symbol, valueMargin, unrealizedProfit, realisedProfit,
// entrySl, be, tp1. Props: isLoading?: boolean. Platzhalterzeilen: data-testid="position-placeholder-row".
// Sortier-Knöpfe heißen "<Wert> sortieren: …", Auswahl mobil "Sortieren", Knopf "Richtung wechseln",
// Ansage per role="status" in der Tabellenkomponente.

function position(positionId: string, overrides: Partial<OpenPosition> = {}): OpenPosition {
  return { ...suiLongPosition, positionId, initialStopLoss: '0.80', ...overrides };
}

const levels = (be: string, tp1: string): OpenPosition['levels'] => [
  { kind: 'TWO_R', rMultiple: 2, price: be, percent: '1' },
  { kind: 'THREE_R', rMultiple: 3, price: tp1, percent: '1' },
];

const rows = [
  position('a', {
    symbol: 'BBB-USDT',
    margin: '30',
    positionValue: '300',
    levels: levels('2.5', '3'),
  }),
  position('b', {
    symbol: 'CCC-USDT',
    margin: '10',
    positionValue: '1000',
    levels: levels('12', '13'),
  }),
  position('c', {
    symbol: 'AAA-USDT',
    margin: '200',
    positionValue: '90',
    levels: levels('0.7', '0.8'),
  }),
  position('d', { symbol: 'DDD-USDT', margin: '1', positionValue: '5', initialStopLoss: null }),
];

function renderTable(
  positions: readonly OpenPosition[] = rows,
  props: Partial<React.ComponentProps<typeof OpenPositionsTable>> = {},
) {
  render(<OpenPositionsTable positions={positions} {...props} />);
  return screen.getByRole('table', { name: /offene positionen/i });
}

function order(table: HTMLElement): string[] {
  const [, ...bodyRows] = within(table).getAllByRole('row');
  return bodyRows.map((row) => row.querySelector('th')?.textContent?.slice(0, 3) ?? '');
}

function th(table: HTMLElement, key: string): HTMLElement {
  const found = table.querySelector<HTMLElement>(`th[data-column-key="${key}"]`);
  if (!found) throw new Error(`Header ${key} fehlt`);
  return found;
}

const sortButton = (name: RegExp) => screen.getByRole('button', { name });

beforeEach(() => window.localStorage.clear());

describe('OpenPositionsTable Spalten (AC2)', () => {
  it('zeigt Symbol, Wert / Margin, Unreal. PnL, Real. PnL, Einstieg / SL, BE (2R), TP1 (3R)', () => {
    const table = renderTable([rows[0] as OpenPosition]);
    const keys = within(table)
      .getAllByRole('columnheader')
      .map((header) => header.getAttribute('data-column-key'));
    expect(keys).toEqual([
      'symbol',
      'valueMargin',
      'unrealizedProfit',
      'realisedProfit',
      'entrySl',
      'be',
      'tp1',
    ]);
    const labels = Array.from(table.querySelectorAll('tbody tr > *')).map((cell) =>
      cell.getAttribute('data-label'),
    );
    expect(labels).toEqual([
      'Symbol',
      'Wert / Margin',
      'Unreal. PnL',
      'Real. PnL',
      'Einstieg / SL',
      'BE (2R)',
      'TP1 (3R)',
    ]);
  });

  it('BE und TP1 sind reine Zahlen ohne Präfix', () => {
    const table = renderTable([rows[0] as OpenPosition]);
    expect(table.querySelector('[data-label="BE (2R)"]')?.textContent).toBe(formatDecimal('2.5'));
    expect(table.querySelector('[data-label="TP1 (3R)"]')?.textContent).toBe(formatDecimal('3'));
  });

  it('BE und TP1 zeigen "–", wenn das Risiko unbekannt ist', () => {
    const table = renderTable([rows[3] as OpenPosition]);
    expect(table.querySelector('[data-label="BE (2R)"]')).toHaveTextContent('–');
    expect(table.querySelector('[data-label="TP1 (3R)"]')).toHaveTextContent('–');
  });

  it('Wert / Margin und Einstieg / SL stapeln zwei Werte', () => {
    const table = renderTable([rows[0] as OpenPosition]);
    const valueMargin = table.querySelector('[data-label="Wert / Margin"]');
    expect(valueMargin).toHaveTextContent(
      formatDecimal('300', { minimumFractionDigits: 2, maximumFractionDigits: 2 }),
    );
    expect(valueMargin).toHaveTextContent(
      formatDecimal('30', { minimumFractionDigits: 2, maximumFractionDigits: 2 }),
    );
    const entrySl = table.querySelector('[data-label="Einstieg / SL"]');
    expect(entrySl).toHaveTextContent(formatDecimal(suiLongPosition.avgOpenPrice));
    expect(entrySl).toHaveTextContent(/SL\s*0\.9/);
  });
});

describe('OpenPositionsTable Sortieren per Klick (AC3, AC4)', () => {
  it('Klick sortiert aufsteigend, dann absteigend, dann zurück zur API-Reihenfolge', async () => {
    const user = userEvent.setup();
    const table = renderTable();
    expect(order(table)).toEqual(['BBB', 'CCC', 'AAA', 'DDD']);
    await user.click(sortButton(/^Symbol sortieren/));
    expect(order(table)).toEqual(['AAA', 'BBB', 'CCC', 'DDD']);
    await user.click(sortButton(/^Symbol sortieren/));
    expect(order(table)).toEqual(['CCC', 'BBB', 'AAA', 'DDD']);
    await user.click(sortButton(/^Symbol sortieren/));
    expect(order(table)).toEqual(['BBB', 'CCC', 'AAA', 'DDD']);
  });

  it('aria-sort und Pfeil zeigen die aktive Spalte, andere stehen auf none', async () => {
    const user = userEvent.setup();
    const table = renderTable();
    expect(th(table, 'symbol')).toHaveAttribute('aria-sort', 'none');
    await user.click(sortButton(/^Symbol sortieren/));
    expect(th(table, 'symbol')).toHaveAttribute('aria-sort', 'ascending');
    expect(th(table, 'symbol')).toHaveTextContent('↑');
    expect(th(table, 'unrealizedProfit')).toHaveAttribute('aria-sort', 'none');
    await user.click(sortButton(/^Symbol sortieren/));
    expect(th(table, 'symbol')).toHaveAttribute('aria-sort', 'descending');
    expect(th(table, 'symbol')).toHaveTextContent('↓');
  });

  it('gestapelte Spalten haben je einen Sortier-Knopf pro Wert', async () => {
    const user = userEvent.setup();
    const table = renderTable();
    const valueMargin = within(th(table, 'valueMargin'));
    expect(valueMargin.getAllByRole('button', { name: /sortieren/ })).toHaveLength(2);
    await user.click(valueMargin.getByRole('button', { name: /^Wert sortieren/ }));
    expect(order(table)).toEqual(['AAA', 'BBB', 'CCC', 'DDD']);
    await user.click(valueMargin.getByRole('button', { name: /^Margin sortieren/ }));
    expect(order(table)).toEqual(['CCC', 'BBB', 'AAA', 'DDD']);
    expect(th(table, 'valueMargin')).toHaveAttribute('aria-sort', 'ascending');
    const entrySl = within(th(table, 'entrySl'));
    expect(entrySl.getByRole('button', { name: /^Einstieg sortieren/ })).toBeInTheDocument();
    expect(entrySl.getByRole('button', { name: /^SL sortieren/ })).toBeInTheDocument();
  });

  it('Zeilen ohne SL bleiben auch absteigend am Ende (AC11)', async () => {
    const user = userEvent.setup();
    const table = renderTable();
    await user.click(sortButton(/^Unreal\. PnL sortieren/));
    await user.click(sortButton(/^Unreal\. PnL sortieren/));
    expect(order(table).at(-1)).toBe('DDD');
  });
});

describe('OpenPositionsTable Tastatur (AC12)', () => {
  it('Enter und Leertaste auf dem Kopf-Knopf sortieren mit demselben Zyklus', async () => {
    const user = userEvent.setup();
    const table = renderTable();
    sortButton(/^Symbol sortieren/).focus();
    await user.keyboard('{Enter}');
    expect(order(table)).toEqual(['AAA', 'BBB', 'CCC', 'DDD']);
    await user.keyboard(' ');
    expect(order(table)).toEqual(['CCC', 'BBB', 'AAA', 'DDD']);
    await user.keyboard('{Enter}');
    expect(order(table)).toEqual(['BBB', 'CCC', 'AAA', 'DDD']);
  });

  it('Alt+Pfeil rechts verschiebt die Spalte, Fokus bleibt, Änderung wird angesagt', async () => {
    const user = userEvent.setup();
    const table = renderTable();
    const handle = screen.getByRole('button', { name: 'Spalte verschieben: BE (2R)' });
    handle.focus();
    await user.keyboard('{Alt>}{ArrowRight}{/Alt}');
    const keys = within(table)
      .getAllByRole('columnheader')
      .map((header) => header.getAttribute('data-column-key'));
    expect(keys.slice(-2)).toEqual(['tp1', 'be']);
    expect(screen.getByRole('button', { name: 'Spalte verschieben: BE (2R)' })).toHaveFocus();
    expect(screen.getByRole('status')).toHaveTextContent(/BE \(2R\)/);
  });

  it('Alt+Pfeil links schiebt die erste verschiebbare Spalte nicht vor Symbol', async () => {
    const user = userEvent.setup();
    const table = renderTable();
    screen.getByRole('button', { name: 'Spalte verschieben: Wert / Margin' }).focus();
    await user.keyboard('{Alt>}{ArrowLeft}{/Alt}');
    expect(th(table, 'symbol')).toBe(within(table).getAllByRole('columnheader')[0]);
  });
});

describe('OpenPositionsTable Symbol fix (AC5)', () => {
  it('Symbol ist nicht ziehbar, andere Köpfe schon; Symbol hat keinen Verschiebe-Griff', () => {
    const table = renderTable();
    expect(th(table, 'symbol')).not.toHaveAttribute('draggable', 'true');
    expect(th(table, 'tp1')).toHaveAttribute('draggable', 'true');
    expect(screen.queryByRole('button', { name: 'Spalte verschieben: Symbol' })).toBeNull();
  });

  it('Drag verschiebt die Spalte und ändert die Sortierung nicht', async () => {
    const user = userEvent.setup();
    const table = renderTable();
    await user.click(sortButton(/^Symbol sortieren/));
    const dataTransfer = {
      setData: () => undefined,
      getData: () => '',
      effectAllowed: '',
      dropEffect: '',
    };
    fireEvent.dragStart(th(table, 'tp1'), { dataTransfer });
    fireEvent.drop(th(table, 'be'), { dataTransfer });
    expect(order(table)).toEqual(['AAA', 'BBB', 'CCC', 'DDD']);
    const keys = within(table)
      .getAllByRole('columnheader')
      .map((h) => h.getAttribute('data-column-key'));
    expect(keys.slice(-2)).toEqual(['tp1', 'be']);
  });
});

describe('OpenPositionsTable leer und lädt (AC6, AC7)', () => {
  it('leer: Sortier-Knöpfe sind nicht klickbar', () => {
    renderTable([]);
    const buttons = screen.getAllByRole('button', { name: /sortieren/ });
    expect(buttons.length).toBeGreaterThan(0);
    buttons.forEach((button) => expect(button).toBeDisabled());
  });

  it('lädt: Sortier-Knöpfe gesperrt und Platzhalterzeilen sichtbar', () => {
    renderTable([], { isLoading: true });
    screen.getAllByRole('button', { name: /sortieren/ }).forEach((b) => expect(b).toBeDisabled());
    expect(screen.getAllByTestId('position-placeholder-row').length).toBeGreaterThan(0);
  });

  it('mit Daten gibt es keine Platzhalterzeilen', () => {
    renderTable();
    expect(screen.queryByTestId('position-placeholder-row')).toBeNull();
  });
});

describe('OpenPositionsTable Auswahl bleibt an positionId (AC13)', () => {
  it('Sortieren behält die gewählte Position und ihr Chart-Knopf liefert dieselbe ID', async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    const table = renderTable(rows, { selectedPositionId: 'c', onSelectPosition: onSelect });
    await user.click(sortButton(/^Symbol sortieren/));
    const expanded = within(table).getAllByRole('button', { name: /^Chart/, expanded: true });
    expect(expanded).toHaveLength(1);
    expect(expanded[0]).toHaveAccessibleName(/AAA/);
    await user.click(expanded[0] as HTMLElement);
    expect(onSelect).toHaveBeenCalledWith('c');
  });

  it('Spalten verschieben behält die Auswahl', async () => {
    const user = userEvent.setup();
    renderTable(rows, { selectedPositionId: 'c' });
    screen.getByRole('button', { name: 'Spalte verschieben: BE (2R)' }).focus();
    await user.keyboard('{Alt>}{ArrowRight}{/Alt}');
    expect(screen.getAllByRole('button', { name: /^Chart/, expanded: true })).toHaveLength(1);
  });
});

describe('OpenPositionsTable mobile Sortierung (AC8)', () => {
  it('Auswahl sortiert aufsteigend, Richtungsknopf wechselt, "Reihenfolge der API" hebt auf', async () => {
    const user = userEvent.setup();
    const table = renderTable();
    const select = screen.getByLabelText('Sortieren');
    const direction = screen.getByRole('button', { name: 'Richtung wechseln' });
    expect(direction).toBeDisabled();
    await user.selectOptions(select, 'Symbol');
    expect(order(table)).toEqual(['AAA', 'BBB', 'CCC', 'DDD']);
    await user.click(direction);
    expect(order(table)).toEqual(['CCC', 'BBB', 'AAA', 'DDD']);
    await user.selectOptions(select, 'Reihenfolge der API');
    expect(order(table)).toEqual(['BBB', 'CCC', 'AAA', 'DDD']);
    expect(direction).toBeDisabled();
  });

  it('bietet einen Eintrag pro sortierbarem Wert', () => {
    renderTable();
    const labels = within(screen.getByLabelText('Sortieren'))
      .getAllByRole('option')
      .map((option) => option.textContent);
    expect(labels).toEqual([
      'Reihenfolge der API',
      'Symbol',
      'Wert',
      'Margin',
      'Unreal. PnL',
      'Real. PnL',
      'Einstieg',
      'SL',
      'BE (2R)',
      'TP1 (3R)',
    ]);
  });
});

describe('OpenPositionsTable Review-Korrekturen (AC12, AC6)', () => {
  const keysOf = (table: HTMLElement) =>
    within(table)
      .getAllByRole('columnheader')
      .map((header) => header.getAttribute('data-column-key'));

  it('Alt+Pfeil rechts auf dem Sortier-Knopf verschiebt die Spalte, Fokus bleibt auf dem Kopf', async () => {
    const user = userEvent.setup();
    const table = renderTable();
    sortButton(/^BE.*sortieren/).focus();
    await user.keyboard('{Alt>}{ArrowRight}{/Alt}');
    expect(keysOf(table).slice(-2)).toEqual(['tp1', 'be']);
    expect(th(table, 'be').contains(document.activeElement)).toBe(true);
  });

  it('Alt+Pfeil links auf dem Sortier-Knopf verschiebt die Spalte nach links', async () => {
    const user = userEvent.setup();
    const table = renderTable();
    sortButton(/^BE.*sortieren/).focus();
    await user.keyboard('{Alt>}{ArrowLeft}{/Alt}');
    expect(keysOf(table).slice(-3)).toEqual(['be', 'entrySl', 'tp1']);
    expect(th(table, 'be').contains(document.activeElement)).toBe(true);
  });

  it('Alt+Pfeil links auf dem Sortier-Knopf von Symbol bewegt nichts', async () => {
    const user = userEvent.setup();
    const table = renderTable();
    const before = keysOf(table);
    sortButton(/^Symbol sortieren/).focus();
    await user.keyboard('{Alt>}{ArrowLeft}{/Alt}');
    await user.keyboard('{Alt>}{ArrowRight}{/Alt}');
    expect(keysOf(table)).toEqual(before);
    expect(keysOf(table)[0]).toBe('symbol');
  });

  it('Ansage nennt Spaltenname und neue Stelle, zwei Verschiebungen ergeben verschiedene Texte', async () => {
    const user = userEvent.setup();
    renderTable();
    screen.getByRole('button', { name: 'Spalte verschieben: BE (2R)' }).focus();
    await user.keyboard('{Alt>}{ArrowLeft}{/Alt}');
    const first = screen.getByRole('status').textContent ?? '';
    expect(first).toMatch(/BE \(2R\)/);
    expect(first).toMatch(/Stelle/);
    expect(first).toMatch(/\b6\b/);
    expect(first).toMatch(/\b7\b/);
    await user.keyboard('{Alt>}{ArrowLeft}{/Alt}');
    const second = screen.getByRole('status').textContent ?? '';
    expect(second).toMatch(/\b5\b/);
    expect(second).not.toBe(first);
  });

  it('leere Liste behält die gewählte Sortierung, Knöpfe sind gesperrt', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<OpenPositionsTable positions={rows} />);
    const table = screen.getByRole('table', { name: /offene positionen/i });
    await user.click(sortButton(/^Symbol sortieren/));
    rerender(<OpenPositionsTable positions={[]} />);
    expect(th(table, 'symbol')).toHaveAttribute('aria-sort', 'ascending');
    expect(th(table, 'symbol')).toHaveTextContent('↑');
    screen.getAllByRole('button', { name: /sortieren/ }).forEach((b) => expect(b).toBeDisabled());
  });
});
