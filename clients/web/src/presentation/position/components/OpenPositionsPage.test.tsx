import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { PositionChartGateway } from '@/domain/position/PositionChartGateway';
import { xagLongChart } from '@/test/fixtures/positionChart';
import { createFakeChartRenderer } from '@/test/fakes/fakeChartRenderer';
import type { OpenPositionsGateway } from '@/domain/position/OpenPositionsGateway';
import type { OpenPositionsResult } from '@/domain/position/OpenPositionsResult';
import type { AssetIcon } from '@/domain/asset/AssetIcon';
import type { AssetIconsGateway } from '@/domain/asset/AssetIconsGateway';
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
    expect(within(table).getByText('SUI', { exact: false, selector: 'span' })).toBeInTheDocument();
    expect(within(table).getByText(/NCCOGOLD2USD/)).toBeInTheDocument();
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
    expect(screen.getByText('SUI', { exact: false, selector: 'span' })).toBeInTheDocument();
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

describe('OpenPositionsPage: Chart-Ansicht', () => {
  function renderWithChart() {
    const { gateway } = createSequenceGateway({ kind: 'ok', snapshot: sampleSnapshot });
    const fetchChart = vi.fn<PositionChartGateway['fetchChart']>().mockResolvedValue({
      kind: 'ok',
      chart: xagLongChart,
    });
    const renderer = createFakeChartRenderer();
    render(
      <OpenPositionsPage
        gateway={gateway}
        chartGateway={{ fetchChart }}
        createChartRenderer={() => renderer}
      />,
    );
    return { fetchChart, renderer };
  }

  it('zeigt anfangs kein Chart-Panel', async () => {
    renderWithChart();
    await screen.findByRole('table', { name: /offene positionen/i });

    expect(screen.queryByRole('heading', { name: /NCCOXAG2USD-USDT/ })).not.toBeInTheDocument();
  });

  it('zeigt nach Klick auf Chart das Panel unterhalb der Tabelle und lädt die gewählte Position', async () => {
    const user = userEvent.setup();
    const { fetchChart } = renderWithChart();
    const table = await screen.findByRole('table', { name: /offene positionen/i });

    await user.click(screen.getByRole('button', { name: /chart.*SUI/i }));

    const heading = await screen.findByRole('heading', { name: /NCCOXAG2USD-USDT/ });
    expect(fetchChart).toHaveBeenCalledWith(
      sampleSnapshot.positions.find((position) => position.symbol === 'SUI-USDT')?.positionId,
      '15m',
      300,
      expect.any(AbortSignal),
    );
    expect(table.compareDocumentPosition(heading) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('blendet das Panel bei erneutem Klick auf denselben Button aus und räumt auf', async () => {
    const user = userEvent.setup();
    const { renderer } = renderWithChart();
    await screen.findByRole('table', { name: /offene positionen/i });
    const chartButton = screen.getByRole('button', { name: /chart.*SUI/i });
    await user.click(chartButton);
    await screen.findByRole('heading', { name: /NCCOXAG2USD-USDT/ });

    await user.click(screen.getByRole('button', { name: /chart.*SUI/i }));

    expect(screen.queryByRole('heading', { name: /NCCOXAG2USD-USDT/ })).not.toBeInTheDocument();
    expect(renderer.destroy).toHaveBeenCalled();
  });

  it('blendet das Panel über Schließen aus', async () => {
    const user = userEvent.setup();
    renderWithChart();
    await screen.findByRole('table', { name: /offene positionen/i });
    await user.click(screen.getByRole('button', { name: /chart.*SUI/i }));
    await screen.findByRole('heading', { name: /NCCOXAG2USD-USDT/ });

    await user.click(screen.getByRole('button', { name: /schließen/i }));

    expect(screen.queryByRole('heading', { name: /NCCOXAG2USD-USDT/ })).not.toBeInTheDocument();
  });

  it('wechselt beim Klick auf eine andere Position das Chart, ohne zwei Panels zu zeigen', async () => {
    const user = userEvent.setup();
    const { fetchChart } = renderWithChart();
    await screen.findByRole('table', { name: /offene positionen/i });
    await user.click(screen.getByRole('button', { name: /chart.*SUI/i }));
    await screen.findByRole('heading', { name: /NCCOXAG2USD-USDT/ });

    await user.click(screen.getByRole('button', { name: /chart.*NCCOGOLD2USD/i }));

    await vi.waitFor(() => expect(fetchChart).toHaveBeenCalledTimes(2));
    expect(screen.getAllByRole('heading', { name: /NCCOXAG2USD-USDT/ })).toHaveLength(1);
  });
});

describe('OpenPositionsPage: Coin-Icons und Anzeigenamen', () => {
  const icons: readonly AssetIcon[] = [
    {
      symbol: 'SUI-USDT',
      baseAsset: 'SUI',
      displayName: 'SUI',
      iconUrl: 'https://coin-images.coingecko.com/coins/images/26375/large/sui.png',
    },
    { symbol: 'NCCOGOLD2USD-USDT', baseAsset: 'XAU', displayName: 'Gold', iconUrl: null },
  ];

  it('ruft die Icons einmal für alle unterschiedlichen Symbole ab und zeigt Anzeigenamen', async () => {
    const { gateway } = createSequenceGateway({ kind: 'ok', snapshot: sampleSnapshot });
    const fetchAssetIcons = vi.fn<AssetIconsGateway['fetchAssetIcons']>().mockResolvedValue(icons);

    render(<OpenPositionsPage gateway={gateway} assetIconsGateway={{ fetchAssetIcons }} />);

    const table = await screen.findByRole('table', { name: /offene positionen/i });
    expect(await within(table).findByText('Gold')).toBeInTheDocument();
    expect(table.textContent).not.toContain('-USDT');
    expect(fetchAssetIcons).toHaveBeenCalledTimes(1);
    const requested = fetchAssetIcons.mock.calls[0]?.[0] ?? [];
    expect([...requested].sort()).toEqual(
      [...new Set(sampleSnapshot.positions.map((position) => position.symbol))].sort(),
    );
  });

  it('zeigt bei Icon-Fehler (leeres Ergebnis) den Fallback-Namen ohne Fehlermeldung', async () => {
    const { gateway } = createSequenceGateway({ kind: 'ok', snapshot: sampleSnapshot });
    const fetchAssetIcons = vi.fn<AssetIconsGateway['fetchAssetIcons']>().mockResolvedValue([]);

    render(<OpenPositionsPage gateway={gateway} assetIconsGateway={{ fetchAssetIcons }} />);

    const table = await screen.findByRole('table', { name: /offene positionen/i });
    await waitFor(() => expect(fetchAssetIcons).toHaveBeenCalled());
    expect(within(table).getByText(/NCCOGOLD2USD/)).toBeInTheDocument();
    expect(table.textContent).not.toContain('-USDT');
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('ruft beim Re-Polling mit gleichen Symbolen die Icons nicht erneut ab', async () => {
    const { gateway, fetchOpenPositions } = createSequenceGateway({
      kind: 'ok',
      snapshot: sampleSnapshot,
    });
    const fetchAssetIcons = vi.fn<AssetIconsGateway['fetchAssetIcons']>().mockResolvedValue(icons);
    const user = userEvent.setup();

    render(<OpenPositionsPage gateway={gateway} assetIconsGateway={{ fetchAssetIcons }} />);
    await screen.findByRole('table', { name: /offene positionen/i });
    await user.click(screen.getByRole('button', { name: /aktualisieren/i }));
    await waitFor(() => expect(fetchOpenPositions.mock.calls.length).toBeGreaterThan(1));

    expect(fetchAssetIcons).toHaveBeenCalledTimes(1);
  });

  it('behält die Überschrift "Offene Positionen"', async () => {
    const { gateway } = createSequenceGateway({ kind: 'ok', snapshot: sampleSnapshot });
    render(<OpenPositionsPage gateway={gateway} />);
    expect(
      await screen.findByRole('heading', { level: 1, name: 'Offene Positionen' }),
    ).toBeInTheDocument();
  });
});
