// Composition Root: einziger Ort, der Konfiguration liest, Adapter baut und den Server startet.
// Konfiguration, Key und Secret werden nie geloggt.
import { GetOpenPositionsUseCase } from '@/application/position/GetOpenPositionsUseCase';
import type { OpenPositionSource } from '@/domain/position/interfaces/OpenPositionSource';
import { BingXClient } from '@/infrastructure/bingx/BingXClient';
import { missingBingXVariables, readBingXConfig } from '@/infrastructure/config/bingxConfig';
import { readServerConfig } from '@/infrastructure/config/serverConfig';
import { BingXOpenPositionSource } from '@/infrastructure/position/BingXOpenPositionSource';
import { UnconfiguredOpenPositionSource } from '@/infrastructure/position/UnconfiguredOpenPositionSource';
import { createServer } from '@/presentation/http/createServer';

const { host, port } = readServerConfig(process.env);
const bingXConfig = readBingXConfig(process.env);

// Eigener Client für die HTTP-API: wenige Retries und kurzer Timeout, damit ein Request nicht hängt.
const openPositionSource: OpenPositionSource =
  bingXConfig === undefined
    ? new UnconfiguredOpenPositionSource()
    : new BingXOpenPositionSource(
        new BingXClient({
          ...bingXConfig,
          maxRetries: 1,
          requestTimeoutMs: 5_000,
          requestPauseMs: 0,
        }),
      );
const getOpenPositionsUseCase = new GetOpenPositionsUseCase(openPositionSource);

const server = createServer(
  { getOpenPositions: () => getOpenPositionsUseCase.execute() },
  { logger: true },
);

if (bingXConfig === undefined) {
  server.log.warn(
    `${missingBingXVariables(process.env).join(', ')} nicht gesetzt: GET /positions/open antwortet mit 503 not-configured.`,
  );
}

try {
  await server.listen({ port, host });
} catch (error: unknown) {
  server.log.error(error);
  process.exit(1);
}
