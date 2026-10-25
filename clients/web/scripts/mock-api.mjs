// Fake-API mit erfundenen Daten, nur für Screenshots (README). Keine echten Konto- oder Handelsdaten.
// Start: node scripts/mock-api.mjs   (Port 3299, MOCK_PORT überschreibt)
import { createServer } from 'node:http';

const now = Date.now();
const hours = (count) => count * 3_600_000;

const position = (fields) => ({
  isolated: true,
  realisedProfit: '0',
  updateTime: now - 60_000,
  breakEven: false,
  initialStopLossSource: 'AUTO',
  ...fields,
});

const levels = (entry, risk, side) =>
  [2, 3].map((multiple) => {
    const price = side === 'LONG' ? entry + risk * multiple : entry - risk * multiple;
    return {
      kind: multiple === 2 ? 'TWO_R' : 'THREE_R',
      rMultiple: multiple,
      price: price.toFixed(2),
      percent: ((Math.abs(price - entry) / entry) * 100).toFixed(2),
    };
  });

const positions = [
  position({
    positionId: '1100000000000000001',
    symbol: 'BTC-USDT',
    side: 'LONG',
    positionAmount: '0.15',
    avgOpenPrice: '67200.0',
    markPrice: '68450.0',
    leverage: 10,
    initialMargin: '1026.75',
    margin: '1026.75',
    positionValue: '10267.50',
    unrealizedProfit: '187.50',
    pnlRatio: '0.1826',
    openTime: now - hours(12),
    stopLoss: '67250.0',
    takeProfit: '70200.0',
    breakEven: true,
    initialStopLoss: '66200.0',
    riskDistance: '1000.0',
    currentRMultiple: '1.25',
    breakEvenTriggerReached: true,
    reached2RAt: now - hours(4),
    levels: levels(67200, 1000, 'LONG'),
  }),
  position({
    positionId: '1100000000000000002',
    symbol: 'ETH-USDT',
    side: 'SHORT',
    isolated: false,
    positionAmount: '2.5',
    avgOpenPrice: '3420.0',
    markPrice: '3365.0',
    leverage: 8,
    initialMargin: '1051.56',
    margin: '1051.56',
    positionValue: '8412.50',
    unrealizedProfit: '137.50',
    pnlRatio: '0.1308',
    openTime: now - hours(9),
    stopLoss: '3520.0',
    takeProfit: '3120.0',
    initialStopLoss: '3520.0',
    riskDistance: '100.0',
    currentRMultiple: '0.55',
    breakEvenTriggerReached: false,
    reached2RAt: null,
    levels: levels(3420, 100, 'SHORT'),
  }),
  position({
    positionId: '1100000000000000003',
    symbol: 'SOL-USDT',
    side: 'LONG',
    positionAmount: '40',
    avgOpenPrice: '168.40',
    markPrice: '164.90',
    leverage: 5,
    initialMargin: '1319.20',
    margin: '1319.20',
    positionValue: '6596.00',
    unrealizedProfit: '-140.00',
    pnlRatio: '-0.1061',
    openTime: now - hours(3),
    stopLoss: '161.50',
    takeProfit: '182.00',
    initialStopLoss: '161.50',
    riskDistance: '6.90',
    currentRMultiple: '-0.51',
    breakEvenTriggerReached: false,
    reached2RAt: null,
    levels: levels(168.4, 6.9, 'LONG'),
  }),
  position({
    positionId: '1100000000000000004',
    symbol: 'SUI-USDT',
    side: 'SHORT',
    positionAmount: '1800',
    avgOpenPrice: '3.215',
    markPrice: '3.152',
    leverage: 5,
    initialMargin: '1136.16',
    margin: '1136.16',
    positionValue: '5673.60',
    unrealizedProfit: '113.40',
    pnlRatio: '0.0998',
    openTime: now - hours(14),
    stopLoss: null,
    takeProfit: '2.85',
    breakEven: false,
  }),
];

const accountSummary = {
  balance: '24850.40',
  equity: '25312.70',
  availableMargin: '18400.55',
  month: '2026-10',
  monthlyRealizedPnl: '1842.30',
  monthlyPnlPercent: '7.28',
  periods: [
    { period: 'today', realizedPnl: '214.60', pnlPercent: '0.86' },
    { period: '7d', realizedPnl: '689.10', pnlPercent: '2.80' },
    { period: '30d', realizedPnl: '2204.75', pnlPercent: '9.54' },
  ],
  unrealizedPnl: '298.40',
  unrealizedPnlPercent: '1.18',
};

const INTERVAL_MS = {
  '5m': 300_000,
  '15m': 900_000,
  '1h': 3_600_000,
  '4h': 14_400_000,
  '1d': 86_400_000,
};

// Deterministische Zufallszahlen, damit der Screenshot bei jedem Lauf gleich aussieht.
const seededRandom = (seed) => () => {
  seed = (seed + 0x6d2b79f5) | 0;
  let mixed = Math.imul(seed ^ (seed >>> 15), 1 | seed);
  mixed = (mixed + Math.imul(mixed ^ (mixed >>> 7), 61 | mixed)) ^ mixed;
  return ((mixed ^ (mixed >>> 14)) >>> 0) / 4294967296;
};

/** Kerzen entlang einer Kurve: Start unter dem Einstieg, Einstieg bei 12 h, Anstieg mit Rücksetzern. */
const chartFor = (position, interval, limit) => {
  const step = INTERVAL_MS[interval] ?? INTERVAL_MS['5m'];
  const count = Math.min(limit, 200);
  const random = seededRandom(7);
  const entry = Number(position.avgOpenPrice);
  const mark = Number(position.markPrice);
  const entryIndex = Math.max(1, Math.round(count - (now - position.openTime) / step));
  const anchors = [
    [0, entry * 0.9955],
    [entryIndex, entry],
    [Math.round((entryIndex + count) / 2), entry * 1.0105],
    [count - 1, mark],
  ];
  const curve = (index) => {
    for (let at = 1; at < anchors.length; at++) {
      const [endIndex, endPrice] = anchors[at];
      const [startIndex, startPrice] = anchors[at - 1];
      if (index <= endIndex) {
        return (
          startPrice +
          ((endPrice - startPrice) * (index - startIndex)) / (endIndex - startIndex || 1)
        );
      }
    }
    return mark;
  };
  const candles = [];
  let previousClose = curve(0);
  for (let index = 0; index < count; index++) {
    const open = previousClose;
    const wave = index < entryIndex ? Math.sin(index / 7) * entry * 0.0022 : 0; // Seitwärtsphase vor dem Einstieg
    const close = curve(index) + wave + (random() - 0.5) * entry * 0.0016;
    const high = Math.max(open, close) + random() * entry * 0.0009;
    const low = Math.min(open, close) - random() * entry * 0.0009;
    candles.push({
      time: now - (count - index) * step,
      open: open.toFixed(1),
      high: high.toFixed(1),
      low: low.toFixed(1),
      close: close.toFixed(1),
      volume: (40 + random() * 160).toFixed(2),
    });
    previousClose = close;
  }
  const ratio = (price) => ((price - mark) / mark).toFixed(4);
  return {
    positionId: position.positionId,
    symbol: position.symbol,
    side: position.side,
    interval,
    candles,
    entry: { price: position.avgOpenPrice, time: position.openTime },
    stopLoss: position.stopLoss
      ? [
          {
            orderId: '2200000000000000001',
            triggerPrice: position.stopLoss,
            quantity: position.positionAmount,
            distanceRatio: ratio(Number(position.stopLoss)),
          },
        ]
      : [],
    takeProfit: position.takeProfit
      ? [
          {
            orderId: '2200000000000000002',
            triggerPrice: position.takeProfit,
            quantity: position.positionAmount,
            distanceRatio: ratio(Number(position.takeProfit)),
          },
        ]
      : [],
    markers: [{ time: position.openTime, kind: 'ENTRY', price: position.avgOpenPrice }],
    levels: position.levels ?? [],
    fetchedAt: Date.now(),
  };
};

const routes = {
  '/health': () => ({ status: 'ok' }),
  '/positions/open': () => ({ positions, totalUnrealizedProfit: '298.40', fetchedAt: Date.now() }),
  '/account/summary': () => accountSummary,
  '/assets/icons': (url) => ({
    icons: (url.searchParams.get('symbols') ?? '')
      .split(',')
      .filter(Boolean)
      .map((symbol) => {
        const baseAsset = symbol.split('-')[0];
        return { symbol, baseAsset, displayName: baseAsset, iconUrl: null };
      }),
  }),
};

const port = Number(process.env.MOCK_PORT ?? 3299);
createServer((request, response) => {
  const url = new URL(request.url ?? '/', 'http://127.0.0.1');
  const chartMatch = url.pathname.match(/^\/positions\/(\d+)\/chart$/);
  const chartPosition = chartMatch && positions.find((item) => item.positionId === chartMatch[1]);
  const handler = chartPosition
    ? () =>
        chartFor(
          chartPosition,
          url.searchParams.get('interval') ?? '5m',
          Number(url.searchParams.get('limit') ?? 200),
        )
    : routes[url.pathname];
  response.setHeader('Content-Type', handler ? 'application/json' : 'application/problem+json');
  response.statusCode = handler ? 200 : 404;
  response.end(
    JSON.stringify(
      handler ? handler(url) : { type: 'not-found', title: 'Nicht gefunden.', status: 404 },
    ),
  );
}).listen(port, '127.0.0.1', () => console.log(`Mock-API auf http://127.0.0.1:${port}`));
