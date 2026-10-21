import type { InitialStopLossProblemType } from '@/domain/position/InitialStopLossGateway';

const PROBLEM_MESSAGES: Readonly<Record<InitialStopLossProblemType, string>> = {
  'invalid-stop-loss': 'SL muss auf der Verlustseite liegen',
  'not-found':
    'Position dem Backend unbekannt (nicht in der lokalen DB). Backend neu starten oder Liste neu laden.',
  'bad-request': 'Ungültige Eingabe.',
  'forbidden-origin': 'Anfrage vom Backend abgelehnt (Origin-Prüfung).',
  'not-configured': 'BingX-Zugang fehlt (BX_KEY/BX_SECRET im Backend setzen).',
  'rate-limited': 'BingX ausgelastet, bitte kurz warten.',
  unreachable: 'BingX ist gerade nicht erreichbar.',
  'upstream-error': 'BingX meldet einen Fehler.',
  'invalid-response': 'BingX hat unerwartete Daten geliefert.',
  internal: 'Interner Fehler im Backend.',
  'backend-unreachable': 'Backend nicht erreichbar (läuft der Server aus backend/?).',
  'invalid-body': 'Unerwartete Antwort vom Backend.',
};

/** Deutsche Meldung zu einem Problem beim Speichern oder Zurücksetzen des Ursprungs-SL. */
export function mapInitialStopLossProblem(problemType: InitialStopLossProblemType): string {
  return PROBLEM_MESSAGES[problemType];
}
