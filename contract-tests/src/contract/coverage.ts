/**
 * Abdeckung des Vertrags: jede operationId aus docs/openapi.yaml braucht einen Vertragstest.
 * Der Wert ist der Name des describe-Blocks im zugehörigen *.contract.test.ts.
 * coverage.test.ts prüft ohne Server, dass Spec und Map übereinstimmen.
 */
export const OPERATION_TEST_NAMES = {
  getHealth: 'GET /health (getHealth)',
  getOpenPositions: 'GET /positions/open (getOpenPositions)',
  getAccountSummary: 'GET /account/summary (getAccountSummary)',
  getPositionChart: 'GET /positions/{positionId}/chart (getPositionChart)',
  getAssetIcons: 'GET /assets/icons (getAssetIcons)',
  setInitialStopLoss: 'PUT /positions/{positionId}/initial-stop-loss (setInitialStopLoss)',
  clearInitialStopLoss: 'DELETE /positions/{positionId}/initial-stop-loss (clearInitialStopLoss)',
} as const satisfies Readonly<Record<string, string>>;
