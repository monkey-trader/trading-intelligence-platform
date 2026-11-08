import type { SwingLongProblemType } from '@/domain/position/SwingLongGateway';

const MESSAGES: Readonly<Record<SwingLongProblemType, string>> = {
  'bad-request': 'Ungültige Anfrage: Die Position konnte nicht geändert werden.',
  'forbidden-origin': 'Zugriff abgelehnt: Die Anfrage kommt von einer nicht erlaubten Herkunft.',
  internal: 'Interner Fehler im Backend. Bitte später erneut versuchen.',
  'backend-unreachable': 'Backend nicht erreichbar. Läuft der Server?',
  'invalid-body': 'Unerwartete Antwort vom Backend.',
};

/** Deutsche Meldung zu einem Problem beim Setzen/Entfernen von Swing Long. */
export function mapSwingLongProblem(problemType: SwingLongProblemType): string {
  return MESSAGES[problemType];
}
