import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { expect, it, vi } from 'vitest';
import type { InitialStopLossGateway } from '@/domain/position/InitialStopLossGateway';
import type { OpenPositionsGateway } from '@/domain/position/OpenPositionsGateway';
import { OpenPositionsPage } from '@/presentation/position/components/OpenPositionsPage';
import { suiLongPosition } from '@/test/fixtures/openPositions';

// Erwartet: OpenPositionsPage-Prop `initialStopLossGateway?: InitialStopLossGateway`; nach erfolgreichem
// Speichern lädt die Seite die Liste sofort neu (refresh des View-Models).

it('lädt die Liste nach dem Speichern des Ursprungs-SL neu', async () => {
  const position = { ...suiLongPosition, levels: undefined, initialStopLossSource: null };
  const snapshot = { positions: [position], totalUnrealizedProfit: '25', fetchedAt: 1 };
  const fetchOpenPositions = vi
    .fn<OpenPositionsGateway['fetchOpenPositions']>()
    .mockResolvedValue({ kind: 'ok', snapshot });
  const setInitialStopLoss = vi
    .fn<InitialStopLossGateway['setInitialStopLoss']>()
    .mockResolvedValue({ kind: 'ok' });
  const initialStopLossGateway: InitialStopLossGateway = {
    setInitialStopLoss,
    clearInitialStopLoss: vi.fn().mockResolvedValue({ kind: 'ok' }),
  };
  render(
    <OpenPositionsPage
      gateway={{ fetchOpenPositions }}
      initialStopLossGateway={initialStopLossGateway}
    />,
  );

  await userEvent.click(
    await screen.findByRole('button', { name: 'Ursprünglichen SL eintragen für SUI-USDT' }),
  );
  await userEvent.type(screen.getByLabelText('Ursprünglicher SL'), '0.9');
  await userEvent.click(screen.getByRole('button', { name: 'Speichern' }));

  await waitFor(() => expect(fetchOpenPositions).toHaveBeenCalledTimes(2));
  expect(setInitialStopLoss).toHaveBeenCalledWith('9100000000000000001', '0.9');
});
