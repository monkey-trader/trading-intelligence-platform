import {
  request as httpRequest,
  type IncomingHttpHeaders,
  type OutgoingHttpHeaders,
} from 'node:http';
import { request as httpsRequest } from 'node:https';

export type HttpRequest = Readonly<{
  method: string;
  path: string;
  /** Wird unverändert gesendet, auch `host` (fetch würde den Host-Header verwerfen). */
  headers?: Readonly<Record<string, string>>;
  /** Optionaler Body, roh als UTF-8 gesendet (auch absichtlich kaputtes JSON). */
  body?: string;
}>;

export type HttpResponse = Readonly<{
  status: number;
  headers: IncomingHttpHeaders;
  /** Roher Body als UTF-8-Text, leer bei HEAD oder ohne Inhalt. */
  body: string;
}>;

const REQUEST_TIMEOUT_MS = 10_000;

/**
 * Schickt eine Anfrage über node:http(s) an die Basis-URL. Der Pfad geht unverändert auf die
 * Leitung (keine Normalisierung oder Kodierung), damit auch ungültige Pfade wie `/%zz` testbar sind.
 */
export function request(
  baseUrl: URL,
  { method, path, headers = {}, body }: HttpRequest,
): Promise<HttpResponse> {
  const send = baseUrl.protocol === 'https:' ? httpsRequest : httpRequest;
  const outgoingHeaders: OutgoingHttpHeaders = { ...headers };
  return new Promise<HttpResponse>((resolve, reject) => {
    const clientRequest = send(
      {
        protocol: baseUrl.protocol,
        hostname: baseUrl.hostname.replace(/^\[(.*)\]$/, '$1'),
        port: baseUrl.port === '' ? undefined : Number(baseUrl.port),
        method,
        path,
        headers: outgoingHeaders,
        timeout: REQUEST_TIMEOUT_MS,
      },
      (incoming) => {
        const chunks: Buffer[] = [];
        incoming.on('data', (chunk: Buffer) => chunks.push(chunk));
        incoming.on('error', reject);
        incoming.on('end', () =>
          resolve({
            status: incoming.statusCode ?? 0,
            headers: incoming.headers,
            body: Buffer.concat(chunks).toString('utf8'),
          }),
        );
      },
    );
    clientRequest.on('timeout', () => {
      clientRequest.destroy(
        new Error(`Zeitüberschreitung nach ${REQUEST_TIMEOUT_MS} ms: ${method} ${path}`),
      );
    });
    clientRequest.on('error', reject);
    clientRequest.end(body);
  });
}

/** Einzelner Header-Wert; bei mehrfach gesendeten Headern undefined (nicht eindeutig). */
export function singleHeader(headers: IncomingHttpHeaders, name: string): string | undefined {
  const value = headers[name.toLowerCase()];
  return typeof value === 'string' ? value : undefined;
}

/** Parst den Body als JSON; leerer Body ergibt undefined. */
export function parseJsonBody(response: HttpResponse): unknown {
  if (response.body === '') return undefined;
  return JSON.parse(response.body) as unknown;
}
