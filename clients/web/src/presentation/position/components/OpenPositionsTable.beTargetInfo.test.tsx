import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it } from 'vitest';
import { suiLongPosition } from '@/test/fixtures/openPositions';
import { OpenPositionsTable } from '@/presentation/position/components/OpenPositionsTable';

beforeEach(() => window.localStorage.clear());

describe('OpenPositionsTable: Erklärung zu den TPs', () => {
  it('öffnet über (i) im Spaltenkopf einen Dialog mit der Erklärung', async () => {
    render(<OpenPositionsTable positions={[suiLongPosition]} />);

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    await userEvent.click(screen.getAllByRole('button', { name: 'Erklärung zu den TPs' })[0]!);

    const dialog = screen.getByRole('dialog', { name: 'BE und TP1' });
    expect(dialog).toHaveTextContent('2R');
    expect(dialog).toHaveTextContent('3R');
    expect(dialog).toHaveTextContent('Break-even');
    expect(dialog).toHaveTextContent('33 %');
  });

  it('schließt den Dialog wieder', async () => {
    render(<OpenPositionsTable positions={[suiLongPosition]} />);
    await userEvent.click(screen.getAllByRole('button', { name: 'Erklärung zu den TPs' })[0]!);

    await userEvent.click(screen.getByRole('button', { name: 'Schließen' }));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('benennt die Spalte (Kopf heißt "BE / TP1")', () => {
    render(<OpenPositionsTable positions={[suiLongPosition]} />);

    expect(screen.getByRole('columnheader', { name: 'BE / TP1' })).toBeInTheDocument();
  });
});
