import type { FastifyInstance, FastifyReply } from 'fastify';
import type { OpenPositionsOverview } from '@/application/position/GetOpenPositionsUseCase';
import { PROBLEM_CONTENT_TYPE, PROBLEM_SCHEMA } from '@/presentation/http/problem';
import { mapOpenPositionsError } from '@/presentation/position/http/errorMapper';
import {
  type OpenPositionsSnapshotDto,
  toOpenPositionsSnapshotDto,
} from '@/presentation/position/http/toOpenPositionDto';

export type GetOpenPositions = () => Promise<OpenPositionsOverview>;

const DECIMAL_SCHEMA = { type: 'string', pattern: '^-?\\d+(\\.\\d+)?$' } as const;

/** Entspricht `OpenPosition` in docs/openapi.yaml. */
const OPEN_POSITION_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: [
    'positionId',
    'symbol',
    'side',
    'isolated',
    'positionAmount',
    'avgOpenPrice',
    'markPrice',
    'leverage',
    'initialMargin',
    'margin',
    'positionValue',
    'unrealizedProfit',
    'realisedProfit',
    'pnlRatio',
    'liquidationPrice',
    'liquidationDistanceRatio',
    'openTime',
    'updateTime',
  ],
  properties: {
    positionId: { type: 'string' },
    symbol: { type: 'string' },
    side: { type: 'string', enum: ['LONG', 'SHORT'] },
    isolated: { type: 'boolean' },
    positionAmount: DECIMAL_SCHEMA,
    avgOpenPrice: DECIMAL_SCHEMA,
    markPrice: DECIMAL_SCHEMA,
    leverage: { type: 'number', exclusiveMinimum: 0 },
    initialMargin: DECIMAL_SCHEMA,
    margin: DECIMAL_SCHEMA,
    positionValue: DECIMAL_SCHEMA,
    unrealizedProfit: DECIMAL_SCHEMA,
    realisedProfit: DECIMAL_SCHEMA,
    pnlRatio: DECIMAL_SCHEMA,
    liquidationPrice: DECIMAL_SCHEMA,
    // Decimal oder null (ohne Liquidationspreis); das Pattern greift nur bei Strings.
    liquidationDistanceRatio: { ...DECIMAL_SCHEMA, type: ['string', 'null'] },
    openTime: { type: 'integer' },
    updateTime: { type: 'integer' },
  },
} as const;

/** Entspricht `OpenPositionsSnapshot` in docs/openapi.yaml. */
const OPEN_POSITIONS_SNAPSHOT_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['positions', 'totalUnrealizedProfit', 'fetchedAt'],
  properties: {
    positions: { type: 'array', items: OPEN_POSITION_SCHEMA },
    totalUnrealizedProfit: DECIMAL_SCHEMA,
    fetchedAt: { type: 'integer' },
  },
} as const;

/** GET /positions/open – aktuell offene Positionen live von BingX, siehe docs/openapi.yaml. */
export function registerOpenPositionsRoute(
  server: FastifyInstance,
  getOpenPositions: GetOpenPositions,
): void {
  server.get(
    '/positions/open',
    {
      schema: {
        response: {
          200: OPEN_POSITIONS_SNAPSHOT_SCHEMA,
          // 403 setzt der Host-Guard im onRequest-Hook, noch vor dem Handler.
          403: PROBLEM_SCHEMA,
          500: PROBLEM_SCHEMA,
          502: PROBLEM_SCHEMA,
          503: PROBLEM_SCHEMA,
          504: PROBLEM_SCHEMA,
        },
      },
    },
    async (request, reply): Promise<OpenPositionsSnapshotDto | FastifyReply> => {
      try {
        const overview = await getOpenPositions();
        return toOpenPositionsSnapshotDto(overview);
      } catch (error: unknown) {
        const mapped = mapOpenPositionsError(error);
        // Die Ursache (inkl. cause) landet nur im Server-Log, nie im Body.
        request.log.error({ err: error }, 'Offene Positionen konnten nicht geladen werden.');
        if (mapped.retryAfterSeconds !== undefined) {
          reply.header('retry-after', String(mapped.retryAfterSeconds));
        }
        return reply.code(mapped.status).type(PROBLEM_CONTENT_TYPE).send(mapped.problem);
      }
    },
  );
}
