import { afterEach, describe, expect, it } from 'vitest';
import Fastify, { type FastifyInstance } from 'fastify';
import { isAllowedHost, registerHostGuard } from '@/presentation/http/hostGuard';

const ALLOWED_HOSTS: ReadonlySet<string> = new Set([
  '127.0.0.1:3000',
  'localhost:3000',
  '[::1]:3000',
]);

describe('isAllowedHost', () => {
  it.each(['127.0.0.1:3000', 'localhost:3000', '[::1]:3000'])('erlaubt %s', (hostHeader) => {
    expect(isAllowedHost(hostHeader, ALLOWED_HOSTS)).toBe(true);
  });

  it.each(['LOCALHOST:3000', 'LocalHost:3000'])(
    'vergleicht ohne Groß-/Kleinschreibung (%s)',
    (hostHeader) => {
      expect(isAllowedHost(hostHeader, ALLOWED_HOSTS)).toBe(true);
    },
  );

  it.each(['localhost:3001', 'localhost', '127.0.0.1:80', '127.0.0.1'])(
    'lehnt einen falschen oder fehlenden Port ab (%s)',
    (hostHeader) => {
      expect(isAllowedHost(hostHeader, ALLOWED_HOSTS)).toBe(false);
    },
  );

  it.each(['evil.example:3000', 'localhost.evil.example:3000', '192.168.1.10:3000', ''])(
    'lehnt den fremden Host "%s" ab',
    (hostHeader) => {
      expect(isAllowedHost(hostHeader, ALLOWED_HOSTS)).toBe(false);
    },
  );

  it.each([
    '[::1]',
    'localhost:03000',
    'localhost.:3000',
    '127.0.0.1:3000.evil.example',
    ' localhost:3000',
  ])('lehnt die Umgehungsvariante "%s" ab', (hostHeader) => {
    expect(isAllowedHost(hostHeader, ALLOWED_HOSTS)).toBe(false);
  });

  it('lehnt einen fehlenden Host-Header ab', () => {
    expect(isAllowedHost(undefined, ALLOWED_HOSTS)).toBe(false);
  });

  it('lehnt alles ab, wenn keine Hosts erlaubt sind', () => {
    expect(isAllowedHost('localhost:3000', new Set())).toBe(false);
  });
});

describe('registerHostGuard', () => {
  let server: FastifyInstance | undefined;

  afterEach(async () => {
    await server?.close();
    server = undefined;
  });

  /** Kleine Fastify-Instanz mit Guard und einer Route, die ihre Aufrufe zählt. */
  function createGuardedServer(allowedHosts: ReadonlySet<string>) {
    let handlerCalls = 0;
    const guardedServer = Fastify();
    registerHostGuard(guardedServer, allowedHosts);
    guardedServer.get('/ping', async () => {
      handlerCalls += 1;
      return { pong: true };
    });
    return { guardedServer, getHandlerCalls: () => handlerCalls };
  }

  it('weist einen fremden Host mit 403 forbidden-host als application/problem+json ab', async () => {
    const { guardedServer, getHandlerCalls } = createGuardedServer(ALLOWED_HOSTS);
    server = guardedServer;

    const response = await server.inject({
      method: 'GET',
      url: '/ping',
      headers: { host: 'evil.example:3000' },
    });

    expect(response.statusCode).toBe(403);
    expect(response.headers['content-type']).toMatch(/^application\/problem\+json/);
    expect(response.json()).toEqual({
      type: 'forbidden-host',
      title: expect.any(String),
      status: 403,
    });
    expect(getHandlerCalls()).toBe(0);
  });

  it('gibt den abgelehnten Host nicht im Body zurück', async () => {
    const { guardedServer } = createGuardedServer(ALLOWED_HOSTS);
    server = guardedServer;

    const response = await server.inject({
      method: 'GET',
      url: '/ping',
      headers: { host: 'evil.example:3000' },
    });

    expect(response.body).not.toContain('evil.example');
  });

  it('weist auch unbekannte Pfade bei fremdem Host mit 403 ab', async () => {
    const { guardedServer } = createGuardedServer(ALLOWED_HOSTS);
    server = guardedServer;

    const response = await server.inject({
      method: 'GET',
      url: '/gibt-es-nicht',
      headers: { host: 'evil.example:3000' },
    });

    expect(response.statusCode).toBe(403);
  });

  it('lässt einen erlaubten Host durch (200)', async () => {
    const { guardedServer, getHandlerCalls } = createGuardedServer(ALLOWED_HOSTS);
    server = guardedServer;

    const response = await server.inject({
      method: 'GET',
      url: '/ping',
      headers: { host: 'localhost:3000' },
    });

    expect(response.statusCode).toBe(200);
    expect(response.json()).toEqual({ pong: true });
    expect(getHandlerCalls()).toBe(1);
  });

  it('lässt einen erlaubten Host in anderer Schreibweise durch', async () => {
    const { guardedServer } = createGuardedServer(ALLOWED_HOSTS);
    server = guardedServer;

    const response = await server.inject({
      method: 'GET',
      url: '/ping',
      headers: { host: 'LOCALHOST:3000' },
    });

    expect(response.statusCode).toBe(200);
  });

  it('weist den inject-Standard-Host localhost:80 ab, wenn er nicht erlaubt ist', async () => {
    const { guardedServer } = createGuardedServer(ALLOWED_HOSTS);
    server = guardedServer;

    const response = await server.inject({ method: 'GET', url: '/ping' });

    expect(response.statusCode).toBe(403);
  });
});
