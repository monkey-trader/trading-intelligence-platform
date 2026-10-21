import { Ajv2020, type ErrorObject, type ValidateFunction } from 'ajv/dist/2020.js';
import addFormatsModule from 'ajv-formats';
import { InvalidOpenApiDocumentError, UnknownOperationError, UnknownSchemaError } from '@/errors';
import {
  isJsonObject,
  type JsonObject,
  listOperations,
  type OpenApiDocument,
  type OperationInfo,
  resolvePointer,
  toPointer,
} from '@/openapi';

/** Ergebnis einer Prüfung: gültig oder Liste lesbarer Abweichungen vom Vertrag. */
export type ValidationResult = Readonly<{ valid: boolean; errors: readonly string[] }>;

export type ResponseToValidate = Readonly<{
  operationId: string;
  status: number;
  /** Roher Content-Type-Header, Parameter wie charset werden ignoriert. */
  contentType: string | undefined;
  /** Bereits geparster Body (JSON) oder undefined bei leerem Body. */
  body: unknown;
}>;

/** Interne $id, unter der das ganze Dokument bei Ajv registriert wird. */
const SPEC_ID = 'https://contracts.local/openapi.json';

/**
 * OpenAPI-Schlüsselwörter, die keine JSON-Schema-Keywords sind. Ajv kennt sie nicht und würde
 * sie im strict-Modus ablehnen; als Vokabular registriert, bleiben Tippfehler in Schemas trotzdem Fehler.
 */
const OPENAPI_KEYWORDS = [
  'openapi',
  'info',
  'jsonSchemaDialect',
  'servers',
  'security',
  'paths',
  'webhooks',
  'components',
  'tags',
  'externalDocs',
  'example',
  'discriminator',
  'xml',
];

// ajv-formats liefert CommonJS; je nach Loader steckt die Funktion in `default`.
type AddFormats = (ajv: Ajv2020) => Ajv2020;
const addFormats: AddFormats =
  (addFormatsModule as unknown as { default?: AddFormats }).default ??
  (addFormatsModule as unknown as AddFormats);

/** Content-Type ohne Parameter, klein geschrieben (z.B. 'application/problem+json'). */
export function mediaTypeOf(contentType: string | undefined): string | undefined {
  if (contentType === undefined) return undefined;
  const mediaType = contentType.split(';')[0]?.trim().toLowerCase();
  return mediaType === '' ? undefined : mediaType;
}

function formatAjvErrors(errors: readonly ErrorObject[] | null | undefined): string[] {
  return (errors ?? []).map((error) =>
    `${error.instancePath === '' ? '(Wurzel)' : error.instancePath} ${error.message ?? ''}`.trim(),
  );
}

const VALID: ValidationResult = { valid: true, errors: [] };

function invalid(...errors: string[]): ValidationResult {
  return { valid: false, errors };
}

/**
 * Prüft Antworten gegen docs/openapi.yaml. Das ganze Dokument wird mit fester $id bei Ajv
 * (JSON Schema 2020-12, wie OpenAPI 3.1) registriert, damit Refs wie '#/components/schemas/X'
 * auflösen. Response-Refs ('#/components/responses/X') löst der Validator selbst auf.
 */
export class OpenApiValidator {
  private readonly operationsById: ReadonlyMap<string, OperationInfo>;
  private readonly compiledSchemas = new Map<string, ValidateFunction>();

  private constructor(
    private readonly document: OpenApiDocument,
    private readonly ajv: Ajv2020,
  ) {
    this.operationsById = new Map(
      listOperations(document).map((operation) => [operation.operationId, operation]),
    );
  }

  static create(document: OpenApiDocument): OpenApiValidator {
    const ajv = new Ajv2020({ strict: true, allErrors: true, allowUnionTypes: true });
    addFormats(ajv);
    ajv.addVocabulary(OPENAPI_KEYWORDS);
    ajv.addSchema({ ...document, $id: SPEC_ID });
    const validator = new OpenApiValidator(document, ajv);
    validator.compileAllComponentSchemas();
    return validator;
  }

  /** Alle operationIds der Spec. */
  operationIds(): string[] {
    return [...this.operationsById.keys()];
  }

  operation(operationId: string): OperationInfo {
    const operation = this.operationsById.get(operationId);
    if (operation === undefined) throw new UnknownOperationError(operationId);
    return operation;
  }

  /** Dokumentierte Status-Schlüssel der Operation, z.B. ['200', '403']. */
  documentedStatuses(operationId: string): string[] {
    return Object.keys(this.operation(operationId).responses);
  }

  /** true, wenn der Status (exakt, als 4XX/5XX oder über default) dokumentiert ist. */
  isDocumentedStatus(operationId: string, status: number): boolean {
    return this.findResponseKey(operationId, status) !== undefined;
  }

  /** Header-Namen (klein geschrieben), die die Spec für diesen Status beschreibt. */
  documentedHeaders(operationId: string, status: number): string[] {
    const resolved = this.resolveResponse(operationId, status);
    if (resolved === undefined) return [];
    const { headers } = resolved.response;
    return isJsonObject(headers) ? Object.keys(headers).map((name) => name.toLowerCase()) : [];
  }

  /** Prüft Status, Content-Type und Body einer Antwort gegen die Operation. */
  validateResponse(response: ResponseToValidate): ValidationResult {
    const resolved = this.resolveResponse(response.operationId, response.status);
    if (resolved === undefined) {
      return invalid(
        `Status ${response.status} ist für ${response.operationId} nicht dokumentiert ` +
          `(erlaubt: ${this.documentedStatuses(response.operationId).join(', ')}).`,
      );
    }
    const { content } = resolved.response;
    if (!isJsonObject(content) || Object.keys(content).length === 0) {
      return response.body === undefined
        ? VALID
        : invalid(`Status ${response.status} hat laut Spec keinen Body.`);
    }
    const mediaType = mediaTypeOf(response.contentType);
    const documentedMediaTypes = Object.keys(content);
    const matchingMediaType = documentedMediaTypes.find(
      (documented) => documented.toLowerCase() === mediaType,
    );
    if (matchingMediaType === undefined) {
      return invalid(
        `Content-Type "${response.contentType ?? '(fehlt)'}" ist für ${response.operationId} ` +
          `${response.status} nicht dokumentiert (erlaubt: ${documentedMediaTypes.join(', ')}).`,
      );
    }
    const mediaTypeObject = content[matchingMediaType];
    if (!isJsonObject(mediaTypeObject) || mediaTypeObject['schema'] === undefined) return VALID;
    return this.validateAt(
      `${resolved.pointer}${toPointer(['content', matchingMediaType, 'schema'])}`,
      response.body,
    );
  }

  /** Prüft einen Wert gegen components/schemas/<schemaName>. */
  validateSchema(schemaName: string, value: unknown): ValidationResult {
    if (
      resolvePointer(this.document, toPointer(['components', 'schemas', schemaName])) === undefined
    ) {
      throw new UnknownSchemaError(schemaName);
    }
    return this.validateAt(toPointer(['components', 'schemas', schemaName]), value);
  }

  private validateAt(pointer: string, value: unknown): ValidationResult {
    const validate = this.compiledAt(pointer);
    if (validate(value)) return VALID;
    return invalid(...formatAjvErrors(validate.errors));
  }

  private compiledAt(pointer: string): ValidateFunction {
    const cached = this.compiledSchemas.get(pointer);
    if (cached !== undefined) return cached;
    const fragment = pointer
      .split('/')
      .map((segment) => encodeURIComponent(segment))
      .join('/');
    const validate = this.ajv.getSchema(`${SPEC_ID}#${fragment}`);
    if (validate === undefined) {
      throw new InvalidOpenApiDocumentError(`Kein Schema unter ${pointer}.`);
    }
    this.compiledSchemas.set(pointer, validate);
    return validate;
  }

  /** Kompiliert alle Komponenten-Schemas sofort, damit Fehler in der Spec früh auffallen. */
  private compileAllComponentSchemas(): void {
    const schemas = resolvePointer(this.document, '/components/schemas');
    if (!isJsonObject(schemas)) return;
    for (const schemaName of Object.keys(schemas)) {
      this.compiledAt(toPointer(['components', 'schemas', schemaName]));
    }
  }

  private findResponseKey(operationId: string, status: number): string | undefined {
    const { responses } = this.operation(operationId);
    const statusText = String(status);
    const rangeKey = `${statusText[0] ?? ''}XX`;
    return [statusText, rangeKey, rangeKey.toLowerCase(), 'default'].find(
      (candidate) => responses[candidate] !== undefined,
    );
  }

  /** Response-Objekt samt Pointer; folgt lokalen $ref-Ketten wie '#/components/responses/Problem'. */
  private resolveResponse(
    operationId: string,
    status: number,
  ): { pointer: string; response: JsonObject } | undefined {
    const responseKey = this.findResponseKey(operationId, status);
    if (responseKey === undefined) return undefined;
    let pointer = `${this.operation(operationId).pointer}${toPointer(['responses', responseKey])}`;
    const visitedPointers = new Set<string>();
    for (;;) {
      if (visitedPointers.has(pointer)) {
        throw new InvalidOpenApiDocumentError(`Zyklische Response-Referenz bei ${pointer}.`);
      }
      visitedPointers.add(pointer);
      const response = resolvePointer(this.document, pointer);
      if (!isJsonObject(response)) {
        throw new InvalidOpenApiDocumentError(`Keine Response unter ${pointer}.`);
      }
      const reference = response['$ref'];
      if (reference === undefined) return { pointer, response };
      if (typeof reference !== 'string' || !reference.startsWith('#')) {
        throw new InvalidOpenApiDocumentError(
          `Nur lokale Response-Referenzen werden unterstützt (${String(reference)}).`,
        );
      }
      pointer = reference.slice(1);
    }
  }
}
