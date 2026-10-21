// Erfundene, anonymisierte Testdaten wie in den Backend-Fixtures (SUI-USDT LONG, Gold SHORT).
import type { OpenPosition, OpenPositionsSnapshot } from '@/domain/position/OpenPosition';

/** 16.10.2026 12:34:56 UTC = 14:34:56 in Europe/Zurich (Sommerzeit). */
export const FETCHED_AT = Date.UTC(2026, 9, 16, 12, 34, 56);

export const suiLongPosition: OpenPosition = {
  positionId: '9100000000000000001',
  symbol: 'SUI-USDT',
  side: 'LONG',
  isolated: true,
  positionAmount: '500',
  avgOpenPrice: '0.9600',
  markPrice: '1.0100',
  leverage: 10,
  initialMargin: '48.0000',
  margin: '60.5000',
  positionValue: '505',
  unrealizedProfit: '25.0000',
  realisedProfit: '-0.2400',
  pnlRatio: '0.5208',
  stopLoss: '0.90',
  takeProfit: '1.20',
  breakEven: false,
  initialStopLoss: null,
  initialStopLossSource: null,
  riskDistance: null,
  currentRMultiple: null,
  breakEvenTriggerReached: null,
  reached2RAt: null,
  openTime: 1791500000000,
  updateTime: 1791600000000,
};

export const goldShortPosition: OpenPosition = {
  positionId: '9100000000000000002',
  symbol: 'NCCOGOLD2USD-USDT',
  side: 'SHORT',
  isolated: true,
  positionAmount: '0.25',
  avgOpenPrice: '4150.00',
  markPrice: '4175.00',
  leverage: 15,
  initialMargin: '69.1667',
  margin: '65.0000',
  positionValue: '1043.75',
  unrealizedProfit: '-6.2500',
  realisedProfit: '-0.5187',
  pnlRatio: '-0.0904',
  stopLoss: '4200',
  takeProfit: '4100',
  breakEven: false,
  initialStopLoss: null,
  initialStopLossSource: null,
  riskDistance: null,
  currentRMultiple: null,
  breakEvenTriggerReached: null,
  reached2RAt: null,
  openTime: 1791700000000,
  updateTime: 1791710000000,
};

/** Position mit PnL genau 0. */
export const flatPosition: OpenPosition = {
  positionId: '9100000000000000003',
  symbol: 'BTC-USDT',
  side: 'LONG',
  isolated: false,
  positionAmount: '0.001',
  avgOpenPrice: '60000.0',
  markPrice: '60000.0',
  leverage: 1,
  initialMargin: '60.0000',
  margin: '60.0000',
  positionValue: '60',
  unrealizedProfit: '0.0000',
  realisedProfit: '0',
  pnlRatio: '0',
  stopLoss: null,
  takeProfit: null,
  breakEven: null,
  initialStopLoss: null,
  initialStopLossSource: null,
  riskDistance: null,
  currentRMultiple: null,
  breakEvenTriggerReached: null,
  reached2RAt: null,
  openTime: 1791400000000,
  updateTime: 1791400000000,
};

/** Neueste zuerst, wie vom Backend sortiert. Summe 25 − 6.25 = 18.75. */
export const sampleSnapshot: OpenPositionsSnapshot = {
  positions: [goldShortPosition, suiLongPosition],
  totalUnrealizedProfit: '18.75',
  fetchedAt: FETCHED_AT,
};

export const emptySnapshot: OpenPositionsSnapshot = {
  positions: [],
  totalUnrealizedProfit: '0',
  fetchedAt: FETCHED_AT,
};

/** Stop-Loss über dem Einstieg (LONG): Break-even. */
export const breakEvenPosition: OpenPosition = {
  ...suiLongPosition,
  stopLoss: '0.97',
  breakEven: true,
};

/** Orders konnten nicht gelesen werden: alles unbekannt. */
export const unknownOrdersPosition: OpenPosition = {
  ...suiLongPosition,
  stopLoss: null,
  takeProfit: null,
  breakEven: null,
};

/** Orders bekannt, aber kein Stop-Loss gesetzt. */
export const noStopLossPosition: OpenPosition = {
  ...suiLongPosition,
  stopLoss: null,
  takeProfit: null,
  breakEven: false,
};
