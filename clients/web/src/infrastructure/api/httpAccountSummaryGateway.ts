import type { AccountSummary, PeriodKey, PeriodPnl } from '@/domain/account/AccountSummary';
import type { AccountSummaryGateway } from '@/domain/account/AccountSummaryGateway';
import type { components } from '@/infrastructure/api/generated';

type AccountSummaryDto = components['schemas']['AccountSummary'];
type PeriodPnlDto = components['schemas']['PeriodPnl'];

const ACCOUNT_SUMMARY_PATH = '/account/summary';
const DEFAULT_TIMEOUT_MS = 10_000;
const PERIOD_KEYS: readonly PeriodKey[] = ['today', '7d', '30d'];
const DECIMAL_PATTERN = /^-?\d+(\.\d+)?$/;
const MONTH_PATTERN = /^\d{4}-(0[1-9]|1[0-2])$/;

/** Gateway auf GET /account/summary des eigenen Backends (Basis wie bei den Positionen). */
export function createHttpAccountSummaryGateway(
  fetchFn: typeof fetch = fetch,
  baseUrl = '/api',
  timeoutMs = DEFAULT_TIMEOUT_MS,
): AccountSummaryGateway {
  return {
    async fetchAccountSummary(signal: AbortSignal): Promise<AccountSummary | undefined> {
      try {
        const response = await fetchFn(`${baseUrl}${ACCOUNT_SUMMARY_PATH}`, {
          method: 'GET',
          headers: { Accept: 'application/json' },
          signal: AbortSignal.any([signal, AbortSignal.timeout(timeoutMs)]),
        });
        if (!response.ok) return undefined;
        const dto = parseDto(await response.json());
        return dto ? toSummary(dto) : undefined;
      } catch (error) {
        if (signal.aborted) throw error;
        return undefined;
      }
    },
  };
}

function toSummary(dto: AccountSummaryDto): AccountSummary {
  return {
    balance: dto.balance,
    equity: dto.equity,
    availableMargin: dto.availableMargin,
    month: dto.month,
    monthlyRealizedPnl: dto.monthlyRealizedPnl,
    monthlyPnlPercent: dto.monthlyPnlPercent ?? undefined,
    periods: dto.periods.map(toPeriodPnl),
    unrealizedPnl: dto.unrealizedPnl,
    unrealizedPnlPercent: dto.unrealizedPnlPercent ?? undefined,
  };
}

function toPeriodPnl(dto: PeriodPnlDto): PeriodPnl {
  return {
    period: dto.period,
    realizedPnl: dto.realizedPnl,
    pnlPercent: dto.pnlPercent ?? undefined,
  };
}

function parsePeriodDtos(candidate: unknown): PeriodPnlDto[] | undefined {
  if (!Array.isArray(candidate)) return undefined;
  const periodDtos: PeriodPnlDto[] = [];
  for (const entry of candidate as unknown[]) {
    if (typeof entry !== 'object' || entry === null) return undefined;
    const { period, realizedPnl, pnlPercent } = entry as Record<string, unknown>;
    const knownPeriod = PERIOD_KEYS.find((key) => key === period);
    if (!knownPeriod || !isDecimal(realizedPnl) || !(pnlPercent === null || isDecimal(pnlPercent)))
      return undefined;
    periodDtos.push({ period: knownPeriod, realizedPnl, pnlPercent });
  }
  return periodDtos;
}

function parseDto(body: unknown): AccountSummaryDto | undefined {
  if (typeof body !== 'object' || body === null) return undefined;
  const {
    balance,
    equity,
    availableMargin,
    month,
    monthlyRealizedPnl,
    monthlyPnlPercent,
    periods,
    unrealizedPnl,
    unrealizedPnlPercent,
  } = body as Record<string, unknown>;
  const periodDtos = parsePeriodDtos(periods);
  if (
    !isDecimal(balance) ||
    !isDecimal(equity) ||
    !isDecimal(availableMargin) ||
    !isDecimal(monthlyRealizedPnl) ||
    typeof month !== 'string' ||
    !MONTH_PATTERN.test(month) ||
    !(monthlyPnlPercent === null || isDecimal(monthlyPnlPercent)) ||
    !periodDtos ||
    !isDecimal(unrealizedPnl) ||
    !(unrealizedPnlPercent === null || isDecimal(unrealizedPnlPercent))
  )
    return undefined;
  return {
    balance,
    equity,
    availableMargin,
    month,
    monthlyRealizedPnl,
    monthlyPnlPercent,
    periods: periodDtos,
    unrealizedPnl,
    unrealizedPnlPercent,
  };
}

function isDecimal(value: unknown): value is string {
  return typeof value === 'string' && DECIMAL_PATTERN.test(value);
}
