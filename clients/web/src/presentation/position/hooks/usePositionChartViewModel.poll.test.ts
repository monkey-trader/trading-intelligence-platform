import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { PositionChartGateway } from '@/domain/position/PositionChartGateway';
import type { PositionChartResult } from '@/domain/position/PositionChartResult';
import { CHART_POSITION_ID, xagChartHourly, xagLongChart } from '@/test/fixtures/positionChart';
import {
  CHART_POLL_INTERVAL_MS,
  usePositionChartViewModel,
} from '@/presentation/position/hooks/usePositionChartViewModel';

const ok = (chart: typeof xagLongChart): PositionChartResult => ({ kind: 'ok', chart });
const problem: PositionChartResult = { kind: 'problem', problemType: 'backend-unreachable' } as PositionChartResult;

function setup(...results: Array<PositionChartResult | Error>) {
  const fetchChart = vi.fn<PositionChartGateway['fetchChart']>();
  results.forEach((result) =>
    result instanceof Error
      ? fetchChart.mockRejectedValueOnce(result)
      : fetchChart.mockResolvedValueOnce(result),
  );
  const last = results.at(-1);
  if (last && !(last instanceof Error)) fetchChart.mockResolvedValue(last);
  const gateway: PositionChartGateway = { fetchChart };
  return { gateway, fetchChart };
}

async function tick(ms: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
}

function setHidden(hidden: boolean) {
  Object.defineProperty(document, 'hidden', { configurable: true, get: () => hidden });
  document.dispatchEvent(new Event('visibilitychange'));
}

describe('usePositionChartViewModel Polling', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => {
    setHidden(false);
    vi.useRealTimers();
  });

  it('exportiert ein 5-Sekunden-Intervall', () => {
    expect(CHART_POLL_INTERVAL_MS).toBe(5000);
  });

  it('ruft nach dem ersten Laden alle 5 s erneut ab und ersetzt den Chart', async () => {
    const { gateway, fetchChart } = setup(ok(xagLongChart), ok(xagChartHourly));
    const { result } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));
    await tick(0);
    expect(fetchChart).toHaveBeenCalledTimes(1);
    expect(result.current.chart).toEqual(xagLongChart);

    await tick(CHART_POLL_INTERVAL_MS - 1);
    expect(fetchChart).toHaveBeenCalledTimes(1);
    await tick(1);
    expect(fetchChart).toHaveBeenCalledTimes(2);
    expect(result.current.chart).toEqual(xagChartHourly);

    await tick(CHART_POLL_INTERVAL_MS);
    expect(fetchChart).toHaveBeenCalledTimes(3);
  });

  it('behält bei Poll-Fehler den letzten Chart, bleibt ready und pollt weiter', async () => {
    const { gateway, fetchChart } = setup(ok(xagLongChart), problem, ok(xagChartHourly));
    const { result } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));
    await tick(0);

    await tick(CHART_POLL_INTERVAL_MS);
    expect(fetchChart).toHaveBeenCalledTimes(2);
    expect(result.current.status).toBe('ready');
    expect(result.current.chart).toEqual(xagLongChart);
    expect(result.current.errorMessage).toBeUndefined();

    await tick(CHART_POLL_INTERVAL_MS);
    expect(result.current.chart).toEqual(xagChartHourly);
  });

  it('behält bei geworfenem Poll-Fehler den letzten Chart', async () => {
    const { gateway } = setup(ok(xagLongChart), new Error('netz'));
    const { result } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));
    await tick(0);
    await tick(CHART_POLL_INTERVAL_MS);
    expect(result.current.status).toBe('ready');
    expect(result.current.chart).toEqual(xagLongChart);
  });

  it('bleibt beim Fehler im allerersten Laden im Status error', async () => {
    const { gateway } = setup(problem);
    const { result } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));
    await tick(0);
    expect(result.current.status).toBe('error');
    expect(result.current.chart).toBeUndefined();
  });

  it('pausiert bei document.hidden und holt beim Sichtbarwerden sofort nach', async () => {
    const { gateway, fetchChart } = setup(ok(xagLongChart));
    renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));
    await tick(0);

    act(() => setHidden(true));
    await tick(CHART_POLL_INTERVAL_MS * 3);
    expect(fetchChart).toHaveBeenCalledTimes(1);

    await act(async () => setHidden(false));
    await tick(0);
    expect(fetchChart).toHaveBeenCalledTimes(2);
    await tick(CHART_POLL_INTERVAL_MS);
    expect(fetchChart).toHaveBeenCalledTimes(3);
  });

  it('beendet beim Intervallwechsel das Polling der alten Anfrage', async () => {
    const { gateway, fetchChart } = setup(ok(xagLongChart));
    const { result } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));
    await tick(0);
    act(() => result.current.selectInterval('1h'));
    await tick(0);
    expect(fetchChart).toHaveBeenCalledTimes(2);
    fetchChart.mockClear();

    await tick(CHART_POLL_INTERVAL_MS);
    expect(fetchChart).toHaveBeenCalledTimes(1);
    expect(fetchChart.mock.calls.every(([, interval]) => interval === '1h')).toBe(true);
  });

  it('verwirft späte Poll-Antworten nach Intervallwechsel', async () => {
    const { gateway, fetchChart } = setup(ok(xagLongChart));
    const { result } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));
    await tick(0);
    let resolveLate: (value: PositionChartResult) => void = () => {};
    fetchChart.mockImplementationOnce(() => new Promise((resolve) => (resolveLate = resolve)));
    await tick(CHART_POLL_INTERVAL_MS);
    fetchChart.mockImplementation(() => new Promise(() => {}));
    act(() => result.current.selectInterval('1h'));
    await act(async () => resolveLate(ok(xagChartHourly)));
    expect(result.current.status).toBe('loading');
    expect(result.current.chart).toBeUndefined();
  });

  it('räumt beim Unmount den Timer auf und bricht ab', async () => {
    const { gateway, fetchChart } = setup(ok(xagLongChart));
    const { unmount } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));
    await tick(0);
    unmount();
    expect(vi.getTimerCount()).toBe(0);
    await tick(CHART_POLL_INTERVAL_MS * 2);
    expect(fetchChart).toHaveBeenCalledTimes(1);
  });
});
