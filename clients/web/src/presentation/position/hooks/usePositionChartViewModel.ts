import { useCallback, useEffect, useState } from 'react';
import {
  DEFAULT_CHART_INTERVAL,
  DEFAULT_CHART_LIMIT,
  type ChartInterval,
  type PositionChart,
} from '@/domain/position/PositionChart';
import type { PositionChartGateway } from '@/domain/position/PositionChartGateway';
import { mapPositionChartProblem } from '@/presentation/position/chartErrorMapper';

type ChartOutcome =
  | Readonly<{ status: 'ready'; chart: PositionChart }>
  | Readonly<{ status: 'error'; errorMessage: string }>;

/** Antwort samt Anfrage-Schlüssel: Nur eine Antwort zur aktuellen Anfrage zählt, sonst gilt "loading". */
type ChartState = Readonly<{ requestKey: string; outcome: ChartOutcome }>;

export type PositionChartViewModel = Readonly<{
  status: 'loading' | ChartOutcome['status'];
  chart?: PositionChart | undefined;
  errorMessage?: string | undefined;
  interval: ChartInterval;
  selectInterval: (interval: ChartInterval) => void;
}>;

/** Lädt den Chart einer Position; bricht laufende Requests ab und verwirft späte Antworten. */
export function usePositionChartViewModel(
  positionId: string,
  gateway: PositionChartGateway,
): PositionChartViewModel {
  const [interval, setInterval] = useState<ChartInterval>(DEFAULT_CHART_INTERVAL);
  const [state, setState] = useState<ChartState>();
  const requestKey = `${positionId}|${interval}`;

  useEffect(() => {
    const controller = new AbortController();
    gateway
      .fetchChart(positionId, interval, DEFAULT_CHART_LIMIT, controller.signal)
      .then((result) => {
        if (controller.signal.aborted) return;
        setState({
          requestKey,
          outcome:
            result.kind === 'ok'
              ? { status: 'ready', chart: result.chart }
              : {
                  status: 'error',
                  errorMessage: mapPositionChartProblem(
                    result.problemType,
                    result.retryAfterSeconds,
                  ),
                },
        });
      })
      .catch(() => {
        if (controller.signal.aborted) return;
        setState({
          requestKey,
          outcome: {
            status: 'error',
            errorMessage: mapPositionChartProblem('backend-unreachable'),
          },
        });
      });
    return () => controller.abort();
  }, [positionId, interval, gateway, requestKey]);

  const selectInterval = useCallback(
    (nextInterval: ChartInterval) => setInterval(nextInterval),
    [],
  );

  const outcome = state?.requestKey === requestKey ? state.outcome : undefined;

  return {
    status: outcome?.status ?? 'loading',
    chart: outcome?.status === 'ready' ? outcome.chart : undefined,
    errorMessage: outcome?.status === 'error' ? outcome.errorMessage : undefined,
    interval,
    selectInterval,
  };
}
