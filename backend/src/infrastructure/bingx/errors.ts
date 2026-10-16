// Fehler des BingX-Adapters. Die Texte enthalten nie Key, Secret oder die signierte URL.

/** Basis aller Fehler beim Zugriff auf BingX. */
export class BingXError extends Error {
  constructor(message: string) {
    super(message);
    this.name = new.target.name;
  }
}

/** BingX hat mit `code != 0` geantwortet. */
export class BingXApiError extends BingXError {
  constructor(
    readonly code: number,
    readonly apiMessage: string,
    readonly path: string,
  ) {
    super(`BingX-Fehler ${code} bei ${path}: ${apiMessage}`);
  }
}

/** HTTP-Status außerhalb von 2xx (429 und 5xx erst nach allen Wiederholungen). */
export class BingXHttpError extends BingXError {
  constructor(
    readonly status: number,
    readonly path: string,
  ) {
    super(`BingX antwortete mit HTTP ${status} bei ${path}.`);
  }
}

/** Rate-Limit (HTTP 429 oder code 100410) auch nach allen Wiederholungen. */
export class BingXRateLimitError extends BingXError {
  constructor(
    readonly path: string,
    readonly attemptCount: number,
  ) {
    super(`BingX-Rate-Limit bei ${path}: nach ${attemptCount} Versuchen aufgegeben.`);
  }
}

/**
 * Netzwerkfehler oder Timeout auch nach allen Wiederholungen. Nennt nur den Fehlertyp, nicht die
 * Originalmeldung, damit nie Teile der signierten URL in Logs landen.
 */
export class BingXNetworkError extends BingXError {
  constructor(
    readonly path: string,
    readonly attemptCount: number,
    readonly causeName: string,
  ) {
    super(
      `BingX nicht erreichbar bei ${path} (${causeName}): nach ${attemptCount} Versuchen aufgegeben.`,
    );
  }
}

/** Die Antwort hat nicht die erwartete Form. Wird geworfen, statt still eine leere Liste zu liefern. */
export class UnexpectedBingXResponseError extends BingXError {}
