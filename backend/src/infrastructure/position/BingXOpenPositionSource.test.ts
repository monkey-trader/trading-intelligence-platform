import { describe, expect, it } from 'vitest';
import { BingXOpenPositionSource } from '@/infrastructure/position/BingXOpenPositionSource';
import {
  BingXApiError,
  BingXHttpError,
  BingXNetworkError,
  BingXRateLimitError,
  UnexpectedBingXResponseError,
} from '@/infrastructure/bingx/errors';
import type { BingXQueryParams } from '@/infrastructure/bingx/BingXGetClient';
import { OpenPositionsUnavailableError } from '@/domain/position/errors/OpenPositionsUnavailableError';
import type { OpenPositionsUnavailableReason } from '@/domain/position/errors/OpenPositionsUnavailableError';
import { PositionDomainError } from '@/domain/position/errors/PositionDomainError';
import { InvalidPositionIdError } from '@/domain/position/errors/InvalidPositionIdError';
import openPositionsFixture from '@/infrastructure/position/bingx/fixtures/openPositions.json';
import openPositionsEmptyFixture from '@/infrastructure/position/bingx/fixtures/openPositions.empty.json';

const OPEN_POSITIONS_PATH = '/openApi/swap/v2/user/positions';

type GetCall = { path: string; params: BingXQueryParams };

/** Fake für den BingX-Client: liefert immer dieselbe vorbereitete `data`-Antwort. */
function createFakeClient(data: unknown) {
  const calls: GetCall[] = [];
  const client = {
    async get(path: string, params: BingXQueryParams): Promise<unknown> {
      calls.push({ path, params });
      return data;
    },
  };
  return { client, calls };
}

/** Fake für den BingX-Client, der immer mit dem übergebenen Fehler scheitert. */
function createFailingClient(clientError: unknown) {
  return {
    async get(): Promise<unknown> {
      throw clientError;
    },
  };
}

async function captureRejection(action: () => Promise<unknown>): Promise<unknown> {
  try {
    await action();
  } catch (error) {
    return error;
  }
  throw new Error('Es wurde ein Fehler erwartet, aber keiner geworfen.');
}

describe('BingXOpenPositionSource', () => {
  it('ruft /openApi/swap/v2/user/positions ohne Parameter genau einmal auf', async () => {
    const { client, calls } = createFakeClient(openPositionsFixture.data);
    const source = new BingXOpenPositionSource(client);

    await source.fetchOpenPositions();

    expect(calls).toEqual([{ path: OPEN_POSITIONS_PATH, params: {} }]);
  });

  it('liefert die gemappten offenen Positionen', async () => {
    const { client } = createFakeClient(openPositionsFixture.data);
    const source = new BingXOpenPositionSource(client);

    const positions = await source.fetchOpenPositions();

    expect(positions.map((position) => position.positionId)).toEqual([
      '9100000000000000001',
      '9100000000000000002',
    ]);
    expect(positions[0]?.liquidationPrice).toBe('0.865');
    expect(positions[1]?.realisedProfit).toBe('-0.5187');
  });

  it('liefert bei leerer Liste ein leeres Array', async () => {
    const { client } = createFakeClient(openPositionsEmptyFixture.data);
    const source = new BingXOpenPositionSource(client);

    await expect(source.fetchOpenPositions()).resolves.toEqual([]);
  });

  it.each([
    ['BingXRateLimitError', new BingXRateLimitError(OPEN_POSITIONS_PATH, 2), 'rate-limited'],
    [
      'BingXNetworkError',
      new BingXNetworkError(OPEN_POSITIONS_PATH, 2, 'TimeoutError'),
      'unreachable',
    ],
    [
      'BingXApiError',
      new BingXApiError(100001, 'signature verification failed', OPEN_POSITIONS_PATH),
      'upstream-error',
    ],
    ['BingXHttpError', new BingXHttpError(500, OPEN_POSITIONS_PATH), 'upstream-error'],
    [
      'UnexpectedBingXResponseError',
      new UnexpectedBingXResponseError('Antwort ohne erwartete Form'),
      'invalid-response',
    ],
    [
      'PositionDomainError',
      new InvalidPositionIdError('Ungültige Positions-ID'),
      'invalid-response',
    ],
  ] as const)(
    'übersetzt %s in OpenPositionsUnavailableError mit reason %s',
    async (_errorName, clientError, expectedReason: OpenPositionsUnavailableReason) => {
      const source = new BingXOpenPositionSource(createFailingClient(clientError));

      const caught = await captureRejection(() => source.fetchOpenPositions());

      expect(caught).toBeInstanceOf(OpenPositionsUnavailableError);
      expect(caught instanceof OpenPositionsUnavailableError ? caught.reason : undefined).toBe(
        expectedReason,
      );
      expect(caught instanceof Error ? caught.cause : undefined).toBe(clientError);
    },
  );

  it('übersetzt eine unerwartete Antwortform aus dem Mapper in invalid-response', async () => {
    const { client } = createFakeClient({ positions: [] });
    const source = new BingXOpenPositionSource(client);

    const caught = await captureRejection(() => source.fetchOpenPositions());

    expect(caught).toBeInstanceOf(OpenPositionsUnavailableError);
    expect(caught instanceof OpenPositionsUnavailableError ? caught.reason : undefined).toBe(
      'invalid-response',
    );
    expect(caught instanceof Error ? caught.cause : undefined).toBeInstanceOf(
      UnexpectedBingXResponseError,
    );
  });

  it('übersetzt einen Domain-Fehler aus dem Mapping in invalid-response', async () => {
    const firstRawPosition = openPositionsFixture.data[0];
    const { client } = createFakeClient([{ ...firstRawPosition, positionId: 'abc' }]);
    const source = new BingXOpenPositionSource(client);

    const caught = await captureRejection(() => source.fetchOpenPositions());

    expect(caught).toBeInstanceOf(OpenPositionsUnavailableError);
    expect(caught instanceof OpenPositionsUnavailableError ? caught.reason : undefined).toBe(
      'invalid-response',
    );
    expect(caught instanceof Error ? caught.cause : undefined).toBeInstanceOf(PositionDomainError);
  });

  it('reicht unbekannte Fehler unverändert durch (werden später zu 500 internal)', async () => {
    const unknownError = new Error('Programmierfehler');
    const source = new BingXOpenPositionSource(createFailingClient(unknownError));

    await expect(source.fetchOpenPositions()).rejects.toBe(unknownError);
  });
});
