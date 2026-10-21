import { readdirSync, readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { OPERATION_TEST_NAMES } from '@/contract/coverage';
import { listOperations, loadOpenApi } from '@/openapi';

describe('Abdeckung des Vertrags (ohne Server)', () => {
  const specOperationIds = listOperations(loadOpenApi()).map((operation) => operation.operationId);
  const coveredOperationIds = Object.keys(OPERATION_TEST_NAMES);

  it('hat für jede operationId der Spec einen Vertragstest', () => {
    const missing = specOperationIds.filter(
      (operationId) => !coveredOperationIds.includes(operationId),
    );

    expect(missing, 'operationIds ohne Vertragstest').toEqual([]);
  });

  it('hat keine Einträge für operationIds, die es in der Spec nicht gibt', () => {
    const superfluous = coveredOperationIds.filter(
      (operationId) => !specOperationIds.includes(operationId),
    );

    expect(superfluous, 'überflüssige Einträge').toEqual([]);
  });

  it('vergibt jeden Testfall-Namen nur einmal', () => {
    const testNames = Object.values(OPERATION_TEST_NAMES);

    expect(new Set(testNames).size).toBe(testNames.length);
  });
});

/** Verzeichnis dieser Datei; dort liegen alle *.contract.test.ts. */
const CONTRACT_DIRECTORY = new URL('./', import.meta.url);

function readContractTestSources(): ReadonlyArray<Readonly<{ fileName: string; source: string }>> {
  return readdirSync(CONTRACT_DIRECTORY)
    .filter((fileName) => fileName.endsWith('.contract.test.ts'))
    .map((fileName) => ({
      fileName,
      source: readFileSync(new URL(fileName, CONTRACT_DIRECTORY), 'utf8'),
    }));
}

function escapeForRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/**
 * Ob ein Quelltext einen describe-Block für die operationId enthält: entweder
 * `describe(OPERATION_TEST_NAMES.<operationId>,` oder `describe('<gemappter Name>',`.
 */
function hasDescribeFor(source: string, operationId: string, testName: string): boolean {
  const byReference = new RegExp(
    `\\bdescribe\\(\\s*OPERATION_TEST_NAMES\\.${escapeForRegExp(operationId)}\\s*,`,
  );
  const quotedName = escapeForRegExp(testName);
  const byLiteral = new RegExp(`\\bdescribe\\(\\s*(['"\`])${quotedName}\\1\\s*,`);
  return byReference.test(source) || byLiteral.test(source);
}

describe('Abdeckung des Vertrags: describe-Blöcke in *.contract.test.ts (statisch)', () => {
  const contractSources = readContractTestSources();

  it('findet Vertragstest-Dateien', () => {
    expect(contractSources.length).toBeGreaterThan(0);
  });

  it.each(Object.entries(OPERATION_TEST_NAMES))(
    'hat für %s einen describe-Block mit dem gemappten Namen',
    (operationId, testName) => {
      const filesWithDescribe = contractSources
        .filter(({ source }) => hasDescribeFor(source, operationId, testName))
        .map(({ fileName }) => fileName);

      expect(filesWithDescribe, `describe für ${operationId} ("${testName}")`).toHaveLength(1);
    },
  );

  it('hat für jede operationId der Spec einen describe-Block (nicht nur einen Map-Eintrag)', () => {
    const testNames: Readonly<Record<string, string>> = OPERATION_TEST_NAMES;
    const missing = specOperationIdsOf().filter((operationId) => {
      const testName = testNames[operationId];
      return (
        testName === undefined ||
        !contractSources.some(({ source }) => hasDescribeFor(source, operationId, testName))
      );
    });

    expect(missing, 'operationIds ohne describe-Block').toEqual([]);
  });

  describe('Erkennung (Selbsttest, damit der Check nicht falsch-grün ist)', () => {
    it('erkennt describe(OPERATION_TEST_NAMES.<operationId>, ...)', () => {
      expect(
        hasDescribeFor('describe(OPERATION_TEST_NAMES.getHealth, () => {});', 'getHealth', 'x'),
      ).toBe(true);
    });

    it('erkennt describe mit genau dem gemappten Namen als String', () => {
      expect(
        hasDescribeFor(
          "describe('GET /health (getHealth)', () => {});",
          'getHealth',
          'GET /health (getHealth)',
        ),
      ).toBe(true);
    });

    it('ignoriert Erwähnungen ohne describe, andere operationIds und Präfixe', () => {
      const testName = 'GET /health (getHealth)';
      expect(
        hasDescribeFor("it('GET /health (getHealth)', () => {});", 'getHealth', testName),
      ).toBe(false);
      expect(hasDescribeFor('// OPERATION_TEST_NAMES.getHealth', 'getHealth', testName)).toBe(
        false,
      );
      expect(
        hasDescribeFor(
          'describe(OPERATION_TEST_NAMES.getHealthX, () => {});',
          'getHealth',
          testName,
        ),
      ).toBe(false);
      expect(
        hasDescribeFor(
          "describe('GET /health (getHealth) extra', () => {});",
          'getHealth',
          testName,
        ),
      ).toBe(false);
      expect(
        hasDescribeFor(
          "describe.skip('GET /health (getHealth)', () => {});",
          'getHealth',
          testName,
        ),
      ).toBe(false);
    });
  });
});

function specOperationIdsOf(): string[] {
  return listOperations(loadOpenApi()).map((operation) => operation.operationId);
}
