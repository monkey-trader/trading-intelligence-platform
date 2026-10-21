import type { AccountSummary } from '@/domain/account/AccountSummary';

/**
 * Port zum Abfragen des Kontostands. Der Kontostand ist Zusatzinformation:
 * bei jedem Fehler kommt undefined, nur ein Abbruch über das Signal wird als AbortError geworfen.
 */
export type AccountSummaryGateway = Readonly<{
  fetchAccountSummary: (signal: AbortSignal) => Promise<AccountSummary | undefined>;
}>;
