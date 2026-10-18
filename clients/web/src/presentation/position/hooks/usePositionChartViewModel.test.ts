import { act, renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { PositionChartGateway } from '@/domain/position/PositionChartGateway';
import type { PositionChartResult } from '@/domain/position/PositionChartResult';
import { CHART_POSITION_ID, xagChartHourly, xagLongChart } from '@/test/fixtures/positionChart';
import { mapPositionChartProblem } from '@/presentation/position/chartErrorMapper';
import { usePositionChartViewModel } from '@/presentation/position/hooks/usePositionChartViewModel';

function createGateway(...results: readonly PositionChartResult[]) {
  const fetchChart = vi.fn<PositionChartGateway['fetchChart']>();
  results.forEach((result) => fetchChart.mockResolvedValueOnce(result));
  const lastResult = results.at(-1);
  if (lastResult) fetchChart.mockResolvedValue(lastResult);
  const gateway: PositionChartGateway = { fetchChart };
  return { gateway, fetchChart };
}

describe('usePositionChartViewModel', () => {
  it('startet im Ladezustand mit dem Standard-Intervall 15m', () => {
    const gateway: PositionChartGateway = { fetchChart: () => new Promise(() => {}) };

    const { result } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));

    expect(result.current.status).toBe('loading');
    expect(result.current.interval).toBe('15m');
    expect(result.current.chart).toBeUndefined();
  });

  it('lädt den Chart mit Standard-Intervall und -Limit', async () => {
    const { gateway, fetchChart } = createGateway({ kind: 'ok', chart: xagLongChart });

    const { result } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));

    await waitFor(() => expect(result.current.status).toBe('ready'));
    expect(result.current.chart).toEqual(xagLongChart);
    expect(result.current.errorMessage).toBeUndefined();
    expect(fetchChart).toHaveBeenCalledWith(CHART_POSITION_ID, '15m', 300, expect.any(AbortSignal));
  });

  it('lädt beim Intervallwechsel neu', async () => {
    const { gateway, fetchChart } = createGateway(
      { kind: 'ok', chart: xagLongChart },
      { kind: 'ok', chart: xagChartHourly },
    );
    const { result } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));
    await waitFor(() => expect(result.current.status).toBe('ready'));

    act(() => result.current.selectInterval('1h'));

    await waitFor(() => expect(result.current.chart).toEqual(xagChartHourly));
    expect(result.current.interval).toBe('1h');
    expect(fetchChart).toHaveBeenLastCalledWith(
      CHART_POSITION_ID,
      '1h',
      300,
      expect.any(AbortSignal),
    );
    expect(fetchChart).toHaveBeenCalledTimes(2);
  });

  it('bricht den laufenden Request beim Intervallwechsel ab und verwirft die späte Antwort', async () => {
    const signals: AbortSignal[] = [];
    const resolvers: Array<(result: PositionChartResult) => void> = [];
    const gateway: PositionChartGateway = {
      fetchChart: (_id, _interval, _limit, signal) => {
        signals.push(signal);
        return new Promise<PositionChartResult>((resolve) => resolvers.push(resolve));
      },
    };
    const { result } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));

    act(() => result.current.selectInterval('1h'));
    await act(async () => {
      resolvers[1]?.({ kind: 'ok', chart: xagChartHourly });
      resolvers[0]?.({ kind: 'ok', chart: xagLongChart });
    });

    expect(signals[0]?.aborted).toBe(true);
    expect(result.current.chart).toEqual(xagChartHourly);
  });

  it.each([
    ['not-found', undefined],
    ['upstream-error', undefined],
    ['rate-limited', 20],
  ] as const)(
    'zeigt bei Problem %s die Meldung des Error-Mappers',
    async (problemType, retryAfterSeconds) => {
      const { gateway } = createGateway({ kind: 'problem', problemType, retryAfterSeconds });

      const { result } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));

      await waitFor(() => expect(result.current.status).toBe('error'));
      expect(result.current.errorMessage).toBe(
        mapPositionChartProblem(problemType, retryAfterSeconds),
      );
      expect(result.current.chart).toBeUndefined();
    },
  );

  it('behandelt eine geworfene Ausnahme wie ein totes Backend', async () => {
    const gateway: PositionChartGateway = { fetchChart: () => Promise.reject(new Error('boom')) };

    const { result } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));

    await waitFor(() => expect(result.current.status).toBe('error'));
    expect(result.current.errorMessage).toBe(mapPositionChartProblem('backend-unreachable'));
  });

  it('erholt sich nach einem Fehler beim Intervallwechsel', async () => {
    const { gateway } = createGateway(
      { kind: 'problem', problemType: 'unreachable' },
      { kind: 'ok', chart: xagChartHourly },
    );
    const { result } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));
    await waitFor(() => expect(result.current.status).toBe('error'));

    act(() => result.current.selectInterval('1h'));

    await waitFor(() => expect(result.current.status).toBe('ready'));
    expect(result.current.errorMessage).toBeUndefined();
  });

  it('bricht den Request beim Unmount ab', () => {
    const signals: AbortSignal[] = [];
    const gateway: PositionChartGateway = {
      fetchChart: (_id, _interval, _limit, signal) => {
        signals.push(signal);
        return new Promise(() => {});
      },
    };
    const { unmount } = renderHook(() => usePositionChartViewModel(CHART_POSITION_ID, gateway));

    unmount();

    expect(signals[0]?.aborted).toBe(true);
  });

  it('lädt neu, wenn sich die positionId ändert', async () => {
    const { gateway, fetchChart } = createGateway({ kind: 'ok', chart: xagLongChart });
    const { rerender } = renderHook(({ id }) => usePositionChartViewModel(id, gateway), {
      initialProps: { id: CHART_POSITION_ID },
    });
    await waitFor(() => expect(fetchChart).toHaveBeenCalledTimes(1));

    rerender({ id: '9100000000000000009' });

    await waitFor(() => expect(fetchChart).toHaveBeenCalledTimes(2));
    expect(fetchChart).toHaveBeenLastCalledWith(
      '9100000000000000009',
      '15m',
      300,
      expect.any(AbortSignal),
    );
  });
});
