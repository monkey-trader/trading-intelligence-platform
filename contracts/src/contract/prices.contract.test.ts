import { request as httpRequest, type IncomingHttpHeaders } from 'node:http';
import { request as httpsRequest } from 'node:https';
import { describe, expect, it } from 'vitest';
import { createContractContext } from '@/contract/contractContext';
import { OPERATION_TEST_NAMES } from '@/contract/coverage';
import { singleHeader } from '@/httpClient';
import { mediaTypeOf } from '@/validator';

const { config, validator } = createContractContext();

const FIRST_EVENT_TIMEOUT_MS = 8_000;

type FirstEvent = Readonly<{
  status: number;
  headers: IncomingHttpHeaders;
  eventName: string | undefined;
  data: string;
}>;

/** Öffnet den Stream, liest bis zum ersten vollständigen Event (Leerzeile) und trennt dann. */
function readFirstEvent(): Promise<FirstEvent> {
  const send = config.baseUrl.protocol === 'https:' ? httpsRequest : httpRequest;
  return new Promise<FirstEvent>((resolve, reject) => {
    const clientRequest = send(
      {
        protocol: config.baseUrl.protocol,
        hostname: config.baseUrl.hostname.replace(/^\[(.*)\]$/, '$1'),
        port: config.baseUrl.port === '' ? undefined : Number(config.baseUrl.port),
        method: 'GET',
        path: '/prices/stream',
        headers: { accept: 'text/event-stream' },
      },
      (incoming) => {
        let buffer = '';
        incoming.setEncoding('utf8');
        incoming.on('data', (chunk: string) => {
          buffer += chunk;
          // Kommentare (Heartbeat, beginnen mit ':') überspringen, nur echte Events zählen.
          for (const block of buffer.split('\n\n').slice(0, -1)) {
            const lines = block.split('\n').filter((line) => !line.startsWith(':'));
            if (lines.length === 0) continue;
            const eventName = lines.find((line) => line.startsWith('event:'))?.slice(6).trim();
            const data = lines
              .filter((line) => line.startsWith('data:'))
              .map((line) => line.slice(5).trim())
              .join('\n');
            clientRequest.destroy();
            resolve({ status: incoming.statusCode ?? 0, headers: incoming.headers, eventName, data });
            return;
          }
        });
        incoming.on('error', () => undefined);
      },
    );
    clientRequest.setTimeout(FIRST_EVENT_TIMEOUT_MS, () => {
      clientRequest.destroy();
      reject(new Error('Kein Event innerhalb von 8 s'));
    });
    clientRequest.on('error', reject);
    clientRequest.end();
  });
}

describe(OPERATION_TEST_NAMES.streamPrices, () => {
  it('antwortet mit 200 und text/event-stream', async () => {
    const first = await readFirstEvent();

    expect(first.status).toBe(200);
    expect(mediaTypeOf(singleHeader(first.headers, 'content-type'))).toBe('text/event-stream');
  });

  it('sendet als erstes Event "prices" mit einem Body laut PriceSnapshot', async () => {
    const first = await readFirstEvent();

    expect(first.eventName).toBe('prices');
    const result = validator.validateSchema('PriceSnapshot', JSON.parse(first.data) as unknown);
    expect(result.errors).toEqual([]);
    expect(result.valid).toBe(true);
  });
});
