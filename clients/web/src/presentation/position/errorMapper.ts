import type { OpenPositionsProblemType } from '@/domain/position/OpenPositionsResult';

const PROBLEM_MESSAGES: Readonly<Record<OpenPositionsProblemType, string>> = {
  'forbidden-host': 'Zugriff vom Backend abgelehnt (Host-Prüfung).',
  'not-configured': 'BingX-Zugang fehlt (BX_KEY/BX_SECRET im Backend setzen).',
  'rate-limited': 'BingX ausgelastet, bitte kurz warten.',
  unreachable: 'BingX ist gerade nicht erreichbar.',
  'upstream-error': 'BingX meldet einen Fehler.',
  'invalid-response': 'BingX hat unerwartete Daten geliefert.',
  internal: 'Interner Fehler im Backend.',
  'backend-unreachable': 'Backend nicht erreichbar (läuft npm run dev in backend/?).',
  'invalid-body': 'Unerwartete Antwort vom Backend (Client und Backend passen nicht zusammen?).',
};

/** Deutsche Meldung zu einem Problem; Retry-After zählt nur bei rate-limited. */
export function mapOpenPositionsProblem(
  problemType: OpenPositionsProblemType,
  retryAfterSeconds?: number,
): string {
  if (problemType === 'rate-limited' && retryAfterSeconds !== undefined) {
    return `BingX ausgelastet, neuer Versuch in ${retryAfterSeconds} s.`;
  }
  return PROBLEM_MESSAGES[problemType];
}
