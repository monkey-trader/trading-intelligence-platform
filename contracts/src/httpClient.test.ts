import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest';
import { createServer, type IncomingMessage, type Server } from 'node:http';
import type { AddressInfo } from 'node:net';
import { parseJsonBody, request, singleHeader } from '@/httpClient';

/** Lokaler Echo-Server statt echter API: spiegelt Methode, Pfad und Host-Header zurück. */
let echoServer: Server;
let baseUrl: URL;
let lastRequest: IncomingMessage | undefined;
let lastRequestBody = '';

beforeAll(async () => {
  echoServer = createServer((incoming, outgoing) => {
    lastRequest = incoming;
    const bodyChunks: Buffer[] = [];
    incoming.on('data', (chunk: Buffer) => bodyChunks.push(chunk));
    incoming.on('end', () => {
      lastRequestBody = Buffer.concat(bodyChunks).toString('utf8');
    });
    if (incoming.url === '/empty') {
      outgoing.writeHead(204).end();
      return;
    }
    const body = JSON.stringify({
      method: incoming.method,
      path: incoming.url,
      host: incoming.headers.host,
    });
    outgoing.writeHead(200, { 'content-type': 'application/json; charset=utf-8', 'x-demo': '1' });
    outgoing.end(incoming.method === 'HEAD' ? undefined : body);
  });
  await new Promise<void>((resolve) => echoServer.listen(0, '127.0.0.1', resolve));
  const { port } = echoServer.address() as AddressInfo;
  baseUrl = new URL(`http://127.0.0.1:${port}`);
});

afterAll(async () => {
  await new Promise<void>((resolve) => echoServer.close(() => resolve()));
});

describe('request', () => {
  it('liefert Status, Header und Body', async () => {
    const response = await request(baseUrl, { method: 'GET', path: '/health' });

    expect(response.status).toBe(200);
    expect(singleHeader(response.headers, 'Content-Type')).toBe('application/json; charset=utf-8');
    expect(parseJsonBody(response)).toEqual({
      method: 'GET',
      path: '/health',
      host: `127.0.0.1:${baseUrl.port}`,
    });
  });

  it('sendet einen eigenen Host-Header unverändert (auch in Großbuchstaben)', async () => {
    const response = await request(baseUrl, {
      method: 'GET',
      path: '/x',
      headers: { host: `LOCALHOST:${baseUrl.port}` },
    });

    expect(parseJsonBody(response)).toMatchObject({ host: `LOCALHOST:${baseUrl.port}` });
    expect(lastRequest?.headers.host).toBe(`LOCALHOST:${baseUrl.port}`);
  });

  it('sendet einen fremden Host-Header', async () => {
    const response = await request(baseUrl, {
      method: 'GET',
      path: '/x',
      headers: { host: 'evil.example:3000' },
    });

    expect(parseJsonBody(response)).toMatchObject({ host: 'evil.example:3000' });
  });

  it('unterstützt HEAD und POST', async () => {
    const headResponse = await request(baseUrl, { method: 'HEAD', path: '/health' });
    const postResponse = await request(baseUrl, { method: 'POST', path: '/health' });

    expect(headResponse.status).toBe(200);
    expect(headResponse.body).toBe('');
    expect(parseJsonBody(headResponse)).toBeUndefined();
    expect(parseJsonBody(postResponse)).toMatchObject({ method: 'POST' });
  });

  it.each(['/%zz', '/a%2Fb', '/x/../health', '/x?y=%zz'])(
    'sendet den Pfad %s unverändert (keine Normalisierung, keine Kodierung)',
    async (rawPath) => {
      const response = await request(baseUrl, { method: 'GET', path: rawPath });

      expect(lastRequest?.url).toBe(rawPath);
      expect(parseJsonBody(response)).toMatchObject({ path: rawPath });
    },
  );

  it('sendet einen Body roh, auch kaputtes JSON', async () => {
    await request(baseUrl, {
      method: 'POST',
      path: '/nope',
      headers: { 'content-type': 'application/json' },
      body: '{',
    });

    expect(lastRequest?.headers['content-type']).toBe('application/json');
    await vi.waitFor(() => expect(lastRequestBody).toBe('{'));
  });

  it('sendet ohne Body auch keinen', async () => {
    await request(baseUrl, { method: 'POST', path: '/health' });

    await vi.waitFor(() => expect(lastRequestBody).toBe(''));
  });

  it('liefert bei leerem Body einen leeren String', async () => {
    const response = await request(baseUrl, { method: 'GET', path: '/empty' });

    expect(response.status).toBe(204);
    expect(response.body).toBe('');
  });

  it('lehnt ab, wenn niemand zuhört', async () => {
    const closedServer = createServer();
    await new Promise<void>((resolve) => closedServer.listen(0, '127.0.0.1', resolve));
    const { port } = closedServer.address() as AddressInfo;
    await new Promise<void>((resolve) => closedServer.close(() => resolve()));

    await expect(
      request(new URL(`http://127.0.0.1:${port}`), { method: 'GET', path: '/health' }),
    ).rejects.toThrow();
  });
});

describe('singleHeader', () => {
  it('liefert undefined für fehlende oder mehrfache Header', () => {
    expect(singleHeader({}, 'retry-after')).toBeUndefined();
    expect(singleHeader({ 'set-cookie': ['a', 'b'] }, 'set-cookie')).toBeUndefined();
  });
});
