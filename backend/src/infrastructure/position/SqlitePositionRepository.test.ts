import type { DatabaseSync } from 'node:sqlite';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { openDatabase } from '@/infrastructure/persistence/sqlite/openDatabase';
import { SqlitePositionRepository } from '@/infrastructure/position/SqlitePositionRepository';
import type { Position } from '@/domain/position/entities/Position';
import type { PositionRecord } from '@/domain/position/interfaces/PositionRecord';

function positionRecord(overrides: Partial<Position> = {}): PositionRecord {
  const position: Position = {
    positionId: '9000000000000000001',
    symbol: 'ONDO-USDT',
    side: 'LONG',
    isolated: true,
    openTime: 1780000000000,
    closeTime: 1780018017000,
    avgOpenPrice: '0.4120',
    avgClosePrice: '0.4188',
    realisedProfit: '1.7000',
    netProfit: '1.5123',
    positionAmount: '250.00',
    closedAmount: '250.00',
    commission: '-0.05121506352000001',
    totalFunding: '-0.1356',
    leverage: 7,
    fullyClosed: true,
    ...overrides,
  };
  return { position, rawPayload: JSON.stringify({ ...position, source: 'test' }) };
}

type StoredRow = Record<string, unknown>;

describe('SqlitePositionRepository', () => {
  let database: DatabaseSync;
  let repository: SqlitePositionRepository;

  const readRow = (positionId: string): StoredRow | undefined =>
    database.prepare('SELECT * FROM positions WHERE position_id = ?').get(positionId) as
      StoredRow | undefined;
  const countPositions = (): number =>
    Number(
      (database.prepare('SELECT COUNT(*) AS positionCount FROM positions').get() as StoredRow)
        .positionCount,
    );

  beforeEach(() => {
    database = openDatabase(':memory:');
    repository = new SqlitePositionRepository(database);
  });

  afterEach(() => {
    database.close();
  });

  it('fügt neue Positionen ein und liefert die Anzahl der verarbeiteten Datensätze', async () => {
    const savedCount = await repository.upsertMany([
      positionRecord({ positionId: '9000000000000000001' }),
      positionRecord({ positionId: '9000000000000000002' }),
    ]);

    expect(savedCount).toBe(2);
    expect(countPositions()).toBe(2);
  });

  it('liefert 0 und schreibt nichts bei einer leeren Liste', async () => {
    await expect(repository.upsertMany([])).resolves.toBe(0);
    expect(countPositions()).toBe(0);
  });

  it('aktualisiert eine vorhandene Position statt sie zu duplizieren (gleiche ID zweimal)', async () => {
    await repository.upsertMany([positionRecord({ netProfit: '1.5123' })]);

    await repository.upsertMany([positionRecord({ netProfit: '1.9999', avgClosePrice: '0.4200' })]);

    expect(countPositions()).toBe(1);
    const row = readRow('9000000000000000001');
    expect(row?.net_profit).toBe('1.9999');
    expect(row?.avg_close_price).toBe('0.4200');
  });

  it('dedupliziert dieselbe ID innerhalb eines Aufrufs (Überlapp-Fenster)', async () => {
    await repository.upsertMany([positionRecord(), positionRecord()]);

    expect(countPositions()).toBe(1);
  });

  it('lässt eine vorhandene Notiz nach erneutem Upsert unverändert', async () => {
    await repository.upsertMany([positionRecord()]);
    database
      .prepare(
        'INSERT INTO notes (position_id, setup, rating, mistakes, notes, updated_at) VALUES (?, ?, ?, ?, ?, ?)',
      )
      .run('9000000000000000001', 'Breakout', 4, 'zu früh raus', 'Meine Notiz', 1780020000000);
    const noteBefore = database
      .prepare('SELECT * FROM notes WHERE position_id = ?')
      .get('9000000000000000001');

    await repository.upsertMany([positionRecord({ netProfit: '2.0000' })]);

    const noteAfter = database
      .prepare('SELECT * FROM notes WHERE position_id = ?')
      .get('9000000000000000001');
    expect(noteAfter).toEqual(noteBefore);
    expect(noteAfter).toMatchObject({ setup: 'Breakout', notes: 'Meine Notiz' });
    expect(readRow('9000000000000000001')?.net_profit).toBe('2.0000');
  });

  it('gibt Beträge als exakt denselben String zurück', async () => {
    await repository.upsertMany([positionRecord()]);

    const row = readRow('9000000000000000001');
    expect(row?.commission).toBe('-0.05121506352000001');
    expect(row?.avg_open_price).toBe('0.4120');
    expect(row?.position_amount).toBe('250.00');
    expect(row?.total_funding).toBe('-0.1356');
  });

  it('speichert eine 19-stellige positionId exakt', async () => {
    await repository.upsertMany([positionRecord({ positionId: '9223372036854775807' })]);

    const row = readRow('9223372036854775807');
    expect(row?.position_id).toBe('9223372036854775807');
    expect(typeof row?.position_id).toBe('string');
  });

  it('speichert den rawPayload unverändert in raw', async () => {
    const record = positionRecord();

    await repository.upsertMany([record]);

    expect(readRow('9000000000000000001')?.raw).toBe(record.rawPayload);
  });

  it('speichert die Fachfelder in snake_case-Spalten', async () => {
    await repository.upsertMany([positionRecord()]);

    expect(readRow('9000000000000000001')).toMatchObject({
      symbol: 'ONDO-USDT',
      side: 'LONG',
      open_time: 1780000000000,
      close_time: 1780018017000,
      realised_profit: '1.7000',
      closed_amount: '250.00',
      leverage: 7,
    });
  });
});
