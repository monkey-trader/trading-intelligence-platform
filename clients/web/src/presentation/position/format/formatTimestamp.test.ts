import { describe, expect, it } from 'vitest';
import { formatDateTime, formatTimestamp } from '@/presentation/position/format/formatTimestamp';

describe('formatTimestamp', () => {
  it('zeigt die Uhrzeit in Europe/Zurich während der Sommerzeit (UTC+2)', () => {
    expect(formatTimestamp(Date.UTC(2026, 9, 16, 12, 34, 56))).toBe('14:34:56');
  });

  it('zeigt die Uhrzeit in Europe/Zurich während der Winterzeit (UTC+1)', () => {
    expect(formatTimestamp(Date.UTC(2026, 0, 15, 12, 0, 0))).toBe('13:00:00');
  });

  it('zeigt Stunden zweistellig, auch nach Mitternacht', () => {
    expect(formatTimestamp(Date.UTC(2026, 9, 16, 22, 30, 0))).toBe('00:30:00');
  });
});

describe('formatDateTime', () => {
  it('zeigt Datum und Uhrzeit (ohne Sekunden) in Europe/Zurich', () => {
    expect(formatDateTime(1791500000000)).toBe('09.10.2026, 00:53');
    expect(formatDateTime(Date.UTC(2026, 0, 15, 12, 0, 0))).toBe('15.01.2026, 13:00');
  });
});
