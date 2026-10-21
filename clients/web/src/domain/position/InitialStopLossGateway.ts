export type InitialStopLossProblemType =
  | 'invalid-stop-loss'
  | 'not-found'
  | 'bad-request'
  | 'forbidden-origin'
  | 'not-configured'
  | 'rate-limited'
  | 'unreachable'
  | 'upstream-error'
  | 'invalid-response'
  | 'internal'
  | 'backend-unreachable'
  | 'invalid-body';

export type InitialStopLossResult =
  Readonly<{ kind: 'ok' }> | Readonly<{ kind: 'problem'; problemType: InitialStopLossProblemType }>;

/** Port zum manuellen Eintragen und Zurücksetzen des ursprünglichen Stop-Loss. */
export type InitialStopLossGateway = Readonly<{
  setInitialStopLoss: (positionId: string, value: string) => Promise<InitialStopLossResult>;
  clearInitialStopLoss: (positionId: string) => Promise<InitialStopLossResult>;
}>;
