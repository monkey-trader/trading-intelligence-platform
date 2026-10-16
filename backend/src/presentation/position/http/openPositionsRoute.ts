import type { FastifyInstance, FastifyReply } from 'fastify';
import type { OpenPosition } from '@/domain/position/entities/OpenPosition';
import { mapOpenPositionsError } from '@/presentation/position/http/errorMapper';
import {
  type OpenPositionDto,
  toOpenPositionDto,
} from '@/presentation/position/http/toOpenPositionDto';

export type GetOpenPositions = () => Promise<OpenPosition[]>;

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
    openTime: { type: 'integer' },
    updateTime: { type: 'integer' },
  },
} as const;

/** Entspricht `Problem` in docs/openapi.yaml. */
const PROBLEM_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['type', 'title', 'status'],
  properties: {
    type: {
      type: 'string',
      enum: [
        'not-configured',
        'rate-limited',
        'unreachable',
        'upstream-error',
        'invalid-response',
        'internal',
      ],
    },
    title: { type: 'string' },
    status: { type: 'integer' },
    detail: { type: 'string' },
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
          200: { type: 'array', items: OPEN_POSITION_SCHEMA },
          500: PROBLEM_SCHEMA,
          502: PROBLEM_SCHEMA,
          503: PROBLEM_SCHEMA,
          504: PROBLEM_SCHEMA,
        },
      },
    },
    async (request, reply): Promise<OpenPositionDto[] | FastifyReply> => {
      try {
        const positions = await getOpenPositions();
        return positions.map(toOpenPositionDto);
      } catch (error: unknown) {
        const mapped = mapOpenPositionsError(error);
        // Die Ursache (inkl. cause) landet nur im Server-Log, nie im Body.
        request.log.error({ err: error }, 'Offene Positionen konnten nicht geladen werden.');
        if (mapped.retryAfterSeconds !== undefined) {
          reply.header('retry-after', String(mapped.retryAfterSeconds));
        }
        return reply.code(mapped.status).type('application/problem+json').send(mapped.problem);
      }
    },
  );
}
