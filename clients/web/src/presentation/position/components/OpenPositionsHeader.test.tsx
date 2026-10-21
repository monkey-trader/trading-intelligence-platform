import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { AccountSummary } from '@/domain/account/AccountSummary';
import { FETCHED_AT } from '@/test/fixtures/openPositions';
import { OpenPositionsHeader } from '@/presentation/position/components/OpenPositionsHeader';

type HeaderOverrides = Partial<{
  totalUnrealizedProfit: string;
  positionCount: number;
  fetchedAt: number;
  onRefresh: () => void;
  account: AccountSummary | undefined;
}>;

function renderHeader(overrides: HeaderOverrides = {}) {
  const onRefresh = overrides.onRefresh ?? vi.fn();
  render(
    <OpenPositionsHeader
      totalUnrealizedProfit={overrides.totalUnrealizedProfit ?? '18.75'}
      positionCount={overrides.positionCount ?? 2}
      fetchedAt={overrides.fetchedAt ?? FETCHED_AT}
      onRefresh={onRefresh}
      account={overrides.account}
    />,
  );
  return { onRefresh };
}

describe('OpenPositionsHeader', () => {
  it('zeigt die Summe des unrealisierten PnL mit zwei Nachkommastellen', () => {
    renderHeader({ totalUnrealizedProfit: '18.745' });

    expect(screen.getByText(/^\+18\.75$/)).toBeInTheDocument();
  });

  it('tönt eine positive Summe positiv', () => {
    renderHeader({ totalUnrealizedProfit: '18.75' });

    expect(screen.getByText(/^\+18\.75$/)).toHaveClass('positive');
  });

  it('tönt eine negative Summe negativ', () => {
    renderHeader({ totalUnrealizedProfit: '-6.25' });

    expect(screen.getByText(/^[-−]6\.25$/)).toHaveClass('negative');
  });

  it('tönt eine Summe von 0 neutral', () => {
    renderHeader({ totalUnrealizedProfit: '0' });

    expect(screen.getByText(/^0\.00$/)).toHaveClass('neutral');
  });

  it('zeigt eine positive Summe mit explizitem "+" und de-CH-Tausendertrennung', () => {
    renderHeader({ totalUnrealizedProfit: '1234.5' });

    expect(screen.getByText(/^\+1['’]234\.50$/)).toBeInTheDocument();
  });

  it.each(['0.004', '-0.004', '-0'])(
    'zeigt eine Summe %s, die gerundet 0 ist, ohne Vorzeichen als 0.00',
    (totalUnrealizedProfit) => {
      renderHeader({ totalUnrealizedProfit });

      expect(screen.getByText(/^0\.00$/)).toHaveClass('neutral');
      expect(screen.queryByText(/[+\-−]0\.00/)).not.toBeInTheDocument();
    },
  );

  it('zeigt die Anzahl der offenen Positionen', () => {
    renderHeader({ positionCount: 2 });

    expect(screen.getByText(/2 Positionen/)).toBeInTheDocument();
  });

  it('verwendet bei einer Position die Einzahl', () => {
    renderHeader({ positionCount: 1 });

    expect(screen.getByText(/1 Position\b/)).toBeInTheDocument();
    expect(screen.queryByText(/1 Positionen/)).not.toBeInTheDocument();
  });

  it('zeigt, wann die Daten zuletzt aktualisiert wurden (Europe/Zurich)', () => {
    renderHeader({ fetchedAt: FETCHED_AT });

    expect(screen.getByText(/zuletzt aktualisiert/i)).toBeInTheDocument();
    expect(screen.getByText(/14:34:56/)).toBeInTheDocument();
  });

  it('ruft onRefresh beim Klick auf Aktualisieren auf', async () => {
    const user = userEvent.setup();
    const { onRefresh } = renderHeader();

    await user.click(screen.getByRole('button', { name: /aktualisieren/i }));

    expect(onRefresh).toHaveBeenCalledTimes(1);
  });

  describe('Konto', () => {
    const account: AccountSummary = {
      balance: '1100.00',
      equity: '1234.5',
      availableMargin: '1000.00',
      month: '2026-10',
      monthlyRealizedPnl: '61.73',
      monthlyPnlPercent: '5.00',
      periods: [
        { period: 'today', realizedPnl: '33.82', pnlPercent: '1.59' },
        { period: '7d', realizedPnl: '-12.5', pnlPercent: '-0.58' },
        { period: '30d', realizedPnl: '61.73', pnlPercent: undefined },
      ],
      unrealizedPnl: '18.75',
      unrealizedPnlPercent: '1.52',
    };

    it('zeigt ohne Kontodaten keinen Kontobereich', () => {
      renderHeader();

      expect(screen.queryByLabelText('Konto')).not.toBeInTheDocument();
    });

    it('zeigt Kontostand, Monats-PnL und Prozent', () => {
      renderHeader({ account });

      const region = screen.getByLabelText('Konto');
      expect(region).toHaveTextContent(/Kontostand/);
      expect(region).toHaveTextContent(/1.?234\.50/);
      expect(region).toHaveTextContent('+33.82');
      expect(region).toHaveTextContent('(+1.59 %)');
    });

    it('zeigt die Prozent des unrealisierten PnL neben der Summe', () => {
      renderHeader({ account });

      expect(screen.getByLabelText('Zusammenfassung')).toHaveTextContent('+18.75 (+1.52 %)');
    });

    it('schaltet den Zeitraum um, tönt Verluste negativ und lässt unbekannte Prozent weg', async () => {
      renderHeader({ account });

      await userEvent.click(screen.getByRole('button', { name: '7T' }));
      expect(screen.getByText(/^-12\.50$/)).toHaveClass('negative');
      expect(screen.getByLabelText('Konto')).toHaveTextContent('(-0.58 %)');

      await userEvent.click(screen.getByRole('button', { name: '30T' }));
      expect(screen.getByLabelText('Konto')).toHaveTextContent('+61.73');
      expect(screen.getByLabelText('Konto')).not.toHaveTextContent('%');
    });
  });
});
