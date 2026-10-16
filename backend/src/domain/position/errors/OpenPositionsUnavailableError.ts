/** Gründe, warum die offenen Positionen gerade nicht geliefert werden können. */
export type OpenPositionsUnavailableReason =
  'not-configured' | 'rate-limited' | 'unreachable' | 'upstream-error' | 'invalid-response';

/**
 * Die offenen Positionen sind nicht verfügbar. Adapter übersetzen ihre eigenen Fehler hierhin,
 * damit äußere Schichten die Fehlerklassen der Quelle nicht kennen müssen.
 * Die Ursache bleibt in `cause` erhalten und ist nur für Server-Logs gedacht.
 */
export class OpenPositionsUnavailableError extends Error {
  constructor(
    readonly reason: OpenPositionsUnavailableReason,
    cause?: unknown,
  ) {
    super(
      `Offene Positionen nicht verfügbar: ${reason}`,
      cause === undefined ? undefined : { cause },
    );
    this.name = new.target.name;
  }
}
