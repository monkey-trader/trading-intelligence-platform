import { createHmac } from 'node:crypto';

/** Signiert einen Query-String für BingX mit HMAC-SHA256 und liefert die Signatur als Hex. */
export function signQuery(query: string, secret: string): string {
  return createHmac('sha256', secret).update(query).digest('hex');
}
