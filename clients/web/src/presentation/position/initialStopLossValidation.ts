import type { PositionSide } from '@/domain/position/OpenPosition';

const POSITIVE_DECIMAL_PATTERN = /^\d+(\.\d+)?$/;

/** undefined = gültig, sonst Fehlertext. Der SL muss auf der Verlustseite des Entry liegen. */
export function validateInitialStopLoss(
  side: PositionSide,
  entry: string,
  raw: string,
): string | undefined {
  const trimmed = raw.trim();
  if (!POSITIVE_DECIMAL_PATTERN.test(trimmed)) {
    return 'Bitte eine Zahl mit Punkt als Dezimaltrenner eingeben.';
  }
  const stopLoss = Number(trimmed);
  if (stopLoss <= 0) return 'Der SL muss größer als 0 sein.';
  const entryPrice = Number(entry);
  const onLossSide = side === 'LONG' ? stopLoss < entryPrice : stopLoss > entryPrice;
  return onLossSide ? undefined : 'SL muss auf der Verlustseite liegen';
}
