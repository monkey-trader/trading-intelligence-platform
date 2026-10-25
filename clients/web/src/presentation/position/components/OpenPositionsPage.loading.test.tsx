import { render, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { OpenPositionsGateway } from '@/domain/position/OpenPositionsGateway';
import type { OpenPositionsResult } from '@/domain/position/OpenPositionsResult';
import { sampleSnapshot } from '@/test/fixtures/openPositions';
import { OpenPositionsPage } from '@/presentation/position/components/OpenPositionsPage';

const pendingGateway: OpenPositionsGateway = {
  fetchOpenPositions: () => new Promise<OpenPositionsResult>(() => {}),
};

function gatewayReturning(result: OpenPositionsResult): OpenPositionsGateway {
  return { fetchOpenPositions: () => Promise.resolve(result) };
}

describe('OpenPositionsPage Ladezustand (AC7)', () => {
  it('rendert beim Erstladen die Tabelle mit allen Spaltenköpfen', () => {
    render(<OpenPositionsPage gateway={pendingGateway} />);

    const table = screen.getByRole('table', { name: /offene positionen/i });
    expect(within(table).getAllByRole('columnheader').length).toBeGreaterThan(5);
    expect(table.querySelector('th[data-column-key="symbol"]')).not.toBeNull();
  });

  it('zeigt beim Erstladen Platzhalterzeilen', () => {
    render(<OpenPositionsPage gateway={pendingGateway} />);

    expect(screen.getAllByTestId('position-placeholder-row').length).toBeGreaterThan(0);
  });

  it('deaktiviert beim Erstladen alle Sortier-Knöpfe', () => {
    render(<OpenPositionsPage gateway={pendingGateway} />);

    const sortButtons = screen.getAllByRole('button', { name: /sortieren/i });
    expect(sortButtons.length).toBeGreaterThan(0);
    for (const button of sortButtons) expect(button).toBeDisabled();
  });

  it('hält den Ladetext als role=status auffindbar', () => {
    render(<OpenPositionsPage gateway={pendingGateway} />);

    const loadingStatus = screen
      .getAllByRole('status')
      .find((region) => /lade offene positionen/i.test(region.textContent ?? ''));
    expect(loadingStatus).toBeDefined();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('entfernt Platzhalter und Ladetext nach dem Laden und aktiviert die Sortierung', async () => {
    render(
      <OpenPositionsPage gateway={gatewayReturning({ kind: 'ok', snapshot: sampleSnapshot })} />,
    );

    await waitFor(() => expect(screen.queryByTestId('position-placeholder-row')).toBeNull());
    expect(screen.queryByText(/lade offene positionen/i)).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Symbol sortieren/ })).toBeEnabled();
  });

  it('zeigt bei geladener leerer Liste keine Platzhalter und keine Tabelle', async () => {
    render(
      <OpenPositionsPage
        gateway={gatewayReturning({
          kind: 'ok',
          snapshot: { ...sampleSnapshot, positions: [] },
        })}
      />,
    );

    await screen.findByText('Keine offenen Positionen.');
    expect(screen.queryByTestId('position-placeholder-row')).toBeNull();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('zeigt im Fehlerfall ohne Daten keine Platzhalter, aber den Alert', async () => {
    render(
      <OpenPositionsPage
        gateway={gatewayReturning({ kind: 'problem', problemType: 'backend-unreachable' })}
      />,
    );

    await screen.findByRole('alert');
    expect(screen.queryByTestId('position-placeholder-row')).toBeNull();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });
});
