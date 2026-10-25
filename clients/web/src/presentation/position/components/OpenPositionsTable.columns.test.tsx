import { fireEvent, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it } from 'vitest';
import { suiLongPosition } from '@/test/fixtures/openPositions';
import { COLUMN_ORDER_STORAGE_KEY } from '@/presentation/position/columnOrder';
import { OpenPositionsTable } from '@/presentation/position/components/OpenPositionsTable';

function setup() {
  render(<OpenPositionsTable positions={[suiLongPosition]} />);
  return screen.getByRole('table', { name: /offene positionen/i });
}

function headerTexts(table: HTMLElement): string[] {
  return within(table)
    .getAllByRole('columnheader')
    .map((header) => header.getAttribute('data-column-key') ?? '');
}

function header(table: HTMLElement, key: string): HTMLElement {
  const found = table.querySelector<HTMLElement>(`th[data-column-key="${key}"]`);
  if (!found) throw new Error(`Header ${key} fehlt`);
  return found;
}

function dragOnto(source: HTMLElement, target: HTMLElement) {
  const dataTransfer = {
    setData: () => undefined,
    getData: () => '',
    effectAllowed: '',
    dropEffect: '',
  };
  fireEvent.dragStart(source, { dataTransfer });
  fireEvent.dragOver(target, { dataTransfer });
  fireEvent.drop(target, { dataTransfer });
}

beforeEach(() => window.localStorage.clear());

describe('OpenPositionsTable Spaltenreihenfolge', () => {
  it('Header tragen data-column-key; Standardreihenfolge', () => {
    const table = setup();
    expect(headerTexts(table).slice(0, 3)).toEqual(['symbol', 'valueMargin', 'unrealizedProfit']);
  });

  it('Header außer Symbol sind draggable, Symbol nicht', () => {
    const table = setup();
    expect(header(table, 'entrySl')).toHaveAttribute('draggable', 'true');
    expect(header(table, 'symbol')).not.toHaveAttribute('draggable', 'true');
  });

  it('Drag & Drop ordnet Header und Zeilenzellen um und speichert', () => {
    const table = setup();
    dragOnto(header(table, 'entrySl'), header(table, 'valueMargin'));
    expect(headerTexts(table).slice(0, 3)).toEqual(['symbol', 'entrySl', 'valueMargin']);
    const cells = Array.from(table.querySelectorAll('tbody tr:first-child > *'));
    expect(cells[0]).toHaveTextContent('SUI');
    expect(cells[1]).toHaveAttribute('data-label', 'Einstieg / SL');
    expect(cells[2]).toHaveAttribute('data-label', 'Wert / Margin');
    expect(JSON.parse(window.localStorage.getItem(COLUMN_ORDER_STORAGE_KEY) ?? 'null')[1]).toBe(
      'entrySl',
    );
  });

  it('Symbol-Spalte bleibt erste Spalte (th scope=row) und lässt sich nicht verschieben', () => {
    const table = setup();
    dragOnto(header(table, 'symbol'), header(table, 'entrySl'));
    dragOnto(header(table, 'entrySl'), header(table, 'symbol'));
    expect(headerTexts(table)[0]).toBe('symbol');
    expect(table.querySelector('tbody tr:first-child > :first-child')).toHaveAttribute(
      'scope',
      'row',
    );
  });

  it('gespeicierte Reihenfolge wird beim Rendern angewendet', () => {
    window.localStorage.setItem(COLUMN_ORDER_STORAGE_KEY, JSON.stringify(['symbol', 'tp1']));
    const table = setup();
    expect(headerTexts(table).slice(0, 3)).toEqual(['symbol', 'tp1', 'valueMargin']);
  });

  it('Tastatur: Alt+Pfeil rechts/links am Griff verschiebt die Spalte', async () => {
    const user = userEvent.setup();
    const table = setup();
    const handle = screen.getByRole('button', { name: 'Spalte verschieben: Wert / Margin' });
    handle.focus();
    await user.keyboard('{Alt>}{ArrowRight}{/Alt}');
    expect(headerTexts(table).slice(0, 3)).toEqual(['symbol', 'unrealizedProfit', 'valueMargin']);
    await user.keyboard('{Alt>}{ArrowLeft}{/Alt}');
    expect(headerTexts(table).slice(0, 3)).toEqual(['symbol', 'valueMargin', 'unrealizedProfit']);
  });

  it('Tastatur: ArrowLeft ohne Alt verschiebt nichts; erste verschiebbare Spalte nicht vor Symbol', async () => {
    const user = userEvent.setup();
    const table = setup();
    const handle = screen.getByRole('button', { name: 'Spalte verschieben: Wert / Margin' });
    handle.focus();
    await user.keyboard('{ArrowRight}');
    await user.keyboard('{Alt>}{ArrowLeft}{/Alt}');
    expect(headerTexts(table)[0]).toBe('symbol');
    expect(headerTexts(table)[1]).toBe('valueMargin');
  });

  it('Symbol hat keinen Verschiebe-Griff', () => {
    setup();
    expect(screen.queryByRole('button', { name: 'Spalte verschieben: Symbol' })).toBeNull();
  });

  it('„Spalten zurücksetzen“ nur bei abweichender Reihenfolge; stellt Standard her', async () => {
    const user = userEvent.setup();
    const table = setup();
    expect(screen.queryByRole('button', { name: 'Spalten zurücksetzen' })).toBeNull();
    dragOnto(header(table, 'entrySl'), header(table, 'valueMargin'));
    await user.click(screen.getByRole('button', { name: 'Spalten zurücksetzen' }));
    expect(headerTexts(table).slice(0, 3)).toEqual(['symbol', 'valueMargin', 'unrealizedProfit']);
    expect(window.localStorage.getItem(COLUMN_ORDER_STORAGE_KEY)).toBeNull();
    expect(screen.queryByRole('button', { name: 'Spalten zurücksetzen' })).toBeNull();
  });

  it('Kartenansicht: jede Zelle behält data-label in neuer Reihenfolge', () => {
    const table = setup();
    dragOnto(header(table, 'tp1'), header(table, 'valueMargin'));
    const labels = Array.from(table.querySelectorAll('tbody tr:first-child > *')).map((cell) =>
      cell.getAttribute('data-label'),
    );
    expect(labels.slice(0, 3)).toEqual(['Symbol', 'TP1 (3R)', 'Wert / Margin']);
  });
});
