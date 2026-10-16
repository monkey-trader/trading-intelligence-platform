import { createHmac } from 'node:crypto';
import { describe, expect, it } from 'vitest';
import { signQuery } from '@/infrastructure/bingx/signing';

describe('signQuery', () => {
  it('liefert den bekannten HMAC-SHA256-Testvektor als Hex', () => {
    expect(signQuery('The quick brown fox jumps over the lazy dog', 'key')).toBe(
      'f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8',
    );
  });

  it('stimmt mit node:crypto für einen BingX-Query-String überein', () => {
    const query =
      'symbol=ONDO-USDT&startTs=1780000000000&endTs=1780100000000&timestamp=1780100000123';
    const secret = 'test-secret-abc';
    const expected = createHmac('sha256', secret).update(query).digest('hex');

    expect(signQuery(query, secret)).toBe(expected);
  });

  it('ergibt für verschiedene Queries verschiedene Signaturen', () => {
    expect(signQuery('timestamp=1', 'secret')).not.toBe(signQuery('timestamp=2', 'secret'));
  });
});
