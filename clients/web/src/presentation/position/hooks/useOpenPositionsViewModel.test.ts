import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { OpenPositionsSnapshot } from '@/domain/position/OpenPosition';
import type { OpenPositionsGateway } from '@/domain/position/OpenPositionsGateway';
import type { OpenPositionsResult } from '@/domain/position/OpenPositionsResult';
import { emptySnapshot, sampleSnapshot, suiLongPosition } from '@/test/fixtures/openPositions';
import { mapOpenPositionsProblem } from '@/presentation/position/errorMapper';
import { useOpenPositionsViewModel } from '@/presentation/position/hooks/useOpenPositionsViewModel';

type PendingRequest = {
  signal: AbortSignal;
  resolve: (result: OpenPositionsResult) => void;
  reject: (reason: unknown) => void;
};

/** Gateway, dessen Requests der Test einzeln abschliesst. */
function createControllableGateway() {
  const requests: PendingRequest[] = [];
  const gateway: OpenPositionsGateway = {
    fetchOpenPositions: (signal: AbortSignal) =>
      new Promise<OpenPositionsResult>((resolve, reject) => {
        requests.push({ signal, resolve, reject });
      }),
  };
  return { gateway, requests };
}

function requestAt(requests: readonly PendingRequest[], index: number): PendingRequest {
  const request = requests[index];
  if (!request)
    throw new Error(`Request ${index} wurde nicht gestellt (${requests.length} gestellt)`);
  return request;
}

async function completeRequest(
  requests: readonly PendingRequest[],
  index: number,
  result: OpenPositionsResult,
) {
  await act(async () => {
    requestAt(requests, index).resolve(result);
  });
}

async function advanceTime(milliseconds: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(milliseconds);
  });
}

function ok(snapshot: OpenPositionsSnapshot): OpenPositionsResult {
  return { kind: 'ok', snapshot };
}

let visibilityState: DocumentVisibilityState = 'visible';

function setVisibility(nextState: DocumentVisibilityState) {
  visibilityState = nextState;
}

async function changeVisibility(nextState: DocumentVisibilityState) {
  setVisibility(nextState);
  await act(async () => {
    document.dispatchEvent(new Event('visibilitychange'));
  });
}

beforeEach(() => {
  vi.useFakeTimers();
  visibilityState = 'visible';
  Object.defineProperty(document, 'visibilityState', {
    configurable: true,
    get: () => visibilityState,
  });
});

afterEach(() => {
  vi.useRealTimers();
  Reflect.deleteProperty(document, 'visibilityState');
});

describe('useOpenPositionsViewModel', () => {
  describe('erster Aufruf', () => {
    it('lädt beim Mount sofort und meldet loading', () => {
      const { gateway, requests } = createControllableGateway();

      const { result } = renderHook(() => useOpenPositionsViewModel(gateway));

      expect(requests).toHaveLength(1);
      expect(requestAt(requests, 0).signal.aborted).toBe(false);
      expect(result.current.status).toBe('loading');
      expect(result.current.snapshot).toBeUndefined();
      expect(result.current.errorMessage).toBeUndefined();
      expect(result.current.isStale).toBe(false);
    });

    it('meldet ready mit Snapshot nach erfolgreicher Antwort', async () => {
      const { gateway, requests } = createControllableGateway();
      const { result } = renderHook(() => useOpenPositionsViewModel(gateway));

      await completeRequest(requests, 0, ok(sampleSnapshot));

      expect(result.current.status).toBe('ready');
      expect(result.current.snapshot).toEqual(sampleSnapshot);
      expect(result.current.errorMessage).toBeUndefined();
      expect(result.current.isStale).toBe(false);
    });

    it('meldet empty, wenn keine Position offen ist', async () => {
      const { gateway, requests } = createControllableGateway();
      const { result } = renderHook(() => useOpenPositionsViewModel(gateway));

      await completeRequest(requests, 0, ok(emptySnapshot));

      expect(result.current.status).toBe('empty');
      expect(result.current.snapshot).toEqual(emptySnapshot);
    });

    it('meldet error mit deutscher Meldung, wenn noch keine Daten vorliegen', async () => {
      const { gateway, requests } = createControllableGateway();
      const { result } = renderHook(() => useOpenPositionsViewModel(gateway));

      await completeRequest(requests, 0, { kind: 'problem', problemType: 'not-configured' });

      expect(result.current.status).toBe('error');
      expect(result.current.snapshot).toBeUndefined();
      expect(result.current.errorMessage).toBe(mapOpenPositionsProblem('not-configured'));
      expect(result.current.isStale).toBe(false);
    });
  });

  describe('Polling', () => {
    it('lädt 15 s nach Abschluss des vorigen Requests erneut', async () => {
      const { gateway, requests } = createControllableGateway();
      renderHook(() => useOpenPositionsViewModel(gateway));
      await completeRequest(requests, 0, ok(sampleSnapshot));

      await advanceTime(14_999);
      expect(requests).toHaveLength(1);

      await advanceTime(1);
      expect(requests).toHaveLength(2);
    });

    it('startet keinen neuen Request, solange der vorige noch läuft', async () => {
      const { gateway, requests } = createControllableGateway();
      renderHook(() => useOpenPositionsViewModel(gateway));

      await advanceTime(60_000);
      expect(requests).toHaveLength(1);

      await completeRequest(requests, 0, ok(sampleSnapshot));
      await advanceTime(15_000);
      expect(requests).toHaveLength(2);
    });

    it('pollt auch nach einem Fehler weiter', async () => {
      const { gateway, requests } = createControllableGateway();
      renderHook(() => useOpenPositionsViewModel(gateway));
      await completeRequest(requests, 0, { kind: 'problem', problemType: 'unreachable' });

      await advanceTime(15_000);

      expect(requests).toHaveLength(2);
    });

    it('verwendet ein eigenes Intervall aus pollIntervalMs', async () => {
      const { gateway, requests } = createControllableGateway();
      renderHook(() => useOpenPositionsViewModel(gateway, { pollIntervalMs: 5_000 }));
      await completeRequest(requests, 0, ok(sampleSnapshot));

      await advanceTime(4_999);
      expect(requests).toHaveLength(1);

      await advanceTime(1);
      expect(requests).toHaveLength(2);
    });

    it('wartet bei Retry-After 30 s statt 15 s bis zum nächsten Request', async () => {
      const { gateway, requests } = createControllableGateway();
      renderHook(() => useOpenPositionsViewModel(gateway));
      await completeRequest(requests, 0, {
        kind: 'problem',
        problemType: 'rate-limited',
        retryAfterSeconds: 30,
      });

      await advanceTime(29_999);
      expect(requests).toHaveLength(1);

      await advanceTime(1);
      expect(requests).toHaveLength(2);
    });

    it('begrenzt eine sehr lange Retry-After-Wartezeit auf 5 Minuten', async () => {
      const { gateway, requests } = createControllableGateway();
      renderHook(() => useOpenPositionsViewModel(gateway));
      await completeRequest(requests, 0, {
        kind: 'problem',
        problemType: 'rate-limited',
        retryAfterSeconds: 3600,
      });

      await advanceTime(299_999);
      expect(requests).toHaveLength(1);

      await advanceTime(1);
      expect(requests).toHaveLength(2);
    });

    it('nennt die Wartezeit aus Retry-After in der Meldung', async () => {
      const { gateway, requests } = createControllableGateway();
      const { result } = renderHook(() => useOpenPositionsViewModel(gateway));

      await completeRequest(requests, 0, {
        kind: 'problem',
        problemType: 'rate-limited',
        retryAfterSeconds: 30,
      });

      expect(result.current.errorMessage).toBe(mapOpenPositionsProblem('rate-limited', 30));
    });

    it('kehrt nach Retry-After wieder zum normalen Intervall zurück', async () => {
      const { gateway, requests } = createControllableGateway();
      renderHook(() => useOpenPositionsViewModel(gateway));
      await completeRequest(requests, 0, {
        kind: 'problem',
        problemType: 'rate-limited',
        retryAfterSeconds: 30,
      });
      await advanceTime(30_000);
      await completeRequest(requests, 1, ok(sampleSnapshot));

      await advanceTime(15_000);

      expect(requests).toHaveLength(3);
    });
  });

  describe('Sichtbarkeit des Tabs', () => {
    it('stellt keinen Request, solange der Tab verborgen ist', async () => {
      setVisibility('hidden');
      const { gateway, requests } = createControllableGateway();

      const { result } = renderHook(() => useOpenPositionsViewModel(gateway));
      await advanceTime(60_000);

      expect(requests).toHaveLength(0);
      expect(result.current.status).toBe('loading');
    });

    it('lädt sofort, wenn der Tab wieder sichtbar wird', async () => {
      setVisibility('hidden');
      const { gateway, requests } = createControllableGateway();
      renderHook(() => useOpenPositionsViewModel(gateway));

      await changeVisibility('visible');

      expect(requests).toHaveLength(1);
    });

    it('pollt nicht weiter, während der Tab verborgen ist, und lädt beim Zurückkehren sofort', async () => {
      const { gateway, requests } = createControllableGateway();
      renderHook(() => useOpenPositionsViewModel(gateway));
      await completeRequest(requests, 0, ok(sampleSnapshot));

      await changeVisibility('hidden');
      await advanceTime(60_000);
      expect(requests).toHaveLength(1);

      await changeVisibility('visible');
      expect(requests).toHaveLength(2);
    });

    it('stellt beim Wechsel zu hidden keinen zusätzlichen Request', async () => {
      const { gateway, requests } = createControllableGateway();
      renderHook(() => useOpenPositionsViewModel(gateway));
      await completeRequest(requests, 0, ok(sampleSnapshot));

      await changeVisibility('hidden');

      expect(requests).toHaveLength(1);
    });
  });

  describe('Abbruch', () => {
    it('bricht den laufenden Request beim Unmount ab', () => {
      const { gateway, requests } = createControllableGateway();
      const { unmount } = renderHook(() => useOpenPositionsViewModel(gateway));

      unmount();

      expect(requestAt(requests, 0).signal.aborted).toBe(true);
    });

    it('stoppt das Polling nach dem Unmount', async () => {
      const { gateway, requests } = createControllableGateway();
      const { unmount } = renderHook(() => useOpenPositionsViewModel(gateway));
      await completeRequest(requests, 0, ok(sampleSnapshot));

      unmount();
      await advanceTime(60_000);

      expect(requests).toHaveLength(1);
    });

    it('reagiert nach dem Unmount nicht mehr auf visibilitychange', async () => {
      const { gateway, requests } = createControllableGateway();
      const { unmount } = renderHook(() => useOpenPositionsViewModel(gateway));
      await completeRequest(requests, 0, ok(sampleSnapshot));
      unmount();

      await changeVisibility('hidden');
      await changeVisibility('visible');

      expect(requests).toHaveLength(1);
    });

    it('ignoriert einen AbortError des Gateways ohne Fehlermeldung', async () => {
      const { gateway, requests } = createControllableGateway();
      const { result } = renderHook(() => useOpenPositionsViewModel(gateway));
      await completeRequest(requests, 0, ok(sampleSnapshot));
      act(() => {
        result.current.refresh();
      });
      act(() => {
        result.current.refresh();
      });

      await act(async () => {
        requestAt(requests, 1).reject(new DOMException('aborted', 'AbortError'));
      });

      expect(result.current.errorMessage).toBeUndefined();
      expect(result.current.isStale).toBe(false);
      expect(result.current.snapshot).toEqual(sampleSnapshot);
    });
  });

  describe('Fehler nach erfolgreichen Daten', () => {
    it('behält den Snapshot, setzt isStale und die Meldung', async () => {
      const { gateway, requests } = createControllableGateway();
      const { result } = renderHook(() => useOpenPositionsViewModel(gateway));
      await completeRequest(requests, 0, ok(sampleSnapshot));
      await advanceTime(15_000);

      await completeRequest(requests, 1, { kind: 'problem', problemType: 'unreachable' });

      expect(result.current.status).toBe('ready');
      expect(result.current.snapshot).toEqual(sampleSnapshot);
      expect(result.current.isStale).toBe(true);
      expect(result.current.errorMessage).toBe(mapOpenPositionsProblem('unreachable'));
    });

    it('setzt isStale und die Meldung zurück, sobald wieder Daten kommen', async () => {
      const { gateway, requests } = createControllableGateway();
      const { result } = renderHook(() => useOpenPositionsViewModel(gateway));
      await completeRequest(requests, 0, ok(sampleSnapshot));
      await advanceTime(15_000);
      await completeRequest(requests, 1, { kind: 'problem', problemType: 'backend-unreachable' });
      await advanceTime(15_000);

      const newerSnapshot: OpenPositionsSnapshot = {
        positions: [suiLongPosition],
        totalUnrealizedProfit: '25.0000',
        fetchedAt: sampleSnapshot.fetchedAt + 30_000,
      };
      await completeRequest(requests, 2, ok(newerSnapshot));

      expect(result.current.status).toBe('ready');
      expect(result.current.snapshot).toEqual(newerSnapshot);
      expect(result.current.isStale).toBe(false);
      expect(result.current.errorMessage).toBeUndefined();
    });
  });

  describe('refresh()', () => {
    it('lädt sofort, ohne auf das Intervall zu warten', async () => {
      const { gateway, requests } = createControllableGateway();
      const { result } = renderHook(() => useOpenPositionsViewModel(gateway));
      await completeRequest(requests, 0, ok(sampleSnapshot));

      act(() => {
        result.current.refresh();
      });

      expect(requests).toHaveLength(2);
    });

    it('startet das Intervall nach dem manuellen Laden neu', async () => {
      const { gateway, requests } = createControllableGateway();
      const { result } = renderHook(() => useOpenPositionsViewModel(gateway));
      await completeRequest(requests, 0, ok(sampleSnapshot));
      await advanceTime(10_000);

      act(() => {
        result.current.refresh();
      });
      await completeRequest(requests, 1, ok(sampleSnapshot));

      await advanceTime(14_999);
      expect(requests).toHaveLength(2);

      await advanceTime(1);
      expect(requests).toHaveLength(3);
    });

    it('bricht einen noch laufenden Request ab, bevor es neu lädt', () => {
      const { gateway, requests } = createControllableGateway();
      const { result } = renderHook(() => useOpenPositionsViewModel(gateway));

      act(() => {
        result.current.refresh();
      });

      expect(requests).toHaveLength(2);
      expect(requestAt(requests, 0).signal.aborted).toBe(true);
      expect(requestAt(requests, 1).signal.aborted).toBe(false);
    });

    it('übernimmt keine verspätete Antwort eines abgebrochenen Requests', async () => {
      const { gateway, requests } = createControllableGateway();
      const { result } = renderHook(() => useOpenPositionsViewModel(gateway));
      act(() => {
        result.current.refresh();
      });

      await completeRequest(requests, 1, ok(emptySnapshot));
      await completeRequest(requests, 0, ok(sampleSnapshot));

      expect(result.current.status).toBe('empty');
      expect(result.current.snapshot).toEqual(emptySnapshot);
    });
  });
});
