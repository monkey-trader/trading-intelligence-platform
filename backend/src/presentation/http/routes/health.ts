import type { FastifyInstance } from 'fastify';

export type HealthResponse = { status: 'ok' };

/** GET /health – Lebenszeichen des Backends, siehe docs/openapi.yaml. */
export function registerHealthRoute(server: FastifyInstance): void {
  server.get(
    '/health',
    {
      schema: {
        response: {
          200: {
            type: 'object',
            properties: { status: { type: 'string', enum: ['ok'] } },
            required: ['status'],
            additionalProperties: false,
          },
        },
      },
    },
    async (): Promise<HealthResponse> => ({ status: 'ok' }),
  );
}
