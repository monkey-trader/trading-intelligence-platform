import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import type { OpenPositionsGateway } from '@/domain/position/OpenPositionsGateway';
import type { OpenPositionsResult } from '@/domain/position/OpenPositionsResult';
import { sampleSnapshot } from '@/test/fixtures/openPositions';
import { OpenPositionsPage } from '@/presentation/position/components/OpenPositionsPage';

describe('OpenPositionsPage Sortierung bleibt bei Fehler (AC6)', () => {
  it('behält die gewählte Sortierung, wenn das Neuladen fehlschlägt (veraltet)', async () => {
    const results: OpenPositionsResult[] = [
      { kind: 'ok', snapshot: sampleSnapshot },
      { kind: 'problem', problemType: 'backend-unreachable' },
    ];
    let call = 0;
    const gateway: OpenPositionsGateway = {
      fetchOpenPositions: () =>
        Promise.resolve(results[Math.min(call++, results.length - 1)] as OpenPositionsResult),
    };
    const user = userEvent.setup();
    render(<OpenPositionsPage gateway={gateway} />);
    await screen.findByRole('table', { name: /offene positionen/i });
    await waitFor(() => expect(screen.queryByTestId('position-placeholder-row')).toBeNull());
    const table = screen.getByRole('table', { name: /offene positionen/i });

    await user.click(screen.getByRole('button', { name: /^Symbol sortieren/ }));
    const header = table.querySelector('th[data-column-key="symbol"]');
    expect(header).toHaveAttribute('aria-sort', 'ascending');

    await user.click(screen.getByRole('button', { name: /aktualisieren/i }));
    await waitFor(() => expect(call).toBeGreaterThan(1));
    await screen.findByText(/möglicherweise veraltet/);

    const sameTable = within(document.body).getByRole('table', { name: /offene positionen/i });
    expect(sameTable.querySelector('th[data-column-key="symbol"]')).toHaveAttribute(
      'aria-sort',
      'ascending',
    );
  });
});
