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

    const dialog = screen.getByRole('dialog', { name: 'TP1 bis TP3' });
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

  it('ändert den Namen der Spalte nicht (Kopf bleibt "TPs (je 33 %)")', () => {
    render(<OpenPositionsTable positions={[suiLongPosition]} />);

    expect(screen.getByRole('columnheader', { name: 'TPs (je 33 %)' })).toBeInTheDocument();
  });
});
