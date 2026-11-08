export type SwingLongProblemType =
  'bad-request' | 'forbidden-origin' | 'internal' | 'backend-unreachable' | 'invalid-body';

export type SwingLongResult =
  Readonly<{ kind: 'ok' }> | Readonly<{ kind: 'problem'; problemType: SwingLongProblemType }>;

/** Port zum Setzen und Entfernen des Swing-Long-Flags einer Position. */
export type SwingLongGateway = Readonly<{
  setSwingLong: (positionId: string) => Promise<SwingLongResult>;
  clearSwingLong: (positionId: string) => Promise<SwingLongResult>;
}>;
