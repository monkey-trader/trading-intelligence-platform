import { useCallback, useEffect, useState } from 'react';
import {
  DEFAULT_CHART_INTERVAL,
  DEFAULT_CHART_LIMIT,
  type ChartInterval,
  type PositionChart,
} from '@/domain/position/PositionChart';
import type { PositionChartGateway } from '@/domain/position/PositionChartGateway';
import { mapPositionChartProblem } from '@/presentation/position/chartErrorMapper';

/** Abstand zwischen zwei Live-Updates des Charts. */
export const CHART_POLL_INTERVAL_MS = 5000;

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
    let timer: ReturnType<typeof setTimeout> | undefined;
    let hasLoaded = false;
    let isLoading = false;

    const clearTimer = () => {
      if (timer !== undefined) clearTimeout(timer);
      timer = undefined;
    };
    const scheduleNext = () => {
      clearTimer();
      if (controller.signal.aborted || document.hidden) return;
      timer = setTimeout(() => void load(), CHART_POLL_INTERVAL_MS);
    };
    // Poll-Fehler behalten den letzten Chart; nur vor dem ersten Erfolg wird ein Fehler gezeigt.
    const showError = (errorMessage: string) => {
      if (!hasLoaded) setState({ requestKey, outcome: { status: 'error', errorMessage } });
    };
    const load = async () => {
      if (isLoading) return;
      isLoading = true;
      try {
        const result = await gateway.fetchChart(
          positionId,
          interval,
          DEFAULT_CHART_LIMIT,
          controller.signal,
        );
        if (controller.signal.aborted) return;
        if (result.kind === 'ok') {
          hasLoaded = true;
          // Neue Objekt-Identität pro Antwort, damit der Chart bei jedem Poll aktualisiert wird.
          setState({ requestKey, outcome: { status: 'ready', chart: { ...result.chart } } });
        } else {
          showError(mapPositionChartProblem(result.problemType, result.retryAfterSeconds));
        }
      } catch {
        if (controller.signal.aborted) return;
        showError(mapPositionChartProblem('backend-unreachable'));
      } finally {
        isLoading = false;
      }
      scheduleNext();
    };
    const onVisibilityChange = () => {
      if (document.hidden) {
        clearTimer();
      } else {
        clearTimer();
        void load();
      }
    };

    document.addEventListener('visibilitychange', onVisibilityChange);
    void load();
    return () => {
      controller.abort();
      clearTimer();
      document.removeEventListener('visibilitychange', onVisibilityChange);
    };
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
