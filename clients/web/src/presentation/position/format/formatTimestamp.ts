import { DISPLAY_LOCALE } from '@/presentation/position/format/formatDecimal';

const DISPLAY_TIME_ZONE = 'Europe/Zurich';

const timeFormat = new Intl.DateTimeFormat(DISPLAY_LOCALE, {
  timeZone: DISPLAY_TIME_ZONE,
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hourCycle: 'h23',
});

const dateTimeFormat = new Intl.DateTimeFormat(DISPLAY_LOCALE, {
  timeZone: DISPLAY_TIME_ZONE,
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
});

/** Uhrzeit mit Sekunden in Europe/Zurich, z.B. 14:34:56. */
export function formatTimestamp(epochMilliseconds: number): string {
  return timeFormat.format(epochMilliseconds);
}

/** Datum und Uhrzeit ohne Sekunden in Europe/Zurich, z.B. 09.10.2026, 00:53. */
export function formatDateTime(epochMilliseconds: number): string {
  return dateTimeFormat.format(epochMilliseconds);
}
