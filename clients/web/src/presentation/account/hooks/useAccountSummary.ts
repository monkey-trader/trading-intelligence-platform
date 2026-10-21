import { useEffect, useState } from 'react';
import type { AccountSummary } from '@/domain/account/AccountSummary';
import type { AccountSummaryGateway } from '@/domain/account/AccountSummaryGateway';

export const ACCOUNT_POLL_INTERVAL_MS = 30_000;

/**
 * Lädt den Kontostand und pollt. Fehler verbergen nur die Anzeige:
 * der letzte bekannte Stand bleibt stehen. Ohne Gateway passiert nichts.
 */
export function useAccountSummary(
  gateway: AccountSummaryGateway | undefined,
  pollIntervalMs = ACCOUNT_POLL_INTERVAL_MS,
): AccountSummary | undefined {
  const [summary, setSummary] = useState<AccountSummary>();

  useEffect(() => {
    if (!gateway) return;
    let activeController: AbortController | undefined;
    const load = () => {
      if (document.visibilityState === 'hidden') return;
      activeController?.abort();
      const controller = new AbortController();
      activeController = controller;
      gateway.fetchAccountSummary(controller.signal).then(
        (loaded) => {
          if (!controller.signal.aborted && loaded) setSummary(loaded);
        },
        () => {
          // Abbruch oder unerwarteter Fehler: Anzeige bleibt unverändert.
        },
      );
    };
    load();
    const timer = setInterval(load, pollIntervalMs);
    return () => {
      clearInterval(timer);
      activeController?.abort();
    };
  }, [gateway, pollIntervalMs]);

  return summary;
}
