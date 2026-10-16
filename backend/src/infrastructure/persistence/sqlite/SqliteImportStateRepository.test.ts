import type { DatabaseSync } from 'node:sqlite';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { openDatabase } from '@/infrastructure/persistence/sqlite/openDatabase';
import { SqliteImportStateRepository } from '@/infrastructure/persistence/sqlite/SqliteImportStateRepository';

describe('SqliteImportStateRepository', () => {
  let database: DatabaseSync;
  let repository: SqliteImportStateRepository;

  beforeEach(() => {
    database = openDatabase(':memory:');
    repository = new SqliteImportStateRepository(database);
  });

  afterEach(() => {
    database.close();
  });

  it('liefert undefined für einen unbekannten Schlüssel', async () => {
    await expect(repository.get('income_scan_until')).resolves.toBeUndefined();
  });

  it('speichert einen Wert und liest ihn exakt wieder', async () => {
    await repository.set('income_scan_until', 1_790_000_000_123);

    await expect(repository.get('income_scan_until')).resolves.toBe(1_790_000_000_123);
  });

  it('überschreibt einen vorhandenen Wert', async () => {
    await repository.set('position_history_until:BTC-USDT', 1_780_000_000_000);

    await repository.set('position_history_until:BTC-USDT', 1_790_000_000_000);

    await expect(repository.get('position_history_until:BTC-USDT')).resolves.toBe(
      1_790_000_000_000,
    );
  });

  it('bleibt über eine neue Repository-Instanz auf derselben Datenbank erhalten', async () => {
    await repository.set('income_scan_until', 1_790_000_000_000);

    const secondRepository = new SqliteImportStateRepository(database);

    await expect(secondRepository.get('income_scan_until')).resolves.toBe(1_790_000_000_000);
  });

  it('listet nur die Schlüssel mit dem angegebenen Präfix', async () => {
    await repository.set('income_scan_until', 1);
    await repository.set('position_history_until:BTC-USDT', 2);
    await repository.set('position_history_until:ETH-USDT', 3);

    const keys = await repository.listKeys('position_history_until:');

    expect([...keys].sort()).toEqual([
      'position_history_until:BTC-USDT',
      'position_history_until:ETH-USDT',
    ]);
  });

  it('behandelt _ und % im Präfix nicht als Platzhalter', async () => {
    await repository.set('position_history_until:BTC-USDT', 1);
    await repository.set('positionXhistoryXuntil:ETH-USDT', 2);
    await repository.set('position%history_until:XRP-USDT', 3);

    const keys = await repository.listKeys('position_history_until:');

    expect(keys).toEqual(['position_history_until:BTC-USDT']);
  });

  it('löscht einen Schlüssel, andere bleiben erhalten', async () => {
    await repository.set('position_suspicious:ETH-USDT', 1_780_000_000_000);
    await repository.set('position_suspicious:SEI-USDT', 1_780_000_000_000);

    await repository.delete('position_suspicious:ETH-USDT');

    await expect(repository.get('position_suspicious:ETH-USDT')).resolves.toBeUndefined();
    await expect(repository.get('position_suspicious:SEI-USDT')).resolves.toBe(1_780_000_000_000);
  });

  it('ignoriert das Löschen eines unbekannten Schlüssels', async () => {
    await expect(repository.delete('gibt_es_nicht')).resolves.toBeUndefined();
  });

  it('liefert eine leere Liste, wenn kein Schlüssel passt', async () => {
    await repository.set('income_scan_until', 1);

    await expect(repository.listKeys('position_history_until:')).resolves.toEqual([]);
  });
});
