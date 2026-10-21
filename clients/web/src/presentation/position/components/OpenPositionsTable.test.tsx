import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { OpenPosition } from '@/domain/position/OpenPosition';
import type { AssetIcon } from '@/domain/asset/AssetIcon';
import {
  breakEvenPosition,
  noStopLossPosition,
  goldShortPosition,
  unknownOrdersPosition,
  flatPosition,
  suiLongPosition,
} from '@/test/fixtures/openPositions';
import { OpenPositionsTable } from '@/presentation/position/components/OpenPositionsTable';

const COLUMN_LABELS = [
  'Symbol',
  'Positionswert',
  'Margin',
  'Unreal. PnL',
  'Real. PnL',
  'Einstieg',
  'TP / SL',
  'TPs (je 33 %)',
] as const;

type ColumnLabel = (typeof COLUMN_LABELS)[number];

const goldIcon: AssetIcon = {
  symbol: 'NCCOGOLD2USD-USDT',
  baseAsset: 'XAU',
  displayName: 'Gold',
  iconUrl: null,
};
const suiIcon: AssetIcon = {
  symbol: 'SUI-USDT',
  baseAsset: 'SUI',
  displayName: 'SUI',
  iconUrl: 'https://coin-images.coingecko.com/coins/images/26375/large/sui.png',
};
const silverIcon: AssetIcon = {
  symbol: 'NCCOXAG2USD-USDT',
  baseAsset: 'XAG',
  displayName: 'Silber',
  iconUrl: null,
};

function iconMap(...icons: readonly AssetIcon[]): ReadonlyMap<string, AssetIcon> {
  return new Map(icons.map((icon) => [icon.symbol, icon]));
}

function renderTable(positions: readonly OpenPosition[], icons?: ReadonlyMap<string, AssetIcon>) {
  render(<OpenPositionsTable positions={positions} iconsBySymbol={icons} />);
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
  const row = bodyRows(table).find((candidate) =>
    (cellsOf(candidate)[0]?.textContent ?? '').includes(symbol),
  );
  if (!row) throw new Error(`Zeile für ${symbol} fehlt`);
  return row;
}

function hasTone(cell: HTMLElement, tone: 'positive' | 'negative' | 'neutral'): boolean {
  return cell.classList.contains(tone) || cell.querySelector(`.${tone}`) !== null;
}

describe('OpenPositionsTable', () => {
  it('ist eine semantische Tabelle mit den 8 Spalten in fester Reihenfolge', () => {
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
    expect(cellOf(rows[0] as HTMLElement, 'Symbol')).toHaveTextContent('NCCOGOLD2USD');
    expect(cellOf(rows[1] as HTMLElement, 'Symbol')).toHaveTextContent('SUI');
  });

  it('versieht jede Zelle mit data-label für die mobile Kartenansicht', () => {
    const table = renderTable([goldShortPosition, suiLongPosition]);

    for (const row of bodyRows(table)) {
      expect(cellsOf(row).map((cell) => cell.getAttribute('data-label'))).toEqual([
        ...COLUMN_LABELS,
      ]);
    }
  });

  it('entfernte Spalten und Texte kommen nicht mehr vor', () => {
    const table = renderTable([suiLongPosition]);
    const headers = within(table)
      .getAllByRole('columnheader')
      .map((header) => header.textContent?.trim() ?? '');
    for (const removed of [
      'Seite',
      'Menge',
      'Mark',
      'Hebel',
      'PnL %',
      'Eröffnet',
      'Auto Break-even',
    ]) {
      expect(headers).not.toContain(removed);
    }
  });

  it('zeigt in der Symbol-Zelle nur den Anzeigenamen, kein -USDT und keinen Quote-Span', () => {
    const table = renderTable([suiLongPosition], iconMap(suiIcon));
    const symbolCell = cellOf(rowFor(table, 'SUI'), 'Symbol');

    expect(symbolCell).toHaveTextContent('SUI');
    expect(symbolCell.textContent).not.toContain('USDT');
    expect(within(symbolCell).queryByText('-USDT')).not.toBeInTheDocument();
    expect(within(symbolCell).getByText('Long')).toBeInTheDocument();
    expect(within(symbolCell).getByText('Isolated')).toBeInTheDocument();
    expect(within(symbolCell).getByText('10X')).toBeInTheDocument();
  });

  it('zeigt Rohstoffe mit deutschem Anzeigenamen', () => {
    const silverPosition: OpenPosition = {
      ...goldShortPosition,
      positionId: '9100000000000000009',
      symbol: 'NCCOXAG2USD-USDT',
    };
    const table = renderTable([goldShortPosition, silverPosition], iconMap(goldIcon, silverIcon));

    const goldCell = cellOf(bodyRows(table)[0] as HTMLElement, 'Symbol');
    const silverCell = cellOf(bodyRows(table)[1] as HTMLElement, 'Symbol');
    expect(goldCell).toHaveTextContent('Gold');
    expect(goldCell.textContent).not.toContain('NCCO');
    expect(goldCell.textContent).not.toContain('USDT');
    expect(silverCell).toHaveTextContent('Silber');
    expect(silverCell.textContent).not.toContain('USDT');
  });

  it('nutzt ohne Icon-Daten den Teil des Symbols vor dem ersten "-" in Großbuchstaben', () => {
    const lowerCasePosition: OpenPosition = {
      ...suiLongPosition,
      positionId: '9100000000000000010',
      symbol: 'ondo-usdt',
    };
    const table = renderTable([goldShortPosition, lowerCasePosition]);

    const goldCell = cellOf(bodyRows(table)[0] as HTMLElement, 'Symbol');
    expect(goldCell).toHaveTextContent('NCCOGOLD2USD');
    expect(goldCell.textContent).not.toContain('USDT');
    const ondoCell = cellOf(bodyRows(table)[1] as HTMLElement, 'Symbol');
    expect(ondoCell).toHaveTextContent('ONDO');
    expect(ondoCell.textContent).not.toContain('USDT');
  });

  it('nutzt für Symbole ohne Eintrag in der Map den Fallback-Namen', () => {
    const table = renderTable([goldShortPosition, suiLongPosition], iconMap(suiIcon));

    expect(cellOf(rowFor(table, 'NCCOGOLD2USD'), 'Symbol').textContent).not.toContain('USDT');
    expect(cellOf(rowFor(table, 'SUI'), 'Symbol')).toHaveTextContent('SUI');
  });

  it('zeigt bei SHORT und Cross die passenden Chips', () => {
    const table = renderTable([goldShortPosition, flatPosition]);

    const goldCell = cellOf(rowFor(table, 'NCCOGOLD2USD'), 'Symbol');
    expect(within(goldCell).getByText('Short')).toBeInTheDocument();
    expect(within(goldCell).getByText('15X')).toBeInTheDocument();
    const flatCell = cellOf(rowFor(table, 'BTC'), 'Symbol');
    expect(within(flatCell).getByText('Cross')).toBeInTheDocument();
    expect(within(flatCell).getByText('1X')).toBeInTheDocument();
  });

  it('formatiert die Werte einer LONG-Position', () => {
    const table = renderTable([suiLongPosition]);
    const row = rowFor(table, 'SUI');

    expect(cellOf(row, 'Positionswert')).toHaveTextContent(/^505\.00\sUSDT$/);
    expect(cellOf(row, 'Margin')).toHaveTextContent(/^60\.50\sUSDT$/);
    expect(cellOf(row, 'Unreal. PnL')).toHaveTextContent(/^\+25\.00\s*\(\+52\.08\s?%\)$/);
    expect(cellOf(row, 'Real. PnL')).toHaveTextContent(/^[-−]0\.24$/);
    expect(cellOf(row, 'Einstieg')).toHaveTextContent(/^0\.96$/);
    expect(cellOf(row, 'TP / SL')).toHaveTextContent(/^TP\s1\.2\d*\s*SL\s0\.9\d*$/);
  });

  it('formatiert die Werte einer SHORT-Position mit Tausendertrennung und Minus', () => {
    const table = renderTable([goldShortPosition]);
    const row = rowFor(table, 'NCCOGOLD2USD');

    expect(cellOf(row, 'Positionswert')).toHaveTextContent(/^1['’]043\.75\sUSDT$/);
    expect(cellOf(row, 'Margin')).toHaveTextContent(/^65\.00\sUSDT$/);
    expect(cellOf(row, 'Einstieg')).toHaveTextContent(/^4['’]150$/);
    expect(cellOf(row, 'Unreal. PnL')).toHaveTextContent(/^[-−]6\.25\s*\([-−]9\.04\s?%\)$/);
    expect(cellOf(row, 'Real. PnL')).toHaveTextContent(/^[-−]0\.52$/);
  });

  it('zeigt TP und SL in zwei Zeilen', () => {
    const row = rowFor(renderTable([goldShortPosition]), 'NCCOGOLD2USD');
    const cell = cellOf(row, 'TP / SL');

    expect(cell).toHaveTextContent(/TP\s4['’]100/);
    expect(cell).toHaveTextContent(/SL\s4['’]200/);
    expect(within(cell).getByText(/^TP/)).toBeInTheDocument();
    expect(within(cell).getByText(/^SL/)).toBeInTheDocument();
  });

  it('zeigt – bei TP und SL null', () => {
    const row = rowFor(renderTable([unknownOrdersPosition]), 'SUI');
    const cell = cellOf(row, 'TP / SL');

    expect(cell).toHaveTextContent(/TP\s[–-]/);
    expect(cell).toHaveTextContent(/SL\s[–-]/);
  });

  it('tönt die PnL-Zellen nach dem Vorzeichen', () => {
    const table = renderTable([goldShortPosition, suiLongPosition, flatPosition]);
    const winner = rowFor(table, 'SUI');
    const loser = rowFor(table, 'NCCOGOLD2USD');
    const flat = rowFor(table, 'BTC');

    expect(hasTone(cellOf(winner, 'Unreal. PnL'), 'positive')).toBe(true);
    expect(hasTone(cellOf(loser, 'Unreal. PnL'), 'negative')).toBe(true);
    expect(hasTone(cellOf(flat, 'Unreal. PnL'), 'neutral')).toBe(true);
    expect(hasTone(cellOf(flat, 'Unreal. PnL'), 'positive')).toBe(false);
    expect(hasTone(cellOf(flat, 'Unreal. PnL'), 'negative')).toBe(false);
  });

  it('zeigt positive PnL-Werte mit explizitem "+" und Preise ohne "+"', () => {
    const profitablePosition: OpenPosition = {
      ...suiLongPosition,
      positionId: '9100000000000000004',
      symbol: 'ETH-USDT',
      avgOpenPrice: '2500.5',
      unrealizedProfit: '1234.5',
      realisedProfit: '12.3456',
      pnlRatio: '0.12345',
    };
    const row = rowFor(renderTable([profitablePosition]), 'ETH');

    expect(cellOf(row, 'Unreal. PnL')).toHaveTextContent(/^\+1['’]234\.50\s*\(\+12\.35\s?%\)$/);
    expect(cellOf(row, 'Real. PnL')).toHaveTextContent(/^\+12\.35$/);
    expect(cellOf(row, 'Einstieg')).toHaveTextContent(/^2['’]500\.5$/);
  });

  it('zeigt PnL-Werte, die gerundet 0 sind, ohne Vorzeichen', () => {
    const nearlyFlatPosition: OpenPosition = {
      ...flatPosition,
      unrealizedProfit: '0.004',
      realisedProfit: '-0.004',
      pnlRatio: '-0.00004',
    };
    const row = rowFor(renderTable([nearlyFlatPosition]), 'BTC');

    expect(cellOf(row, 'Unreal. PnL')).toHaveTextContent(/^0\.00\s*\(0\.00\s?%\)$/);
    expect(cellOf(row, 'Real. PnL')).toHaveTextContent(/^0\.00$/);
    expect(hasTone(cellOf(row, 'Real. PnL'), 'neutral')).toBe(true);
  });

  it('zeigt bei PnL genau 0 kein Vorzeichen', () => {
    const row = rowFor(renderTable([flatPosition]), 'BTC');

    expect(cellOf(row, 'Unreal. PnL')).toHaveTextContent(/^0\.00\s*\(0\.00\s?%\)$/);
    expect(cellOf(row, 'Real. PnL')).toHaveTextContent(/^0\.00$/);
  });

  it('tönt den realisierten PnL nach dem gerundeten Vorzeichen', () => {
    const realisedWinner: OpenPosition = { ...suiLongPosition, realisedProfit: '3.5' };
    const table = renderTable([realisedWinner, goldShortPosition]);

    expect(hasTone(cellOf(rowFor(table, 'SUI'), 'Real. PnL'), 'positive')).toBe(true);
    expect(hasTone(cellOf(rowFor(table, 'NCCOGOLD2USD'), 'Real. PnL'), 'negative')).toBe(true);
  });
});

describe('OpenPositionsTable: Chart-Auswahl', () => {
  it('benennt den Chart-Button mit dem Anzeigenamen statt dem Rohsymbol', () => {
    renderTable([goldShortPosition, suiLongPosition], iconMap(goldIcon, suiIcon));

    const goldButton = screen.getByRole('button', { name: /chart.*Gold/i });
    expect(goldButton.getAttribute('aria-label')).not.toContain('NCCO');
    expect(goldButton.getAttribute('aria-label')).not.toContain('USDT');
    expect(
      screen.getByRole('button', { name: /chart.*SUI/i }).getAttribute('aria-label'),
    ).not.toContain('USDT');
  });

  it('hat pro Zeile einen Chart-Button mit Symbol im Namen, ohne die Spalten zu ändern', () => {
    const table = renderTable([suiLongPosition, goldShortPosition]);

    expect(screen.getByRole('button', { name: /chart.*SUI/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /chart.*NCCOGOLD2USD/i })).toBeInTheDocument();
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

    await user.click(screen.getByRole('button', { name: /chart.*NCCOGOLD2USD/i }));

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

    expect(screen.getByRole('button', { name: /chart.*SUI/i })).toHaveAttribute(
      'aria-expanded',
      'true',
    );
    expect(screen.getByRole('button', { name: /chart.*NCCOGOLD2USD/i })).toHaveAttribute(
      'aria-expanded',
      'false',
    );
  });
});

describe('OpenPositionsTable: SL, TP und Break-even', () => {
  it('zeigt in der SL-Zeile das Badge BE (role img, Break-even), wenn breakEven true ist', () => {
    const cell = cellOf(rowFor(renderTable([breakEvenPosition]), 'SUI'), 'TP / SL');

    const badge = within(cell).getByRole('img', { name: 'Break-even' });
    expect(badge).toHaveAttribute('aria-label', 'Break-even');
    expect(badge).toHaveTextContent('BE');
    expect(cell).toHaveTextContent(/SL\s0\.97/);
  });

  it('zeigt kein BE-Badge bei breakEven false', () => {
    const cell = cellOf(rowFor(renderTable([goldShortPosition]), 'NCCOGOLD2USD'), 'TP / SL');

    expect(within(cell).queryByRole('img', { name: 'Break-even' })).not.toBeInTheDocument();
    expect(within(cell).queryByText('BE')).not.toBeInTheDocument();
  });

  it('zeigt den Hinweis "kein SL", wenn Orders bekannt sind und kein SL existiert', () => {
    const cell = cellOf(rowFor(renderTable([noStopLossPosition]), 'SUI'), 'TP / SL');

    expect(within(cell).getByText('kein SL')).toBeInTheDocument();
    expect(within(cell).queryByRole('img', { name: 'Break-even' })).not.toBeInTheDocument();
  });

  it('zeigt bei vorhandenem SL keinen Hinweis "kein SL"', () => {
    const cell = cellOf(rowFor(renderTable([goldShortPosition]), 'NCCOGOLD2USD'), 'TP / SL');

    expect(within(cell).queryByText('kein SL')).not.toBeInTheDocument();
  });

  it('zeigt bei breakEven null (Orders unbekannt) weder Badge noch "kein SL"', () => {
    const cell = cellOf(rowFor(renderTable([unknownOrdersPosition]), 'SUI'), 'TP / SL');

    expect(within(cell).queryByRole('img', { name: 'Break-even' })).not.toBeInTheDocument();
    expect(within(cell).queryByText('BE')).not.toBeInTheDocument();
    expect(within(cell).queryByText('kein SL')).not.toBeInTheDocument();
  });

  it('tönt fehlenden SL nicht negativ, vorhandenen SL schon', () => {
    const table = renderTable([unknownOrdersPosition, goldShortPosition]);
    const missingLine = within(cellOf(bodyRows(table)[0] as HTMLElement, 'TP / SL')).getByText(
      /^SL/,
    ).parentElement as HTMLElement;
    const presentLine = within(cellOf(bodyRows(table)[1] as HTMLElement, 'TP / SL')).getByText(
      /^SL/,
    ).parentElement as HTMLElement;

    // Ausnahme: Klassenwechsel, da getComputedStyle die CSS-Module-Tönung nicht sichtbar macht.
    expect(missingLine.className).not.toMatch(/slLine|negative/);
    expect(presentLine.className).toMatch(/slLine|negative/);
  });
});
