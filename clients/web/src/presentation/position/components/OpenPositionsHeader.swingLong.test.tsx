import { render, screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { AccountSummary } from '@/domain/account/AccountSummary';
import { OpenPositionsHeader } from '@/presentation/position/components/OpenPositionsHeader';
import { FETCHED_AT } from '@/test/fixtures/openPositions';

const account: AccountSummary = {
  balance: '1100.00',
  equity: '1234.5',
  availableMargin: '1000.00',
  month: '2026-10',
  monthlyRealizedPnl: '61.73',
  monthlyPnlPercent: '5.00',
  periods: [{ period: 'today', realizedPnl: '33.82', pnlPercent: '1.59' }],
  unrealizedPnl: '18.75',
  unrealizedPnlPercent: '1.52',
  swingLongUnrealizedPnl: '40.5',
  swingLongRealizedPnl: '-12.25',
};

function renderHeader(
  swingLongUnrealizedProfit: string | null = '40.5',
  view?: 'trades' | 'swing-long',
) {
  render(
    <OpenPositionsHeader
      totalUnrealizedProfit="18.75"
      swingLongUnrealizedProfit={swingLongUnrealizedProfit ?? undefined}
      positionCount={2}
      view={view}
      fetchedAt={FETCHED_AT}
      onRefresh={vi.fn()}
      account={account}
    />,
  );
}

describe('OpenPositionsHeader Swing Long', () => {
  it('zeigt Swing-Long-Werte in einem eigenen, benannten Bereich', () => {
    renderHeader();

    const area = screen.getByRole('group', { name: 'Swing Long' });
    expect(within(area).getByText('Unrealisiert')).toBeInTheDocument();
    expect(within(area).getByText('+40.50')).toBeInTheDocument();
    expect(within(area).getByText('Realisiert (Monat)')).toBeInTheDocument();
    expect(within(area).getByText(/^[-−]12\.25$/)).toBeInTheDocument();
  });

  it('hält die Handels-Summe außerhalb des Swing-Long-Bereichs', () => {
    renderHeader();

    const area = screen.getByRole('group', { name: 'Swing Long' });
    expect(within(area).queryByText(/18\.75/)).not.toBeInTheDocument();
    expect(screen.getByText(/^\+18\.75/)).toBeInTheDocument();
    expect(screen.getByText('Unrealisierter PnL')).toBeInTheDocument();
  });

  it('tönt Swing-Long-Werte nach Vorzeichen', () => {
    renderHeader();

    const area = screen.getByRole('group', { name: 'Swing Long' });
    expect(within(area).getByText('+40.50')).toHaveClass('positive');
    expect(within(area).getByText(/^[-−]12\.25$/)).toHaveClass('negative');
  });

  it('zeigt ohne Swing-Long-Summe keinen Swing-Long-Bereich', () => {
    renderHeader(null);

    expect(screen.queryByRole('group', { name: 'Swing Long' })).not.toBeInTheDocument();
  });

  it('zeigt im Swing-Long-Reiter den Swing-Long-PnL als Hauptwert statt der Handels-Summe', () => {
    renderHeader('40.5', 'swing-long');

    expect(screen.getByText('Unrealisierter PnL Swing Long')).toBeInTheDocument();
    expect(screen.getByText('+40.50')).toBeInTheDocument();
    expect(screen.queryByText(/18\.75/)).not.toBeInTheDocument();
    expect(screen.queryByText(/\(\+1\.52 %\)/)).not.toBeInTheDocument();
  });

  it('zeigt im Swing-Long-Reiter im Bereich nur den realisierten Monatswert', () => {
    renderHeader('40.5', 'swing-long');

    const area = screen.getByRole('group', { name: 'Swing Long' });
    expect(within(area).queryByText('Unrealisiert')).not.toBeInTheDocument();
    expect(within(area).getByText('Realisiert (Monat)')).toBeInTheDocument();
  });
});
