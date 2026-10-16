import type { Position } from '@/domain/position/entities/Position';

/** Position plus die unveränderte Rohantwort der Börse (JSON), damit nichts verloren geht. */
export type PositionRecord = Readonly<{ position: Position; rawPayload: string }>;
