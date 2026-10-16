import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import { addDecimals } from '@/domain/shared/decimal/decimalArithmetic';

export type OpenPositionsSummary = Readonly<{ totalUnrealizedProfit: string }>;

/** Exakte Summe des unrealisierten PnL aller offenen Positionen, "0" bei leerer Liste. */
export function summarizeOpenPositions(
  positions: readonly Pick<OpenPosition, 'unrealizedProfit'>[],
): OpenPositionsSummary {
  const totalUnrealizedProfit = positions.reduce(
    (sum, position) => addDecimals(sum, position.unrealizedProfit),
    '0',
  );
  return { totalUnrealizedProfit };
}
