import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { FETCHED_AT } from '@/test/fixtures/openPositions';
import { OpenPositionsHeader } from '@/presentation/position/components/OpenPositionsHeader';

type HeaderOverrides = Partial<{
  totalUnrealizedProfit: string;
  positionCount: number;
  fetchedAt: number;
  onRefresh: () => void;
}>;

function renderHeader(overrides: HeaderOverrides = {}) {
  const onRefresh = overrides.onRefresh ?? vi.fn();
  render(
    <OpenPositionsHeader
      totalUnrealizedProfit={overrides.totalUnrealizedProfit ?? '18.75'}
      positionCount={overrides.positionCount ?? 2}
      fetchedAt={overrides.fetchedAt ?? FETCHED_AT}
      onRefresh={onRefresh}
    />,
  );
  return { onRefresh };
}

describe('OpenPositionsHeader', () => {
  it('zeigt die Summe des unrealisierten PnL mit zwei Nachkommastellen', () => {
    renderHeader({ totalUnrealizedProfit: '18.745' });

    expect(screen.getByText(/18\.75/)).toBeInTheDocument();
  });

  it('tönt eine positive Summe positiv', () => {
    renderHeader({ totalUnrealizedProfit: '18.75' });

    expect(screen.getByText(/18\.75/)).toHaveClass('positive');
  });

  it('tönt eine negative Summe negativ', () => {
    renderHeader({ totalUnrealizedProfit: '-6.25' });

    expect(screen.getByText(/[-−]6\.25/)).toHaveClass('negative');
  });

  it('tönt eine Summe von 0 neutral', () => {
    renderHeader({ totalUnrealizedProfit: '0' });

    expect(screen.getByText(/0\.00/)).toHaveClass('neutral');
  });

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
});
