/**
 * Abdeckung des Vertrags: jede operationId aus docs/openapi.yaml braucht einen Vertragstest.
 * Der Wert ist der Name des describe-Blocks im zugehörigen *.contract.test.ts.
 * coverage.test.ts prüft ohne Server, dass Spec und Map übereinstimmen.
 */
export const OPERATION_TEST_NAMES = {
  getHealth: 'GET /health (getHealth)',
  getOpenPositions: 'GET /positions/open (getOpenPositions)',
  getPositionChart: 'GET /positions/{positionId}/chart (getPositionChart)',
  getAssetIcons: 'GET /assets/icons (getAssetIcons)',
} as const satisfies Readonly<Record<string, string>>;
