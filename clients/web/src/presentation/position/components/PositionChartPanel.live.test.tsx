import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { PositionChartGateway } from '@/domain/position/PositionChartGateway';
import type { PositionChartResult } from '@/domain/position/PositionChartResult';
import {
  CHART_POSITION_ID,
  xagLongChart,
  xagLongChartWithLossStop,
} from '@/test/fixtures/positionChart';
import { createFakeChartRenderer } from '@/test/fakes/fakeChartRenderer';
import type { ChartRendererFactory } from '@/presentation/position/chart/ChartRenderer';
import { CHART_POLL_INTERVAL_MS } from '@/presentation/position/hooks/usePositionChartViewModel';
import { PositionChartPanel } from '@/presentation/position/components/PositionChartPanel';

type ThemeListener = () => void;

function stubColorScheme() {
  const listeners: ThemeListener[] = [];
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: false,
    media: query,
    addEventListener: (_type: string, listener: ThemeListener) => listeners.push(listener),
    removeEventListener: (_type: string, listener: ThemeListener) => {
      const index = listeners.indexOf(listener);
      if (index >= 0) listeners.splice(index, 1);
    },
    addListener: (listener: ThemeListener) => listeners.push(listener),
    removeListener: () => {},
    dispatchEvent: () => true,
    onchange: null,
  }));
  return () => listeners.slice().forEach((listener) => listener());
}

function renderLive(...results: PositionChartResult[]) {
  const fetchChart = vi.fn<PositionChartGateway['fetchChart']>();
  results.forEach((result) => fetchChart.mockResolvedValueOnce(result));
  const last = results.at(-1);
  if (last) fetchChart.mockResolvedValue(last);
  const renderer = createFakeChartRenderer();
  const createRenderer = vi.fn<ChartRendererFactory>(() => renderer);
  const view = render(
    <PositionChartPanel
      positionId={CHART_POSITION_ID}
      gateway={{ fetchChart }}
      createRenderer={createRenderer}
    />,
  );
  return { renderer, fetchChart, ...view };
}

async function tick(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
}

describe('PositionChartPanel Live-Update', () => {
  beforeEach(() => vi.useFakeTimers({ shouldAdvanceTime: true }));
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('setzt Kerzen beim ersten Mal mit setCandles, danach nur mit updateCandles', async () => {
    const { renderer } = renderLive({ kind: 'ok', chart: xagLongChart });
    await tick(0);
    expect(renderer.setCandles).toHaveBeenCalledTimes(1);

    await tick(CHART_POLL_INTERVAL_MS);
    expect(renderer.setCandles).toHaveBeenCalledTimes(1);
    expect(renderer.updateCandles).toHaveBeenCalledTimes(1);
  });

  it('setzt Price Lines nicht neu, wenn Entry/SL/TP unverändert sind', async () => {
    const { renderer } = renderLive({ kind: 'ok', chart: xagLongChart });
    await tick(0);
    expect(renderer.setPriceLines).toHaveBeenCalledTimes(1);
    await tick(CHART_POLL_INTERVAL_MS);
    expect(renderer.setPriceLines).toHaveBeenCalledTimes(1);
  });

  it('setzt Price Lines neu, wenn sich SL/TP geändert haben', async () => {
    const { renderer } = renderLive(
      { kind: 'ok', chart: xagLongChart },
      { kind: 'ok', chart: xagLongChartWithLossStop },
    );
    await tick(0);
    await tick(CHART_POLL_INTERVAL_MS);
    expect(renderer.setPriceLines).toHaveBeenCalledTimes(2);
  });

  it('ruft bei Intervallwechsel wieder setCandles auf (fitContent)', async () => {
    const { renderer } = renderLive({ kind: 'ok', chart: xagLongChart });
    await tick(0);
    await act(async () => {
      await userEvent.setup({ advanceTimers: vi.advanceTimersByTime }).click(
        screen.getByRole('button', { name: '1h' }),
      );
    });
    await waitFor(() => expect(renderer.setCandles).toHaveBeenCalledTimes(2));
  });

  it('ruft applyTheme beim Wechsel des System-Designs auf', async () => {
    const emitThemeChange = stubColorScheme();
    const { renderer } = renderLive({ kind: 'ok', chart: xagLongChart });
    await tick(0);
    expect(renderer.applyTheme).not.toHaveBeenCalled();

    act(() => emitThemeChange());
    expect(renderer.applyTheme).toHaveBeenCalledTimes(1);
  });

  it('meldet den Theme-Listener beim Unmount ab', async () => {
    const emitThemeChange = stubColorScheme();
    const { renderer, unmount } = renderLive({ kind: 'ok', chart: xagLongChart });
    await tick(0);
    unmount();
    emitThemeChange();
    expect(renderer.applyTheme).not.toHaveBeenCalled();
  });
});
