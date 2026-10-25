import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { OpenPosition } from '@/domain/position/OpenPosition';
import type { InitialStopLossGateway } from '@/domain/position/InitialStopLossGateway';
import { goldShortPosition, suiLongPosition } from '@/test/fixtures/openPositions';
import { OpenPositionsTable } from '@/presentation/position/components/OpenPositionsTable';

// Erwartet: OpenPositionsTable-Props `initialStopLossGateway?: InitialStopLossGateway` und
// `onInitialStopLossChanged?: () => void` (wird nach erfolgreichem Speichern/Zurücksetzen aufgerufen,
// z.B. mit refresh() des View-Models). Zelle "Einstieg / SL" (Control in der SL-Zeile):
//  - unbekanntes Risiko (levels fehlt/leer): Button "Ursprünglichen SL eintragen für <Symbol>"
//  - Klick öffnet Feld (Label "Ursprünglicher SL", type text, inputmode decimal) mit "Speichern"/"Abbrechen"
//  - Source MANUAL: Text "manuell" und Button "Zurücksetzen"; Source AUTO: kein Button

const levels = [
  { kind: 'TWO_R', rMultiple: 2, price: '1.0800', percent: '12.5' },
  { kind: 'THREE_R', rMultiple: 3, price: '1.1400', percent: '18.75' },
] as const;

function fakeGateway(overrides: Partial<InitialStopLossGateway> = {}) {
  const setInitialStopLoss = vi
    .fn<InitialStopLossGateway['setInitialStopLoss']>()
    .mockResolvedValue({ kind: 'ok' });
  const clearInitialStopLoss = vi
    .fn<InitialStopLossGateway['clearInitialStopLoss']>()
    .mockResolvedValue({ kind: 'ok' });
  const gateway: InitialStopLossGateway = {
    setInitialStopLoss,
    clearInitialStopLoss,
    ...overrides,
  };
  return { gateway, setInitialStopLoss, clearInitialStopLoss };
}

function renderRow(position: OpenPosition, gateway = fakeGateway(), onChanged = vi.fn()) {
  render(
    <OpenPositionsTable
      positions={[position]}
      initialStopLossGateway={gateway.gateway}
      onInitialStopLossChanged={onChanged}
    />,
  );
  const cell = document.querySelector<HTMLElement>('[data-label="Einstieg / SL"]');
  if (!cell) throw new Error('Einstieg / SL-Zelle fehlt');
  return { cell, gateway, onChanged };
}

const unknownRisk: OpenPosition = {
  ...suiLongPosition,
  levels: undefined,
  initialStopLossSource: null,
};
const entryButton = (symbol: string) =>
  screen.getByRole('button', { name: `Ursprünglichen SL eintragen für ${symbol}` });

beforeEach(() => window.localStorage.clear());

describe('Ursprünglichen SL eintragen: Sichtbarkeit', () => {
  it('Button bei unbekanntem Risiko', () => {
    renderRow(unknownRisk);
    expect(entryButton('SUI-USDT')).toBeInTheDocument();
  });

  it('kein Button bei bekanntem Risiko (AUTO)', () => {
    renderRow({ ...suiLongPosition, levels, initialStopLossSource: 'AUTO' });
    expect(
      screen.queryByRole('button', { name: /Ursprünglichen SL eintragen/ }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('manuell')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Zurücksetzen' })).not.toBeInTheDocument();
  });

  it('ohne Gateway gibt es keinen Button', () => {
    render(<OpenPositionsTable positions={[unknownRisk]} />);
    expect(
      screen.queryByRole('button', { name: /Ursprünglichen SL eintragen/ }),
    ).not.toBeInTheDocument();
  });

  it('MANUAL: Text "manuell" und Button "Zurücksetzen", kein Eintragen-Button', () => {
    const { cell } = renderRow({ ...suiLongPosition, levels, initialStopLossSource: 'MANUAL' });
    expect(within(cell).getByText('manuell')).toBeInTheDocument();
    expect(within(cell).getByRole('button', { name: 'Zurücksetzen' })).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /Ursprünglichen SL eintragen/ }),
    ).not.toBeInTheDocument();
  });
});

describe('Ursprünglichen SL eintragen: Eingabe', () => {
  it('öffnet ein Textfeld mit inputmode decimal, Speichern und Abbrechen', async () => {
    renderRow(unknownRisk);
    await userEvent.click(entryButton('SUI-USDT'));

    const input = screen.getByLabelText('Ursprünglicher SL');
    expect(input).toHaveAttribute('type', 'text');
    expect(input).toHaveAttribute('inputmode', 'decimal');
    expect(screen.getByRole('button', { name: 'Speichern' })).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Abbrechen' }));
    expect(screen.queryByLabelText('Ursprünglicher SL')).not.toBeInTheDocument();
  });

  it('speichert: Gateway mit positionId und Wert, danach Refresh-Callback', async () => {
    const { gateway, onChanged } = renderRow(unknownRisk);
    await userEvent.click(entryButton('SUI-USDT'));
    await userEvent.type(screen.getByLabelText('Ursprünglicher SL'), '0.90');
    await userEvent.click(screen.getByRole('button', { name: 'Speichern' }));

    await waitFor(() => expect(onChanged).toHaveBeenCalledTimes(1));
    expect(gateway.setInitialStopLoss).toHaveBeenCalledWith('9100000000000000001', '0.90');
    expect(screen.queryByLabelText('Ursprünglicher SL')).not.toBeInTheDocument();
  });

  it.each([
    ['LONG', unknownRisk, 'abc', /zahl/i],
    ['LONG', unknownRisk, '0', /größer als 0|> ?0|positiv/i],
    ['LONG', unknownRisk, '1.00', /Verlustseite/],
    ['LONG', unknownRisk, '0.96', /Verlustseite/],
    [
      'SHORT',
      { ...goldShortPosition, levels: undefined, initialStopLossSource: null },
      '1',
      /Verlustseite/,
    ],
  ] as const)(
    '%s: ungültige Eingabe "%s" zeigt Fehler und ruft das Gateway nicht',
    async (_side, position, raw, message) => {
      const { gateway, onChanged } = renderRow(position);
      await userEvent.click(entryButton(position.symbol));
      await userEvent.type(screen.getByLabelText('Ursprünglicher SL'), raw);
      await userEvent.click(screen.getByRole('button', { name: 'Speichern' }));

      expect(await screen.findByRole('alert')).toHaveTextContent(message);
      expect(gateway.setInitialStopLoss).not.toHaveBeenCalled();
      expect(onChanged).not.toHaveBeenCalled();
    },
  );

  it('zeigt den Serverfehler (422) und lässt das Feld offen', async () => {
    const gateway = fakeGateway({
      setInitialStopLoss: vi
        .fn()
        .mockResolvedValue({ kind: 'problem', problemType: 'invalid-stop-loss' }),
    });
    const { onChanged } = renderRow(unknownRisk, gateway);
    await userEvent.click(entryButton('SUI-USDT'));
    await userEvent.type(screen.getByLabelText('Ursprünglicher SL'), '0.9');
    await userEvent.click(screen.getByRole('button', { name: 'Speichern' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'SL muss auf der Verlustseite liegen',
    );
    expect(screen.getByLabelText('Ursprünglicher SL')).toBeInTheDocument();
    expect(onChanged).not.toHaveBeenCalled();
  });

  it('zeigt "Backend nicht erreichbar" bei Netzwerkfehler', async () => {
    const gateway = fakeGateway({
      setInitialStopLoss: vi
        .fn()
        .mockResolvedValue({ kind: 'problem', problemType: 'backend-unreachable' }),
    });
    renderRow(unknownRisk, gateway);
    await userEvent.click(entryButton('SUI-USDT'));
    await userEvent.type(screen.getByLabelText('Ursprünglicher SL'), '0.9');
    await userEvent.click(screen.getByRole('button', { name: 'Speichern' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(/nicht erreichbar/i);
  });
});

describe('Zurücksetzen', () => {
  it('ruft clearInitialStopLoss mit der positionId und danach den Refresh-Callback', async () => {
    const { gateway, onChanged } = renderRow({
      ...suiLongPosition,
      levels,
      initialStopLossSource: 'MANUAL',
    });
    await userEvent.click(screen.getByRole('button', { name: 'Zurücksetzen' }));

    await waitFor(() => expect(onChanged).toHaveBeenCalledTimes(1));
    expect(gateway.clearInitialStopLoss).toHaveBeenCalledWith('9100000000000000001');
  });

  it('zeigt einen Fehler und ruft den Refresh-Callback nicht auf', async () => {
    const gateway = fakeGateway({
      clearInitialStopLoss: vi
        .fn()
        .mockResolvedValue({ kind: 'problem', problemType: 'backend-unreachable' }),
    });
    const { onChanged } = renderRow(
      { ...suiLongPosition, levels, initialStopLossSource: 'MANUAL' },
      gateway,
    );
    await userEvent.click(screen.getByRole('button', { name: 'Zurücksetzen' }));

    expect(await screen.findByRole('alert')).toBeInTheDocument();
    expect(onChanged).not.toHaveBeenCalled();
  });
});
