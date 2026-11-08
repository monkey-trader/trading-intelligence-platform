import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { OpenPositionsGateway } from '@/domain/position/OpenPositionsGateway';
import type { SwingLongGateway } from '@/domain/position/SwingLongGateway';
import { OpenPositionsPage } from '@/presentation/position/components/OpenPositionsPage';
import { mixedSnapshot } from '@/test/fixtures/openPositions';

const stubGateway: SwingLongGateway = {
  setSwingLong: vi.fn().mockResolvedValue({ kind: 'ok' }),
  clearSwingLong: vi.fn().mockResolvedValue({ kind: 'ok' }),
};

function renderPage(swingLongGateway?: SwingLongGateway) {
  const fetchOpenPositions = vi
    .fn<OpenPositionsGateway['fetchOpenPositions']>()
    .mockResolvedValue({ kind: 'ok', snapshot: mixedSnapshot });
  render(
    <OpenPositionsPage gateway={{ fetchOpenPositions }} swingLongGateway={swingLongGateway} />,
  );
  return { fetchOpenPositions };
}

describe('OpenPositionsPage Swing Long', () => {
  it('zeigt im Reiter Trades nur Positionen ohne Swing-Long-Flag', async () => {
    renderPage(stubGateway);

    expect(
      await screen.findByRole('switch', { name: 'Swing Long für SUI-USDT' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('switch', { name: 'Swing Long für NCCOGOLD2USD-USDT' }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('switch', { name: 'Swing Long für ETH-USDT' }),
    ).not.toBeInTheDocument();
  });

  it('zeigt im Reiter Swing Long nur Positionen mit Flag', async () => {
    renderPage(stubGateway);
    await screen.findByRole('switch', { name: 'Swing Long für SUI-USDT' });

    await userEvent.click(screen.getByRole('tab', { name: 'Swing Long' }));

    expect(screen.getByRole('switch', { name: 'Swing Long für ETH-USDT' })).toBeInTheDocument();
    expect(
      screen.queryByRole('switch', { name: 'Swing Long für SUI-USDT' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('switch', { name: 'Swing Long für NCCOGOLD2USD-USDT' }),
    ).not.toBeInTheDocument();
  });

  it('setzt per Schalter das Flag und lädt die Liste neu', async () => {
    const swingLongGateway: SwingLongGateway = {
      setSwingLong: vi.fn().mockResolvedValue({ kind: 'ok' }),
      clearSwingLong: vi.fn().mockResolvedValue({ kind: 'ok' }),
    };
    const { fetchOpenPositions } = renderPage(swingLongGateway);

    await userEvent.click(await screen.findByRole('switch', { name: 'Swing Long für SUI-USDT' }));

    expect(swingLongGateway.setSwingLong).toHaveBeenCalledWith('9100000000000000001');
    await waitFor(() => expect(fetchOpenPositions).toHaveBeenCalledTimes(2));
  });

  it('bietet im Reiter Swing Long das Entfernen des Flags an', async () => {
    const swingLongGateway: SwingLongGateway = {
      setSwingLong: vi.fn().mockResolvedValue({ kind: 'ok' }),
      clearSwingLong: vi.fn().mockResolvedValue({ kind: 'ok' }),
    };
    renderPage(swingLongGateway);
    await screen.findByRole('switch', { name: 'Swing Long für SUI-USDT' });
    await userEvent.click(screen.getByRole('tab', { name: 'Swing Long' }));

    const toggle = within(screen.getByRole('table')).getByRole('switch', {
      name: 'Swing Long für ETH-USDT',
    });
    await userEvent.click(toggle);

    expect(swingLongGateway.clearSwingLong).toHaveBeenCalledWith('9100000000000000004');
  });

  it('zeigt ohne Gateway keine Schalter', async () => {
    renderPage();
    await screen.findByRole('table');

    expect(screen.queryByRole('switch')).not.toBeInTheDocument();
  });
});
