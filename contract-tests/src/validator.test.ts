import { describe, expect, it } from 'vitest';
import { InvalidOpenApiDocumentError, UnknownOperationError, UnknownSchemaError } from '@/errors';
import { loadOpenApi, parseOpenApi, toOpenApiDocument } from '@/openapi';
import { mediaTypeOf, OpenApiValidator } from '@/validator';

/**
 * Kleine Spec mit genau den Stolperstellen von OpenAPI 3.1 / JSON Schema 2020-12:
 * Pattern-Strings, oneOf mit type 'null', exclusiveMinimum, $ref mit Geschwister-Keywords,
 * Response-Refs auf components/responses und Pfade mit Sonderzeichen.
 */
const MINI_SPEC_YAML = String.raw`
openapi: 3.1.0
info:
  title: Mini
  version: 1.0.0
paths:
  /things/{thingId}:
    get:
      operationId: getThing
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Thing'
              example:
                amount: '1.5'
                ratio: null
                leverage: 2
        '404':
          $ref: '#/components/responses/Problem'
        5XX:
          $ref: '#/components/responses/Alias'
  /empty:
    delete:
      operationId: deleteEmpty
      responses:
        '204':
          description: kein Body
components:
  responses:
    Alias:
      $ref: '#/components/responses/Problem'
    Problem:
      description: Fehler
      headers:
        Retry-After:
          schema:
            type: integer
      content:
        application/problem+json:
          schema:
            $ref: '#/components/schemas/Problem'
  schemas:
    Decimal:
      type: string
      pattern: '^-?\d+(\.\d+)?$'
      example: '0.5'
    Problem:
      type: object
      additionalProperties: false
      required: [type, title, status]
      properties:
        type:
          type: string
          enum: [not-found, internal]
        title:
          type: string
        status:
          type: integer
    Thing:
      type: object
      additionalProperties: false
      required: [amount, ratio, leverage]
      properties:
        amount:
          $ref: '#/components/schemas/Decimal'
          description: Geschwister-Keyword neben $ref ist in 2020-12 erlaubt.
        ratio:
          oneOf:
            - $ref: '#/components/schemas/Decimal'
            - type: 'null'
        leverage:
          type: number
          exclusiveMinimum: 0
`;

function miniValidator(): OpenApiValidator {
  return OpenApiValidator.create(parseOpenApi(MINI_SPEC_YAML));
}

const validThing = { amount: '-12.50', ratio: null, leverage: 0.5 };

describe('OpenApiValidator – Schemas', () => {
  it('akzeptiert ein gültiges Objekt', () => {
    const result = miniValidator().validateSchema('Thing', validThing);

    expect(result).toEqual({ valid: true, errors: [] });
  });

  it('akzeptiert für oneOf sowohl Decimal als auch null', () => {
    const validator = miniValidator();

    expect(validator.validateSchema('Thing', { ...validThing, ratio: '0.06' }).valid).toBe(true);
    expect(validator.validateSchema('Thing', { ...validThing, ratio: null }).valid).toBe(true);
  });

  it.each([
    ['Zahl statt String', { ...validThing, amount: 1.5 }],
    ['Pattern verletzt', { ...validThing, amount: '1,5' }],
    ['Pattern verletzt (Exponent)', { ...validThing, amount: '1e5' }],
    ['ratio weder Decimal noch null', { ...validThing, ratio: 'abc' }],
    ['exclusiveMinimum: 0 ist nicht erlaubt', { ...validThing, leverage: 0 }],
    ['exclusiveMinimum: negativ', { ...validThing, leverage: -1 }],
    ['Pflichtfeld fehlt', { amount: '1', ratio: null }],
    ['zusätzliches Feld', { ...validThing, extra: true }],
  ])('lehnt ab: %s', (_label, candidate) => {
    const result = miniValidator().validateSchema('Thing', candidate);

    expect(result.valid).toBe(false);
    expect(result.errors.length).toBeGreaterThan(0);
  });

  it('nennt den Pfad der Abweichung in der Fehlermeldung', () => {
    const result = miniValidator().validateSchema('Thing', { ...validThing, amount: 'x' });

    expect(result.errors.join('\n')).toContain('/amount');
  });

  it('wirft UnknownSchemaError für ein unbekanntes Schema', () => {
    expect(() => miniValidator().validateSchema('Gibtsnicht', {})).toThrow(UnknownSchemaError);
  });
});

describe('OpenApiValidator – Antworten', () => {
  it('prüft einen 200-Body gegen das referenzierte Schema, charset im Content-Type stört nicht', () => {
    const result = miniValidator().validateResponse({
      operationId: 'getThing',
      status: 200,
      contentType: 'application/json; charset=utf-8',
      body: validThing,
    });

    expect(result).toEqual({ valid: true, errors: [] });
  });

  it('meldet einen ungültigen 200-Body', () => {
    const result = miniValidator().validateResponse({
      operationId: 'getThing',
      status: 200,
      contentType: 'application/json',
      body: { ...validThing, leverage: 0 },
    });

    expect(result.valid).toBe(false);
  });

  it('löst Response-Refs auf (#/components/responses/Problem)', () => {
    const validator = miniValidator();
    const problem = { type: 'not-found', title: 'Nicht gefunden.', status: 404 };

    expect(
      validator.validateResponse({
        operationId: 'getThing',
        status: 404,
        contentType: 'application/problem+json',
        body: problem,
      }).valid,
    ).toBe(true);
    expect(
      validator.validateResponse({
        operationId: 'getThing',
        status: 404,
        contentType: 'application/problem+json',
        body: { ...problem, message: 'intern' },
      }).valid,
    ).toBe(false);
  });

  it('löst Bereichs-Status (5XX) und verkettete Response-Refs auf', () => {
    const validator = miniValidator();

    expect(validator.isDocumentedStatus('getThing', 503)).toBe(true);
    expect(
      validator.validateResponse({
        operationId: 'getThing',
        status: 500,
        contentType: 'application/problem+json',
        body: { type: 'internal', title: 'Interner Fehler.', status: 500 },
      }).valid,
    ).toBe(true);
  });

  it('meldet einen nicht dokumentierten Status', () => {
    const validator = miniValidator();

    const result = validator.validateResponse({
      operationId: 'getThing',
      status: 418,
      contentType: 'application/json',
      body: {},
    });

    expect(validator.isDocumentedStatus('getThing', 418)).toBe(false);
    expect(result.valid).toBe(false);
    expect(result.errors.join('\n')).toContain('418');
  });

  it('meldet einen nicht dokumentierten Content-Type', () => {
    const result = miniValidator().validateResponse({
      operationId: 'getThing',
      status: 404,
      contentType: 'application/json',
      body: { type: 'not-found', title: 'x', status: 404 },
    });

    expect(result.valid).toBe(false);
    expect(result.errors.join('\n')).toContain('application/json');
  });

  it('meldet einen fehlenden Content-Type', () => {
    const result = miniValidator().validateResponse({
      operationId: 'getThing',
      status: 200,
      contentType: undefined,
      body: validThing,
    });

    expect(result.valid).toBe(false);
  });

  it('akzeptiert eine Antwort ohne Body, wenn die Spec keinen Inhalt beschreibt', () => {
    const validator = miniValidator();

    expect(
      validator.validateResponse({
        operationId: 'deleteEmpty',
        status: 204,
        contentType: undefined,
        body: undefined,
      }).valid,
    ).toBe(true);
    expect(
      validator.validateResponse({
        operationId: 'deleteEmpty',
        status: 204,
        contentType: 'application/json',
        body: { unerwartet: true },
      }).valid,
    ).toBe(false);
  });

  it('liefert dokumentierte Status und Header (klein geschrieben, über Response-Refs)', () => {
    const validator = miniValidator();

    expect(validator.documentedStatuses('getThing')).toEqual(['200', '404', '5XX']);
    expect(validator.documentedHeaders('getThing', 404)).toEqual(['retry-after']);
    expect(validator.documentedHeaders('getThing', 200)).toEqual([]);
  });

  it('listet alle operationIds', () => {
    expect(miniValidator().operationIds()).toEqual(['getThing', 'deleteEmpty']);
  });

  it('wirft UnknownOperationError für eine unbekannte operationId', () => {
    const validator = miniValidator();

    expect(() => validator.documentedStatuses('gibtsNicht')).toThrow(UnknownOperationError);
    expect(() =>
      validator.validateResponse({
        operationId: 'gibtsNicht',
        status: 200,
        contentType: 'application/json',
        body: {},
      }),
    ).toThrow(UnknownOperationError);
  });
});

describe('OpenApiValidator – ungültige Specs', () => {
  it('wirft InvalidOpenApiDocumentError für OpenAPI 3.0', () => {
    expect(() => toOpenApiDocument({ openapi: '3.0.3', paths: {} })).toThrow(
      InvalidOpenApiDocumentError,
    );
  });

  it('wirft InvalidOpenApiDocumentError ohne paths', () => {
    expect(() => toOpenApiDocument({ openapi: '3.1.0' })).toThrow(InvalidOpenApiDocumentError);
  });

  it('wirft InvalidOpenApiDocumentError bei einer Operation ohne operationId', () => {
    const document = toOpenApiDocument({
      openapi: '3.1.0',
      paths: { '/x': { get: { responses: { '200': { description: 'ok' } } } } },
    });

    expect(() => OpenApiValidator.create(document)).toThrow(InvalidOpenApiDocumentError);
  });

  it('wirft InvalidOpenApiDocumentError bei doppelter operationId', () => {
    const operation = { operationId: 'doppelt', responses: { '200': { description: 'ok' } } };
    const document = toOpenApiDocument({
      openapi: '3.1.0',
      paths: { '/a': { get: operation }, '/b': { get: operation } },
    });

    expect(() => OpenApiValidator.create(document)).toThrow(InvalidOpenApiDocumentError);
  });

  it('wirft InvalidOpenApiDocumentError bei einer zyklischen Response-Referenz', () => {
    const document = toOpenApiDocument({
      openapi: '3.1.0',
      paths: {
        '/x': {
          get: {
            operationId: 'zyklisch',
            responses: { '200': { $ref: '#/components/responses/A' } },
          },
        },
      },
      components: {
        responses: {
          A: { $ref: '#/components/responses/B' },
          B: { $ref: '#/components/responses/A' },
        },
      },
    });
    const validator = OpenApiValidator.create(document);

    expect(() =>
      validator.validateResponse({
        operationId: 'zyklisch',
        status: 200,
        contentType: 'application/json',
        body: {},
      }),
    ).toThrow(InvalidOpenApiDocumentError);
  });

  it('lehnt einen Tippfehler in einem Schema-Keyword ab (strict)', () => {
    const document = toOpenApiDocument({
      openapi: '3.1.0',
      paths: {},
      components: { schemas: { Kaputt: { type: 'string', patern: '^a$' } } },
    });

    expect(() => OpenApiValidator.create(document)).toThrow();
  });
});

describe('mediaTypeOf', () => {
  it.each([
    ['application/json; charset=utf-8', 'application/json'],
    ['Application/Problem+JSON', 'application/problem+json'],
    [undefined, undefined],
    ['', undefined],
  ])('%j → %j', (contentType, expected) => {
    expect(mediaTypeOf(contentType)).toBe(expected);
  });
});

describe('docs/openapi.yaml', () => {
  it('lässt sich laden und vollständig kompilieren', () => {
    expect(() => OpenApiValidator.create(loadOpenApi())).not.toThrow();
  });

  it('akzeptiert das Beispiel aus der Spec für getHealth', () => {
    const validator = OpenApiValidator.create(loadOpenApi());

    expect(
      validator.validateResponse({
        operationId: 'getHealth',
        status: 200,
        contentType: 'application/json',
        body: { status: 'ok' },
      }),
    ).toEqual({ valid: true, errors: [] });
  });

  it('kennt den Problem-Typ not-found', () => {
    const validator = OpenApiValidator.create(loadOpenApi());

    expect(
      validator.validateSchema('Problem', { type: 'not-found', title: 'x', status: 404 }).valid,
    ).toBe(true);
  });

  it('prüft liquidationDistanceRatio als Decimal oder null', () => {
    const validator = OpenApiValidator.create(loadOpenApi());
    const position = {
      positionId: '9100000000000000001',
      symbol: 'SUI-USDT',
      side: 'LONG',
      isolated: true,
      positionAmount: '500',
      avgOpenPrice: '0.9600',
      markPrice: '1.0100',
      leverage: 10,
      initialMargin: '48.0000',
      margin: '60.5000',
      positionValue: '505',
      unrealizedProfit: '25.0000',
      realisedProfit: '-0.2400',
      pnlRatio: '0.5208',
      liquidationPrice: '0',
      liquidationDistanceRatio: null,
      openTime: 1791500000000,
      updateTime: 1791600000000,
    };

    expect(validator.validateSchema('OpenPosition', position).valid).toBe(true);
    expect(
      validator.validateSchema('OpenPosition', { ...position, liquidationDistanceRatio: 0.06 })
        .valid,
    ).toBe(false);
  });
});
