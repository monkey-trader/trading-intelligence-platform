/**
 * Typisierte Fehler der Testinfrastruktur. Sie melden Bedienfehler (falsche Konfiguration,
 * unbekannte operationId) und unterscheiden sich so von Vertragsverletzungen der API,
 * die der Validator als Ergebnis zurückgibt.
 */

/** CONTRACT_BASE_URL fehlt. */
export class MissingBaseUrlError extends Error {
  override readonly name = 'MissingBaseUrlError';
  constructor() {
    super(
      'CONTRACT_BASE_URL ist nicht gesetzt. Beispiel: CONTRACT_BASE_URL=http://127.0.0.1:3000 npm test',
    );
  }
}

/** CONTRACT_BASE_URL ist keine brauchbare http(s)-Basis-URL. */
export class InvalidBaseUrlError extends Error {
  override readonly name = 'InvalidBaseUrlError';
  constructor(
    readonly value: string,
    reason: string,
  ) {
    super(`CONTRACT_BASE_URL "${value}" ist ungültig: ${reason}`);
  }
}

/** CONTRACT_EXPECT_CONFIGURED hat einen anderen Wert als "true" oder "false". */
export class InvalidExpectConfiguredError extends Error {
  override readonly name = 'InvalidExpectConfiguredError';
  constructor(readonly value: string) {
    super(`CONTRACT_EXPECT_CONFIGURED muss "true" oder "false" sein, war "${value}".`);
  }
}

/** Die OpenAPI-Datei ist kein brauchbares OpenAPI-3.1-Dokument. */
export class InvalidOpenApiDocumentError extends Error {
  override readonly name = 'InvalidOpenApiDocumentError';
}

/** Die operationId steht nicht in der Spec. */
export class UnknownOperationError extends Error {
  override readonly name = 'UnknownOperationError';
  constructor(readonly operationId: string) {
    super(`operationId "${operationId}" steht nicht in docs/openapi.yaml.`);
  }
}

/** Das Schema steht nicht unter components/schemas. */
export class UnknownSchemaError extends Error {
  override readonly name = 'UnknownSchemaError';
  constructor(readonly schemaName: string) {
    super(`Schema "${schemaName}" steht nicht unter components/schemas.`);
  }
}
