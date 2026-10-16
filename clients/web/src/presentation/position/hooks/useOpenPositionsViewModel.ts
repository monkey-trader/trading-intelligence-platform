import { useCallback, useEffect, useRef, useState } from 'react';
import type { OpenPositionsSnapshot } from '@/domain/position/OpenPosition';
import type { OpenPositionsGateway } from '@/domain/position/OpenPositionsGateway';
import type { OpenPositionsResult } from '@/domain/position/OpenPositionsResult';
import { mapOpenPositionsProblem } from '@/presentation/position/errorMapper';

export const DEFAULT_POLL_INTERVAL_MS = 15_000;

export type OpenPositionsStatus = 'loading' | 'ready' | 'empty' | 'error';

export type OpenPositionsViewState = Readonly<{
  status: OpenPositionsStatus;
  snapshot: OpenPositionsSnapshot | undefined;
  errorMessage: string | undefined;
  /** true, wenn der angezeigte Snapshot nach einem Fehler nicht mehr aktuell ist. */
  isStale: boolean;
}>;

export type OpenPositionsViewModel = OpenPositionsViewState &
  Readonly<{
    /** Lädt sofort neu und startet das Intervall danach neu. */
    refresh: () => void;
  }>;

export type OpenPositionsViewModelOptions = Readonly<{
  pollIntervalMs?: number;
}>;

const INITIAL_STATE: OpenPositionsViewState = {
  status: 'loading',
  snapshot: undefined,
  errorMessage: undefined,
  isStale: false,
};

/**
 * Lädt die offenen Positionen und pollt danach.
 * - setTimeout-Kette: der nächste Request startet erst nach Abschluss des vorigen.
 * - Retry-After verlängert die Wartezeit einmalig.
 * - Kein Polling bei verborgenem Tab; beim Zurückkehren wird sofort geladen.
 * - Ein neuer Request bricht den laufenden ab, verspätete Antworten werden verworfen.
 */
export function useOpenPositionsViewModel(
  gateway: OpenPositionsGateway,
  options: OpenPositionsViewModelOptions = {},
): OpenPositionsViewModel {
  const pollIntervalMs = options.pollIntervalMs ?? DEFAULT_POLL_INTERVAL_MS;
  const [viewState, setViewState] = useState<OpenPositionsViewState>(INITIAL_STATE);
  const loadNowRef = useRef<() => void>(() => {});

  useEffect(() => {
    let activeController: AbortController | undefined;
    let pollTimer: ReturnType<typeof setTimeout> | undefined;
    let isDisposed = false;

    const isTabVisible = () => document.visibilityState !== 'hidden';

    const clearPollTimer = () => {
      if (pollTimer !== undefined) clearTimeout(pollTimer);
      pollTimer = undefined;
    };

    const scheduleNextLoad = (delayMs: number) => {
      clearPollTimer();
      if (isDisposed || !isTabVisible()) return;
      pollTimer = setTimeout(loadNow, delayMs);
    };

    const handleResult = (result: OpenPositionsResult) => {
      setViewState((previousState) => nextViewState(previousState, result));
      scheduleNextLoad(nextDelayMs(result, pollIntervalMs));
    };

    function loadNow() {
      clearPollTimer();
      activeController?.abort();
      activeController = undefined;
      if (isDisposed || !isTabVisible()) return;

      const controller = new AbortController();
      activeController = controller;
      const isCurrentRequest = () => !isDisposed && activeController === controller;

      gateway.fetchOpenPositions(controller.signal).then(
        (result) => {
          if (!isCurrentRequest()) return;
          activeController = undefined;
          handleResult(result);
        },
        (error: unknown) => {
          if (!isCurrentRequest()) return;
          activeController = undefined;
          if (isAbortError(error)) return;
          // Das Gateway liefert Fehler als Ergebnis; alles andere behandeln wir wie ein totes Backend.
          handleResult({ kind: 'problem', problemType: 'backend-unreachable' });
        },
      );
    }

    const handleVisibilityChange = () => {
      if (isTabVisible()) loadNow();
      else clearPollTimer();
    };

    loadNowRef.current = loadNow;
    document.addEventListener('visibilitychange', handleVisibilityChange);
    loadNow();

    return () => {
      isDisposed = true;
      clearPollTimer();
      activeController?.abort();
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      loadNowRef.current = () => {};
    };
  }, [gateway, pollIntervalMs]);

  const refresh = useCallback(() => {
    loadNowRef.current();
  }, []);

  return { ...viewState, refresh };
}

function nextViewState(
  previousState: OpenPositionsViewState,
  result: OpenPositionsResult,
): OpenPositionsViewState {
  if (result.kind === 'ok') {
    return {
      status: result.snapshot.positions.length > 0 ? 'ready' : 'empty',
      snapshot: result.snapshot,
      errorMessage: undefined,
      isStale: false,
    };
  }
  const errorMessage = mapOpenPositionsProblem(result.problemType, result.retryAfterSeconds);
  if (previousState.snapshot) {
    return { ...previousState, errorMessage, isStale: true };
  }
  return { status: 'error', snapshot: undefined, errorMessage, isStale: false };
}

const MAX_RETRY_AFTER_MS = 5 * 60_000;

function nextDelayMs(result: OpenPositionsResult, pollIntervalMs: number): number {
  if (result.kind === 'problem' && result.retryAfterSeconds !== undefined) {
    // Obergrenze, damit ein extremes Retry-After das Polling nicht stundenlang anhält.
    return Math.min(MAX_RETRY_AFTER_MS, Math.max(pollIntervalMs, result.retryAfterSeconds * 1000));
  }
  return pollIntervalMs;
}

function isAbortError(error: unknown): boolean {
  return (
    typeof error === 'object' && error !== null && 'name' in error && error.name === 'AbortError'
  );
}
