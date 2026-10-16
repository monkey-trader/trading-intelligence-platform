import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { OpenPositionsGateway } from '@/domain/position/OpenPositionsGateway';
import type { OpenPositionsResult } from '@/domain/position/OpenPositionsResult';
import { emptySnapshot, sampleSnapshot } from '@/test/fixtures/openPositions';
import { mapOpenPositionsProblem } from '@/presentation/position/errorMapper';
import { OpenPositionsPage } from '@/presentation/position/components/OpenPositionsPage';

/** Fake-Gateway, das die Ergebnisse der Reihe nach liefert; danach immer das letzte. */
function createSequenceGateway(...results: readonly OpenPositionsResult[]) {
  const fetchOpenPositions = vi.fn<OpenPositionsGateway['fetchOpenPositions']>();
  results.forEach((result) => fetchOpenPositions.mockResolvedValueOnce(result));
  const lastResult = results.at(-1);
  if (lastResult) fetchOpenPositions.mockResolvedValue(lastResult);
  const gateway: OpenPositionsGateway = { fetchOpenPositions };
  return { gateway, fetchOpenPositions };
}

function createPendingGateway(): OpenPositionsGateway {
  return { fetchOpenPositions: () => new Promise<OpenPositionsResult>(() => {}) };
}

describe('OpenPositionsPage', () => {
  it('zeigt einen Lade-Zustand, solange die erste Antwort aussteht', () => {
    render(<OpenPositionsPage gateway={createPendingGateway()} />);

    expect(screen.getByText(/lade/i)).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('zeigt die Tabelle mit allen Positionen und die Summe', async () => {
    const { gateway } = createSequenceGateway({ kind: 'ok', snapshot: sampleSnapshot });

    render(<OpenPositionsPage gateway={gateway} />);

    const table = await screen.findByRole('table', { name: /offene positionen/i });
    expect(within(table).getAllByRole('row')).toHaveLength(3);
    expect(within(table).getByText('SUI-USDT')).toBeInTheDocument();
    expect(within(table).getByText('NCCOGOLD2USD-USDT')).toBeInTheDocument();
    expect(screen.getByText(/18\.75/)).toBeInTheDocument();
    expect(screen.getByText(/2 Positionen/)).toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('meldet, wenn keine Position offen ist', async () => {
    const { gateway } = createSequenceGateway({ kind: 'ok', snapshot: emptySnapshot });

    render(<OpenPositionsPage gateway={gateway} />);

    expect(await screen.findByText('Keine offenen Positionen.')).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('zeigt einen Fehler ohne Daten als Alert', async () => {
    const { gateway } = createSequenceGateway({ kind: 'problem', problemType: 'not-configured' });

    render(<OpenPositionsPage gateway={gateway} />);

    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent(mapOpenPositionsProblem('not-configured'));
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('nennt bei rate-limited die Wartezeit', async () => {
    const { gateway } = createSequenceGateway({
      kind: 'problem',
      problemType: 'rate-limited',
      retryAfterSeconds: 30,
    });

    render(<OpenPositionsPage gateway={gateway} />);

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'BingX ausgelastet, neuer Versuch in 30 s.',
    );
  });

  it('lässt im Fehlerzustand ohne Daten einen neuen Versuch zu', async () => {
    const user = userEvent.setup();
    const { gateway, fetchOpenPositions } = createSequenceGateway(
      { kind: 'problem', problemType: 'backend-unreachable' },
      { kind: 'ok', snapshot: sampleSnapshot },
    );
    render(<OpenPositionsPage gateway={gateway} />);
    await screen.findByRole('alert');

    await user.click(screen.getByRole('button', { name: /aktualisieren|erneut versuchen/i }));

    expect(await screen.findByRole('table', { name: /offene positionen/i })).toBeInTheDocument();
    expect(fetchOpenPositions).toHaveBeenCalledTimes(2);
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('behält bei einem Fehler nach Daten die Tabelle und zeigt einen Warnhinweis', async () => {
    const user = userEvent.setup();
    const { gateway } = createSequenceGateway(
      { kind: 'ok', snapshot: sampleSnapshot },
      { kind: 'problem', problemType: 'unreachable' },
    );
    render(<OpenPositionsPage gateway={gateway} />);
    await screen.findByRole('table', { name: /offene positionen/i });

    await user.click(screen.getByRole('button', { name: /aktualisieren/i }));

    const warning = await screen.findByText(mapOpenPositionsProblem('unreachable'), {
      exact: false,
    });
    expect(warning.closest('[role="status"], [role="alert"]')).not.toBeNull();
    expect(screen.getByRole('table', { name: /offene positionen/i })).toBeInTheDocument();
    expect(screen.getByText('SUI-USDT')).toBeInTheDocument();
  });

  it('lädt beim Klick auf Aktualisieren neu', async () => {
    const user = userEvent.setup();
    const { gateway, fetchOpenPositions } = createSequenceGateway(
      { kind: 'ok', snapshot: sampleSnapshot },
      { kind: 'ok', snapshot: emptySnapshot },
    );
    render(<OpenPositionsPage gateway={gateway} />);
    await screen.findByRole('table', { name: /offene positionen/i });

    await user.click(screen.getByRole('button', { name: /aktualisieren/i }));

    expect(await screen.findByText('Keine offenen Positionen.')).toBeInTheDocument();
    expect(fetchOpenPositions).toHaveBeenCalledTimes(2);
  });
});
