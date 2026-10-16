// Composition Root: einziger Ort, der Konfiguration liest, Adapter baut und den Server startet.
import { readServerConfig } from '@/infrastructure/config/serverConfig';
import { createServer } from '@/presentation/http/createServer';

const { host, port } = readServerConfig(process.env);
const server = createServer({ logger: true });

try {
  await server.listen({ port, host });
} catch (error: unknown) {
  server.log.error(error);
  process.exit(1);
}
