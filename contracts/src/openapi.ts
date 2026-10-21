import { readFileSync } from 'node:fs';
import { parse } from 'yaml';
import { InvalidOpenApiDocumentError } from '@/errors';

/** Standardpfad der Spec: docs/openapi.yaml im Repo-Root. */
export const DEFAULT_OPENAPI_URL = new URL('../../docs/openapi.yaml', import.meta.url);

export type JsonObject = { [key: string]: unknown };

/** Minimal geprüftes OpenAPI-3.1-Dokument; Details prüft der Validator beim Zugriff. */
export type OpenApiDocument = JsonObject & { openapi: string; paths: JsonObject };

export const HTTP_METHODS = [
  'get',
  'put',
  'post',
  'delete',
  'options',
  'head',
  'patch',
  'trace',
] as const;

export type HttpMethod = (typeof HTTP_METHODS)[number];

/** Eine Operation der Spec mit JSON-Pointer auf ihr Objekt im Dokument. */
export type OperationInfo = Readonly<{
  operationId: string;
  method: HttpMethod;
  path: string;
  pointer: string;
  responses: JsonObject;
}>;

export function isJsonObject(value: unknown): value is JsonObject {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/** Prüft die Grundform (openapi 3.1.x, paths) und gibt das Dokument typisiert zurück. */
export function toOpenApiDocument(value: unknown): OpenApiDocument {
  if (!isJsonObject(value)) {
    throw new InvalidOpenApiDocumentError('Die Spec ist kein Objekt.');
  }
  const { openapi, paths } = value;
  if (typeof openapi !== 'string' || !openapi.startsWith('3.1.')) {
    throw new InvalidOpenApiDocumentError('Nur OpenAPI 3.1.x wird unterstützt.');
  }
  if (!isJsonObject(paths)) {
    throw new InvalidOpenApiDocumentError('Die Spec hat kein Objekt "paths".');
  }
  return { ...value, openapi, paths };
}

/** Parst YAML-Text zu einem OpenAPI-Dokument. */
export function parseOpenApi(yamlText: string): OpenApiDocument {
  return toOpenApiDocument(parse(yamlText));
}

/** Lädt docs/openapi.yaml (oder eine andere Datei) von der Platte. */
export function loadOpenApi(fileUrl: URL = DEFAULT_OPENAPI_URL): OpenApiDocument {
  return parseOpenApi(readFileSync(fileUrl, 'utf8'));
}

/** Maskiert ein Segment für einen JSON-Pointer (RFC 6901). */
export function escapePointerSegment(segment: string): string {
  return segment.replaceAll('~', '~0').replaceAll('/', '~1');
}

/** Baut einen JSON-Pointer aus Segmenten, z.B. ['paths', '/health'] → '/paths/~1health'. */
export function toPointer(segments: readonly string[]): string {
  return segments.map((segment) => `/${escapePointerSegment(segment)}`).join('');
}

/** Löst einen lokalen JSON-Pointer im Dokument auf; undefined, wenn er ins Leere zeigt. */
export function resolvePointer(document: JsonObject, pointer: string): unknown {
  if (pointer === '') return document;
  if (!pointer.startsWith('/')) return undefined;
  let current: unknown = document;
  for (const rawSegment of pointer.slice(1).split('/')) {
    const segment = rawSegment.replaceAll('~1', '/').replaceAll('~0', '~');
    if (!isJsonObject(current)) return undefined;
    current = current[segment];
  }
  return current;
}

/** Alle Operationen der Spec in Dokumentreihenfolge. Wirft bei fehlender oder doppelter operationId. */
export function listOperations(document: OpenApiDocument): OperationInfo[] {
  const operations: OperationInfo[] = [];
  const seenOperationIds = new Set<string>();
  for (const [path, pathItem] of Object.entries(document.paths)) {
    if (!isJsonObject(pathItem)) continue;
    for (const method of HTTP_METHODS) {
      const operation = pathItem[method];
      if (!isJsonObject(operation)) continue;
      const { operationId, responses } = operation;
      if (typeof operationId !== 'string' || operationId === '') {
        throw new InvalidOpenApiDocumentError(
          `${method.toUpperCase()} ${path} hat keine operationId.`,
        );
      }
      if (seenOperationIds.has(operationId)) {
        throw new InvalidOpenApiDocumentError(`operationId "${operationId}" ist doppelt.`);
      }
      if (!isJsonObject(responses)) {
        throw new InvalidOpenApiDocumentError(`${operationId} hat keine responses.`);
      }
      seenOperationIds.add(operationId);
      operations.push({
        operationId,
        method,
        path,
        pointer: toPointer(['paths', path, method]),
        responses,
      });
    }
  }
  return operations;
}
