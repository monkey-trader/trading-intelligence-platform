import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it } from 'vitest';
import type { OpenPosition } from '@/domain/position/OpenPosition';
import type { PositionPrice } from '@/domain/price/PriceStream';
import { suiLongPosition } from '@/test/fixtures/openPositions';
import { OpenPositionsTable } from '@/presentation/position/components/OpenPositionsTable';

// Vertrag (TASK-16): neue Props prices?: Record<positionId, PositionPrice>, pricesStale?: boolean,
// pricesAnnouncement?: string (aria-live, role="status" mit aria-label "Kursstatus").
// Spalte data-column-key="price", data-label="Kurs", Sortier-Knopf "Kurs sortieren: …".

const position = (positionId: string, symbol: string): OpenPosition => ({
  ...suiLongPosition,
  positionId,
  symbol,
  initialStopLoss: '0.80',
});
const priceOf = (positionId: string, overrides: Partial<PositionPrice> = {}): PositionPrice => ({
  positionId,
  symbol: 'X-USDT',
  price: '0.0000123',
  change24hPercent: '1.3',
  distanceToBreakEvenPercent: '2.5',
  distanceToTp1Percent: '-1.2',
  ...overrides,
});
const rows = [position('a', 'AAA-USDT'), position('b', 'BBB-USDT'), position('c', 'CCC-USDT')];

type Props = React.ComponentProps<typeof OpenPositionsTable>;
const table = () => screen.getByRole('table', { name: /offene positionen/i });
const cell = (label: string, rowIndex = 0) =>
  table()
    .querySelectorAll('tbody tr')
    [rowIndex]?.querySelector(`[data-label="${label}"]`) as HTMLElement;
const order = () =>
  within(table())
    .getAllByRole('row')
    .slice(1)
    .map((row) => row.querySelector('th')?.textContent?.slice(0, 3));

beforeEach(() => window.localStorage.clear());

describe('Kurs-Spalte', () => {
  it('steht direkt nach Symbol und hat mobil data-label "Kurs"', () => {
    render(<OpenPositionsTable positions={rows} prices={{ a: priceOf('a') }} />);
    const keys = within(table())
      .getAllByRole('columnheader')
      .map((header) => header.getAttribute('data-column-key'));
    expect(keys.slice(0, 2)).toEqual(['symbol', 'price']);
    expect(cell('Kurs')).toBeInTheDocument();
  });

  it('zeigt Kurs exakt (de-CH) und 24h-Änderung mit Vorzeichen, Farbe über CSS-Klasse', () => {
    render(<OpenPositionsTable positions={rows} prices={{ a: priceOf('a') }} />);
    expect(cell('Kurs')).toHaveTextContent('0.0000123');
    expect(cell('Kurs')).toHaveTextContent(/\+1\.3\d*\s?%/);
    expect(cell('Kurs').innerHTML).not.toMatch(/style=/);
    expect(cell('Kurs').querySelector('[class*="positive"]')).not.toBeNull();
    expect(cell('Kurs').querySelector('[class*="tabular"], [class*="numeric"]')).not.toBeNull();
  });

  it('negative 24h-Änderung hat Minus und negative Klasse', () => {
    render(
      <OpenPositionsTable
        positions={rows}
        prices={{ a: priceOf('a', { change24hPercent: '-0.5' }) }}
      />,
    );
    expect(cell('Kurs')).toHaveTextContent(/[-−]0\.5/);
    expect(cell('Kurs').querySelector('[class*="negative"]')).not.toBeNull();
  });

  it('unbekannter Kurs zeigt "–"', () => {
    render(<OpenPositionsTable positions={rows} prices={{}} />);
    expect(cell('Kurs')).toHaveTextContent('–');
  });
});

describe('Abstand in BE-/TP1-Zellen', () => {
  it('BE zeigt Abstand mit + und "offen", TP1 mit Minus und "erreicht"', () => {
    render(<OpenPositionsTable positions={rows} prices={{ a: priceOf('a') }} />);
    expect(cell('BE (2R)')).toHaveTextContent(/\+2\.5\d*\s?%/);
    expect(cell('BE (2R)')).toHaveTextContent('offen');
    expect(cell('TP1 (3R)')).toHaveTextContent(/[-−]1\.2\d*\s?%/);
    expect(cell('TP1 (3R)')).toHaveTextContent('erreicht');
  });

  it('Abstand null zeigt keinen Abstand, das Level bleibt sichtbar', () => {
    render(
      <OpenPositionsTable
        positions={rows}
        prices={{ a: priceOf('a', { distanceToBreakEvenPercent: null }) }}
      />,
    );
    expect(cell('BE (2R)')).not.toHaveTextContent(/offen|erreicht/);
  });

  it('Abstand ist nicht einzeln sortierbar', () => {
    render(<OpenPositionsTable positions={rows} prices={{ a: priceOf('a') }} />);
    expect(screen.queryByRole('button', { name: /abstand.*sortieren/i })).toBeNull();
  });
});

describe('veraltet', () => {
  it('zeigt Banner und je Kurs-Zelle den Text "veraltet", Wert bleibt', () => {
    render(<OpenPositionsTable positions={rows} prices={{ a: priceOf('a') }} pricesStale />);
    expect(screen.getByText(/kurse veraltet/i)).toBeInTheDocument();
    expect(cell('Kurs')).toHaveTextContent('veraltet');
    expect(cell('Kurs')).toHaveTextContent('0.0000123');
  });

  it('ohne stale kein Banner und kein "veraltet"', () => {
    render(<OpenPositionsTable positions={rows} prices={{ a: priceOf('a') }} />);
    expect(screen.queryByText(/veraltet/i)).toBeNull();
  });

  it('aria-live-Region ändert sich bei Ticks nicht, nur beim Wechsel', () => {
    const base: Props = { positions: rows, prices: { a: priceOf('a') } };
    const { rerender } = render(<OpenPositionsTable {...base} pricesAnnouncement="Kurse live" />);
    const region = screen.getByRole('status', { name: /kursstatus/i });
    expect(region).toHaveTextContent('Kurse live');
    rerender(
      <OpenPositionsTable
        {...base}
        prices={{ a: priceOf('a', { price: '9' }) }}
        pricesAnnouncement="Kurse live"
      />,
    );
    expect(screen.getByRole('status', { name: /kursstatus/i })).toHaveTextContent('Kurse live');
    rerender(<OpenPositionsTable {...base} pricesStale pricesAnnouncement="Kurse veraltet" />);
    expect(screen.getByRole('status', { name: /kursstatus/i })).toHaveTextContent('Kurse veraltet');
  });
});

describe('Sortierung und Verschieben der Kurs-Spalte', () => {
  const prices = {
    a: priceOf('a', { price: '3' }),
    b: priceOf('b', { price: '1' }),
    c: priceOf('c', { price: '2' }),
  };

  it('Zyklus aufsteigend, absteigend, API-Reihenfolge mit aria-sort', async () => {
    const user = userEvent.setup();
    render(<OpenPositionsTable positions={rows} prices={prices} />);
    const header = () => table().querySelector('th[data-column-key="price"]') as HTMLElement;
    await user.click(screen.getByRole('button', { name: /kurs sortieren/i }));
    expect(order()).toEqual(['BBB', 'CCC', 'AAA']);
    expect(header()).toHaveAttribute('aria-sort', 'ascending');
    await user.click(screen.getByRole('button', { name: /kurs sortieren/i }));
    expect(order()).toEqual(['AAA', 'CCC', 'BBB']);
    expect(header()).toHaveAttribute('aria-sort', 'descending');
    await user.click(screen.getByRole('button', { name: /kurs sortieren/i }));
    expect(order()).toEqual(['AAA', 'BBB', 'CCC']);
    expect(header()).not.toHaveAttribute('aria-sort', 'ascending');
  });

  it('Reihenfolge friert nach Klick ein, Zellen zeigen live; erneuter Klick sortiert neu', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<OpenPositionsTable positions={rows} prices={prices} />);
    await user.click(screen.getByRole('button', { name: /kurs sortieren/i }));
    expect(order()).toEqual(['BBB', 'CCC', 'AAA']);
    const moved = { ...prices, b: priceOf('b', { price: '99' }) };
    rerender(<OpenPositionsTable positions={rows} prices={moved} />);
    expect(order()).toEqual(['BBB', 'CCC', 'AAA']);
    expect(cell('Kurs', 0)).toHaveTextContent('99');
    await user.click(screen.getByRole('button', { name: /kurs sortieren/i }));
    expect(order()).toEqual(['BBB', 'AAA', 'CCC']);
  });

  it('Position ohne Kurs steht in beiden Richtungen am Ende', async () => {
    const user = userEvent.setup();
    render(<OpenPositionsTable positions={rows} prices={{ a: prices.a, c: prices.c }} />);
    await user.click(screen.getByRole('button', { name: /kurs sortieren/i }));
    expect(order()).toEqual(['CCC', 'AAA', 'BBB']);
    await user.click(screen.getByRole('button', { name: /kurs sortieren/i }));
    expect(order()).toEqual(['AAA', 'CCC', 'BBB']);
  });

  it('Alt+Pfeil rechts verschiebt Kurs hinter die nächste Spalte, Symbol bleibt links', async () => {
    const user = userEvent.setup();
    render(<OpenPositionsTable positions={rows} prices={prices} />);
    const header = table().querySelector('th[data-column-key="price"]') as HTMLElement;
    header.focus();
    await user.keyboard('{Alt>}{ArrowRight}{/Alt}');
    const keys = within(table())
      .getAllByRole('columnheader')
      .map((h) => h.getAttribute('data-column-key'));
    expect(keys[0]).toBe('symbol');
    expect(keys[1]).not.toBe('price');
    expect(keys[2]).toBe('price');
  });
});
