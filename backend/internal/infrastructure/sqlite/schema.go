package sqlite

// schemaStatements legt das Schema idempotent an (siehe docs/database/schema.md).
var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS positions (
		position_id TEXT PRIMARY KEY,
		symbol TEXT NOT NULL,
		side TEXT NOT NULL,
		isolated INTEGER NOT NULL,
		open_time INTEGER NOT NULL,
		close_time INTEGER NOT NULL,
		avg_open_price TEXT NOT NULL,
		avg_close_price TEXT NOT NULL,
		realised_profit TEXT NOT NULL,
		net_profit TEXT NOT NULL,
		position_amount TEXT NOT NULL,
		closed_amount TEXT NOT NULL,
		commission TEXT NOT NULL,
		total_funding TEXT NOT NULL,
		leverage REAL NOT NULL,
		fully_closed INTEGER NOT NULL,
		raw TEXT NOT NULL,
		imported_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS positions_symbol_close_time ON positions (symbol, close_time)`,
	`CREATE TABLE IF NOT EXISTS notes (
		position_id TEXT PRIMARY KEY REFERENCES positions (position_id),
		setup TEXT,
		rating INTEGER,
		mistakes TEXT,
		notes TEXT,
		updated_at INTEGER
	)`,
	`CREATE TABLE IF NOT EXISTS income (
		transaction_id TEXT PRIMARY KEY,
		symbol TEXT NOT NULL,
		income_type TEXT NOT NULL,
		amount TEXT NOT NULL,
		time INTEGER NOT NULL,
		asset TEXT NOT NULL,
		trade_id TEXT NOT NULL,
		raw TEXT NOT NULL,
		imported_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS income_symbol_time ON income (symbol, time)`,
	`CREATE INDEX IF NOT EXISTS income_type_time ON income (income_type, time)`,
	`CREATE TABLE IF NOT EXISTS candles (
		symbol TEXT NOT NULL,
		interval TEXT NOT NULL,
		open_time INTEGER NOT NULL,
		open TEXT NOT NULL,
		high TEXT NOT NULL,
		low TEXT NOT NULL,
		close TEXT NOT NULL,
		volume TEXT NOT NULL,
		PRIMARY KEY (symbol, interval, open_time)
	)`,
	`CREATE TABLE IF NOT EXISTS import_state (
		key TEXT PRIMARY KEY,
		value INTEGER NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS position_risk (
		position_id TEXT PRIMARY KEY,
		entry_price TEXT NOT NULL,
		initial_stop_loss TEXT NOT NULL,
		first_seen_at INTEGER NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS position_initial_stop_override (
		position_id TEXT PRIMARY KEY,
		entry_price TEXT NOT NULL,
		initial_stop_loss TEXT NOT NULL,
		set_at INTEGER NOT NULL
	)`,
}
