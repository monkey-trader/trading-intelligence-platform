// Erfundene, anonymisierte Testdaten nach der Beispielposition NCCOXAG2USD-USDT (Schema PositionChart).
import type { PositionChart } from '@/domain/position/PositionChart';

export const CHART_POSITION_ID = '9100000000000000003';

/** Zwei Kerzen, bewusst aufsteigend nach time (Backend-Vertrag). */
const candles: PositionChart['candles'] = [
  {
    time: 1791700000000,
    open: '55.70',
    high: '55.90',
    low: '55.60',
    close: '55.80',
    volume: '120.5',
  },
  {
    time: 1791700900000,
    open: '55.80',
    high: '56.10',
    low: '55.75',
    close: '56.00',
    volume: '98.25',
  },
];

export const xagLongChart: PositionChart = {
  positionId: CHART_POSITION_ID,
  symbol: 'NCCOXAG2USD-USDT',
  side: 'LONG',
  interval: '15m',
  candles,
  entry: { price: '55.75', time: 1791700000000 },
  stopLoss: [
    {
      orderId: '9200000000000000001',
      triggerPrice: '55.80',
      quantity: '0.0000',
      distanceRatio: '0.00089686',
    },
  ],
  takeProfit: [
    {
      orderId: '9200000000000000002',
      triggerPrice: '63.36',
      quantity: '2.9483',
      distanceRatio: '0.13650224',
    },
  ],
  markers: [{ time: 1791700000000, kind: 'ENTRY', price: '55.75' }],
  fetchedAt: 1791700950000,
};

/** SL unter dem Entry: negativer Abstand. */
export const xagLongChartWithLossStop: PositionChart = {
  ...xagLongChart,
  stopLoss: [
    {
      orderId: '9200000000000000004',
      triggerPrice: '51.54',
      quantity: '2.9483',
      distanceRatio: '-0.07551121',
    },
  ],
};

export const xagChartWithoutProtection: PositionChart = {
  ...xagLongChart,
  stopLoss: [],
  takeProfit: [],
};

export const xagChartWithThreeTakeProfits: PositionChart = {
  ...xagLongChart,
  takeProfit: [
    {
      orderId: '9200000000000000011',
      triggerPrice: '58.00',
      quantity: '1.0000',
      distanceRatio: '0.04035874',
    },
    {
      orderId: '9200000000000000012',
      triggerPrice: '60.50',
      quantity: '1.0000',
      distanceRatio: '0.08520179',
    },
    {
      orderId: '9200000000000000013',
      triggerPrice: '63.36',
      quantity: '0.9483',
      distanceRatio: '0.13650224',
    },
  ],
};

/** Antwort zu einem anderen Intervall (Intervallwechsel). */
export const xagChartHourly: PositionChart = {
  ...xagLongChart,
  interval: '1h',
  candles: [
    {
      time: 1791698400000,
      open: '55.40',
      high: '56.20',
      low: '55.30',
      close: '56.00',
      volume: '400',
    },
  ],
};
