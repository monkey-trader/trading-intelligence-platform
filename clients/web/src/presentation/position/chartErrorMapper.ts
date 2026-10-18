import type { PositionChartProblemType } from '@/domain/position/PositionChartResult';
import { mapOpenPositionsProblem } from '@/presentation/position/errorMapper';

/** Deutsche Meldung zu einem Chart-Problem; übrige Probleme wie in der Positionsliste. */
export function mapPositionChartProblem(
  problemType: PositionChartProblemType,
  retryAfterSeconds?: number,
): string {
  if (problemType === 'not-found') return 'Position nicht gefunden (nicht mehr offen?).';
  if (problemType === 'bad-request') return 'Ungültige Chart-Anfrage.';
  return mapOpenPositionsProblem(problemType, retryAfterSeconds);
}
