import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { PositionChartGateway } from '@/domain/position/PositionChartGateway';
import type { PositionChartResult } from '@/domain/position/PositionChartResult';
import {
  CHART_POSITION_ID,
  xagChartHourly,
  xagChartWithThreeTakeProfits,
  xagChartWithoutProtection,
  xagLongChart,
  xagLongChartWithLossStop,
} from '@/test/fixtures/positionChart';
import { createFakeChartRenderer } from '@/test/fakes/fakeChartRenderer';
import type { ChartRendererFactory } from '@/presentation/position/chart/ChartRenderer';
import { mapPositionChartProblem } from '@/presentation/position/chartErrorMapper';
import { PositionChartPanel } from '@/presentation/position/components/PositionChartPanel';

// Zwischen Zahl und % darf ein (geschütztes) Leerzeichen stehen.
const SP = '[\\s\\u00a0\\u202f]?';

function gatewayReturning(...results: readonly PositionChartResult[]) {
  const fetchChart = vi.fn<PositionChartGateway['fetchChart']>();
  results.forEach((result) => fetchChart.mockResolvedValueOnce(result));
  const lastResult = results.at(-1);
  if (lastResult) fetchChart.mockResolvedValue(lastResult);
  return { gateway: { fetchChart } satisfies PositionChartGateway, fetchChart };
}

function renderPanel(gateway: PositionChartGateway, onClose?: () => void) {
  const renderer = createFakeChartRenderer();
  const createRenderer = vi.fn<ChartRendererFactory>(() => renderer);
  const view = render(
    <PositionChartPanel
      positionId={CHART_POSITION_ID}
      gateway={gateway}
      createRenderer={createRenderer}
      onClose={onClose}
    />,
  );
  return { renderer, createRenderer, ...view };
}

function lastPriceLines(renderer: ReturnType<typeof createFakeChartRenderer>) {
  return renderer.setPriceLines.mock.lastCall?.[0] ?? [];
}

describe('PositionChartPanel', () => {
  it('zeigt einen Ladezustand, solange die Antwort aussteht', () => {
    renderPanel({ fetchChart: () => new Promise(() => {}) });

    expect(screen.getByRole('status')).toHaveTextContent(/lade/i);
  });

  it('zeigt Symbol und Seite im Titel', async () => {
    const { gateway } = gatewayReturning({ kind: 'ok', chart: xagLongChart });
    renderPanel(gateway);

    const heading = await screen.findByRole('heading', { name: /NCCOXAG2USD-USDT/ });
    expect(heading).toHaveTextContent(/long/i);
  });

  it('bietet die Intervalle 5m, 15m, 1h, 4h, 1d an, 15m ist gewählt', async () => {
    const { gateway } = gatewayReturning({ kind: 'ok', chart: xagLongChart });
    renderPanel(gateway);
    await screen.findByRole('heading', { name: /NCCOXAG2USD-USDT/ });

    const group = screen.getByRole('group', { name: /intervall/i });
    const buttons = within(group).getAllByRole('button');
    expect(buttons.map((button) => button.textContent)).toEqual(['5m', '15m', '1h', '4h', '1d']);
    expect(within(group).getByRole('button', { name: '15m' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(within(group).getByRole('button', { name: '1h' })).toHaveAttribute(
      'aria-pressed',
      'false',
    );
  });

  it('lädt beim Klick auf ein Intervall neu und aktualisiert den Renderer', async () => {
    const user = userEvent.setup();
    const { gateway, fetchChart } = gatewayReturning(
      { kind: 'ok', chart: xagLongChart },
      { kind: 'ok', chart: xagChartHourly },
    );
    const { renderer } = renderPanel(gateway);
    await waitFor(() => expect(renderer.setCandles).toHaveBeenCalledTimes(1));

    await user.click(screen.getByRole('button', { name: '1h' }));

    await waitFor(() => expect(renderer.setCandles).toHaveBeenCalledTimes(2));
    expect(fetchChart).toHaveBeenLastCalledWith(
      CHART_POSITION_ID,
      '1h',
      300,
      expect.any(AbortSignal),
    );
    expect(renderer.setCandles.mock.lastCall?.[0]).toHaveLength(1);
    expect(screen.getByRole('button', { name: '1h' })).toHaveAttribute('aria-pressed', 'true');
  });

  it('zeigt Fehler als Alert mit der Meldung des Error-Mappers', async () => {
    const { gateway } = gatewayReturning({ kind: 'problem', problemType: 'not-found' });
    renderPanel(gateway);

    expect(await screen.findByRole('alert')).toHaveTextContent(
      mapPositionChartProblem('not-found'),
    );
  });

  it('ruft onClose beim Klick auf Schließen', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const { gateway } = gatewayReturning({ kind: 'ok', chart: xagLongChart });
    renderPanel(gateway, onClose);

    await user.click(await screen.findByRole('button', { name: /schließen/i }));

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  describe('Renderer-Port', () => {
    it('erzeugt den Renderer genau einmal und übergibt die Kerzen aufsteigend als Zahlen in Sekunden', async () => {
      const { gateway } = gatewayReturning({ kind: 'ok', chart: xagLongChart });
      const { renderer, createRenderer } = renderPanel(gateway);

      await waitFor(() => expect(renderer.setCandles).toHaveBeenCalled());
      expect(createRenderer).toHaveBeenCalledTimes(1);
      expect(createRenderer.mock.calls[0]?.[0]).toBeInstanceOf(HTMLElement);
      expect(renderer.setCandles.mock.lastCall?.[0]).toEqual([
        { time: 1791700000, open: 55.7, high: 55.9, low: 55.6, close: 55.8 },
        { time: 1791700900, open: 55.8, high: 56.1, low: 55.75, close: 56 },
      ]);
    });

    it('sortiert Kerzen aufsteigend, auch wenn sie absteigend ankommen', async () => {
      const reversed = { ...xagLongChart, candles: [...xagLongChart.candles].reverse() };
      const { gateway } = gatewayReturning({ kind: 'ok', chart: reversed });
      const { renderer } = renderPanel(gateway);

      await waitFor(() => expect(renderer.setCandles).toHaveBeenCalled());
      const times = (renderer.setCandles.mock.lastCall?.[0] ?? []).map((candle) => candle.time);
      expect(times).toEqual([1791700000, 1791700900]);
    });

    it('übergibt Entry (neutral), SL (stop) und TP (target) als Preislinien mit Abstand in Prozent', async () => {
      const { gateway } = gatewayReturning({ kind: 'ok', chart: xagLongChart });
      const { renderer } = renderPanel(gateway);

      await waitFor(() => expect(renderer.setPriceLines).toHaveBeenCalled());
      const lines = lastPriceLines(renderer);
      expect(lines).toHaveLength(3);
      const entry = lines.find((line) => line.kind === 'entry');
      const stop = lines.find((line) => line.kind === 'stop');
      const target = lines.find((line) => line.kind === 'target');
      expect(entry).toMatchObject({ price: 55.75 });
      expect(entry?.title).toMatch(/entry/i);
      expect(entry?.title).toMatch(/55\.75/);
      expect(entry?.title).not.toMatch(/%/);
      expect(stop).toMatchObject({ price: 55.8 });
      expect(stop?.title).toMatch(new RegExp(`55\\.8.*\\+0\\.09${SP}%`));
      expect(target).toMatchObject({ price: 63.36 });
      expect(target?.title).toMatch(new RegExp(`63\\.36.*\\+13\\.65${SP}%`));
    });

    it('zeigt negative Abstände mit Minuszeichen', async () => {
      const { gateway } = gatewayReturning({ kind: 'ok', chart: xagLongChartWithLossStop });
      const { renderer } = renderPanel(gateway);

      await waitFor(() => expect(renderer.setPriceLines).toHaveBeenCalled());
      const stop = lastPriceLines(renderer).find((line) => line.kind === 'stop');
      expect(stop?.title).toMatch(new RegExp(`51\\.54.*[-−]7\\.55${SP}%`));
    });

    it('übergibt ohne SL/TP nur die Entry-Linie und zeigt keinen Fehler', async () => {
      const { gateway } = gatewayReturning({ kind: 'ok', chart: xagChartWithoutProtection });
      const { renderer } = renderPanel(gateway);

      await waitFor(() => expect(renderer.setPriceLines).toHaveBeenCalled());
      const lines = lastPriceLines(renderer);
      expect(lines.map((line) => line.kind)).toEqual(['entry']);
      expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    });

    it('übergibt jeden Teil-TP als eigene Linie', async () => {
      const { gateway } = gatewayReturning({ kind: 'ok', chart: xagChartWithThreeTakeProfits });
      const { renderer } = renderPanel(gateway);

      await waitFor(() => expect(renderer.setPriceLines).toHaveBeenCalled());
      const targets = lastPriceLines(renderer).filter((line) => line.kind === 'target');
      expect(targets.map((line) => line.price)).toEqual([58, 60.5, 63.36]);
      expect(targets[0]?.title).toMatch(new RegExp(`\\+4\\.04${SP}%`));
      expect(targets[1]?.title).toMatch(new RegExp(`\\+8\\.52${SP}%`));
    });

    it('übergibt den ENTRY-Marker', async () => {
      const { gateway } = gatewayReturning({ kind: 'ok', chart: xagLongChart });
      const { renderer } = renderPanel(gateway);

      await waitFor(() => expect(renderer.setMarkers).toHaveBeenCalled());
      expect(renderer.setMarkers.mock.lastCall?.[0]).toEqual([
        expect.objectContaining({ kind: 'entry', time: 1791700000, price: 55.75 }),
      ]);
    });

    it('aktualisiert Preislinien und Marker beim Intervallwechsel mit den neuen Daten', async () => {
      const user = userEvent.setup();
      const { gateway } = gatewayReturning(
        { kind: 'ok', chart: xagLongChart },
        { kind: 'ok', chart: { ...xagChartHourly, takeProfit: [] } },
      );
      const { renderer } = renderPanel(gateway);
      await waitFor(() => expect(renderer.setPriceLines).toHaveBeenCalledTimes(1));

      await user.click(screen.getByRole('button', { name: '1h' }));

      await waitFor(() => expect(renderer.setPriceLines).toHaveBeenCalledTimes(2));
      expect(lastPriceLines(renderer).map((line) => line.kind)).toEqual(['entry', 'stop']);
    });

    it('räumt den Renderer beim Unmount genau einmal auf', async () => {
      const { gateway } = gatewayReturning({ kind: 'ok', chart: xagLongChart });
      const { renderer, unmount } = renderPanel(gateway);
      await waitFor(() => expect(renderer.setCandles).toHaveBeenCalled());

      unmount();

      expect(renderer.destroy).toHaveBeenCalledTimes(1);
    });
  });

  describe('Legende', () => {
    it('zeigt Entry, SL und TP mit Preis und Abstand', async () => {
      const { gateway } = gatewayReturning({ kind: 'ok', chart: xagLongChart });
      renderPanel(gateway);

      const legend = await screen.findByRole('list', { name: /legende/i });
      const items = within(legend).getAllByRole('listitem');
      expect(items).toHaveLength(3);
      expect(legend).toHaveTextContent(/entry.*55\.75/i);
      expect(legend).toHaveTextContent(new RegExp(`63\\.36.*\\+13\\.65${SP}%`));
    });

    it('zeigt SL mit Menge 0.0000 als ganze Position, nie als 0', async () => {
      const { gateway } = gatewayReturning({ kind: 'ok', chart: xagLongChart });
      renderPanel(gateway);

      const legend = await screen.findByRole('list', { name: /legende/i });
      const stopItem = within(legend)
        .getAllByRole('listitem')
        .find((item) => /sl|stop/i.test(item.textContent ?? ''));
      expect(stopItem).toHaveTextContent(/ganze position/i);
      expect(stopItem?.textContent).not.toMatch(/(^|[^\d.])0(\.0+)?([^\d.%]|$)/);
    });

    it('zeigt die Menge eines Teil-TPs', async () => {
      const { gateway } = gatewayReturning({ kind: 'ok', chart: xagLongChart });
      renderPanel(gateway);

      const legend = await screen.findByRole('list', { name: /legende/i });
      expect(legend).toHaveTextContent(/2\.9483/);
    });

    it('listet drei Teil-TPs einzeln', async () => {
      const { gateway } = gatewayReturning({ kind: 'ok', chart: xagChartWithThreeTakeProfits });
      renderPanel(gateway);

      const legend = await screen.findByRole('list', { name: /legende/i });
      expect(within(legend).getAllByRole('listitem')).toHaveLength(5);
    });

    it('zeigt ohne SL/TP nur den Entry', async () => {
      const { gateway } = gatewayReturning({ kind: 'ok', chart: xagChartWithoutProtection });
      renderPanel(gateway);

      const legend = await screen.findByRole('list', { name: /legende/i });
      expect(within(legend).getAllByRole('listitem')).toHaveLength(1);
    });
  });
});
