package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	db     *sql.DB
	dbPath string
	mu     sync.RWMutex
	inited bool
)

func Init(path string) error {
	mu.Lock()
	defer mu.Unlock()
	return initDB(path)
}

func initDB(path string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	dsn := path + "?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	newDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	if err = newDB.Ping(); err != nil {
		newDB.Close()
		return err
	}

	newDB.SetMaxOpenConns(10)
	newDB.SetMaxIdleConns(5)
	newDB.SetConnMaxLifetime(time.Hour)

	db = newDB
	dbPath = path
	inited = true

	if err := createTables(); err != nil {
		return err
	}
	return runMigrations()
}

// CloseAndRelease 关闭数据库连接并释放所有文件句柄，以便替换数据库文件
func CloseAndRelease() error {
	mu.Lock()
	defer mu.Unlock()
	if db != nil {
		_, _ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
		err := db.Close()
		db = nil
		return err
	}
	return nil
}

// Reopen 重新打开数据库（文件替换后调用）
func Reopen() error {
	mu.Lock()
	defer mu.Unlock()
	if dbPath == "" {
		return fmt.Errorf("database path not set, call Init first")
	}
	return initDB(dbPath)
}

func GetDB() *sql.DB {
	return db
}

func createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS groups (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_groups_name ON groups(name);

	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		is_admin INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

	CREATE TABLE IF NOT EXISTS user_amp_settings (
		id TEXT PRIMARY KEY,
		user_id TEXT UNIQUE NOT NULL,
		upstream_url TEXT NOT NULL DEFAULT '',
		upstream_api_key TEXT NOT NULL DEFAULT '',
		model_mappings_json TEXT NOT NULL DEFAULT '[]',
		force_model_mappings INTEGER DEFAULT 0,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_amp_settings_user_id ON user_amp_settings(user_id);

	CREATE TABLE IF NOT EXISTS user_api_keys (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		key_hash TEXT UNIQUE NOT NULL,
		api_key TEXT NOT NULL DEFAULT '',
		prefix TEXT NOT NULL,
		last_used_at DATETIME,
		expires_at DATETIME,
		revoked_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON user_api_keys(user_id);
	CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON user_api_keys(key_hash);

	CREATE TABLE IF NOT EXISTS channels (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		endpoint TEXT NOT NULL DEFAULT 'chat_completions',
		name TEXT NOT NULL,
		base_url TEXT NOT NULL,
		api_key TEXT NOT NULL DEFAULT '',
		enabled INTEGER NOT NULL DEFAULT 1,
		weight INTEGER NOT NULL DEFAULT 1,
		priority INTEGER NOT NULL DEFAULT 100,
		models_json TEXT NOT NULL DEFAULT '[]',
		headers_json TEXT NOT NULL DEFAULT '{}',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_channels_type_enabled ON channels(type, enabled);
	CREATE INDEX IF NOT EXISTS idx_channels_enabled ON channels(enabled);

	CREATE TABLE IF NOT EXISTS user_groups (
		user_id TEXT NOT NULL,
		group_id TEXT NOT NULL,
		PRIMARY KEY (user_id, group_id),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_user_groups_user ON user_groups(user_id);
	CREATE INDEX IF NOT EXISTS idx_user_groups_group ON user_groups(group_id);

	CREATE TABLE IF NOT EXISTS channel_groups (
		channel_id TEXT NOT NULL,
		group_id TEXT NOT NULL,
		PRIMARY KEY (channel_id, group_id),
		FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
		FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_channel_groups_channel ON channel_groups(channel_id);
	CREATE INDEX IF NOT EXISTS idx_channel_groups_group ON channel_groups(group_id);

	CREATE TABLE IF NOT EXISTS channel_models (
		id TEXT PRIMARY KEY,
		channel_id TEXT NOT NULL,
		model_id TEXT NOT NULL,
		display_name TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_channel_models_channel ON channel_models(channel_id);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_channel_models_unique ON channel_models(channel_id, model_id);

	CREATE TABLE IF NOT EXISTS model_metadata (
		id TEXT PRIMARY KEY,
		model_pattern TEXT UNIQUE NOT NULL,
		display_name TEXT NOT NULL DEFAULT '',
		context_length INTEGER NOT NULL DEFAULT 200000,
		max_completion_tokens INTEGER NOT NULL DEFAULT 32000,
		provider TEXT NOT NULL DEFAULT '',
		is_builtin INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_model_metadata_pattern ON model_metadata(model_pattern);
	CREATE INDEX IF NOT EXISTS idx_model_metadata_provider ON model_metadata(provider);

	CREATE TABLE IF NOT EXISTS request_logs (
		id TEXT PRIMARY KEY,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		user_id TEXT NOT NULL,
		api_key_id TEXT NOT NULL,
		original_model TEXT,
		mapped_model TEXT,
		provider TEXT,
		channel_id TEXT,
		endpoint TEXT,
		method TEXT NOT NULL,
		path TEXT NOT NULL,
		status_code INTEGER NOT NULL,
		latency_ms INTEGER NOT NULL,
		is_streaming INTEGER NOT NULL DEFAULT 0,
		input_tokens INTEGER,
		output_tokens INTEGER,
		cache_read_input_tokens INTEGER,
		cache_creation_input_tokens INTEGER,
		error_type TEXT,
		request_id TEXT,
		cost_micros INTEGER,
		cost_usd TEXT,
		pricing_model TEXT,
		thinking_level TEXT,
		response_text TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_request_logs_user_time ON request_logs(user_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_request_logs_apikey_time ON request_logs(api_key_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_request_logs_model_time ON request_logs(mapped_model, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_request_logs_time ON request_logs(created_at DESC);

	CREATE TABLE IF NOT EXISTS model_prices (
		id TEXT PRIMARY KEY,
		model TEXT UNIQUE NOT NULL,
		provider TEXT,
		price_data TEXT NOT NULL DEFAULT '{}',
		source TEXT NOT NULL DEFAULT 'manual',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_model_prices_model ON model_prices(model);
	CREATE INDEX IF NOT EXISTS idx_model_prices_provider ON model_prices(provider);

	CREATE TABLE IF NOT EXISTS system_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS request_log_details (
		request_id TEXT PRIMARY KEY,
		request_headers TEXT,
		request_body TEXT,
		response_headers TEXT,
		response_body TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_request_log_details_created ON request_log_details(created_at DESC);

	CREATE TABLE IF NOT EXISTS subscription_plans (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_subscription_plans_enabled ON subscription_plans(enabled);

	CREATE TABLE IF NOT EXISTS subscription_plan_limits (
		id TEXT PRIMARY KEY,
		plan_id TEXT NOT NULL,
		limit_type TEXT NOT NULL CHECK (limit_type IN ('daily', 'weekly', 'monthly', 'rolling_5h', 'total')),
		window_mode TEXT NOT NULL DEFAULT 'fixed' CHECK (window_mode IN ('fixed', 'sliding')),
		limit_micros INTEGER NOT NULL CHECK (limit_micros >= 0),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (plan_id) REFERENCES subscription_plans(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_plan_limits_plan ON subscription_plan_limits(plan_id);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_plan_limits_unique ON subscription_plan_limits(plan_id, limit_type);

	CREATE TABLE IF NOT EXISTS user_subscriptions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		plan_id TEXT NOT NULL,
		starts_at DATETIME NOT NULL,
		expires_at DATETIME,
		status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'expired', 'cancelled')),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (plan_id) REFERENCES subscription_plans(id) ON DELETE RESTRICT
	);
	CREATE INDEX IF NOT EXISTS idx_user_subs_user ON user_subscriptions(user_id);
	CREATE INDEX IF NOT EXISTS idx_user_subs_plan ON user_subscriptions(plan_id);
	CREATE INDEX IF NOT EXISTS idx_user_subs_status ON user_subscriptions(status);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_user_subs_active_unique ON user_subscriptions(user_id) WHERE status = 'active';

	CREATE TABLE IF NOT EXISTS user_billing_settings (
		user_id TEXT PRIMARY KEY,
		primary_source TEXT NOT NULL DEFAULT 'subscription' CHECK (primary_source IN ('subscription', 'balance')),
		secondary_source TEXT NOT NULL DEFAULT 'balance' CHECK (secondary_source IN ('subscription', 'balance')),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		CHECK (primary_source != secondary_source),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS billing_events (
		id TEXT PRIMARY KEY,
		request_log_id TEXT,
		user_id TEXT NOT NULL,
		user_subscription_id TEXT,
		source TEXT NOT NULL CHECK (source IN ('subscription', 'balance')),
		event_type TEXT NOT NULL CHECK (event_type IN ('charge', 'refund', 'adjustment')),
		amount_micros INTEGER NOT NULL CHECK (amount_micros >= 0),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (user_subscription_id) REFERENCES user_subscriptions(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_billing_events_user ON billing_events(user_id);
	CREATE INDEX IF NOT EXISTS idx_billing_events_sub ON billing_events(user_subscription_id);
	CREATE INDEX IF NOT EXISTS idx_billing_events_request ON billing_events(request_log_id);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_billing_events_idempotent ON billing_events(request_log_id, source, event_type) WHERE request_log_id IS NOT NULL;
	`
	_, err := db.Exec(schema)
	return err
}

func runMigrations() error {
	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "add_api_key_column",
			sql:  `ALTER TABLE user_api_keys ADD COLUMN api_key TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: "migrate_proxy_tokens",
			sql: `INSERT INTO user_api_keys (id, user_id, name, key_hash, prefix, last_used_at, expires_at, revoked_at, created_at)
				SELECT id, user_id, name, token_hash, prefix, last_used_at, expires_at, revoked_at, created_at
				FROM user_proxy_tokens
				WHERE NOT EXISTS (SELECT 1 FROM user_api_keys WHERE user_api_keys.id = user_proxy_tokens.id)`,
		},
		{
			name: "add_channels_endpoint",
			sql:  `ALTER TABLE channels ADD COLUMN endpoint TEXT NOT NULL DEFAULT 'chat_completions'`,
		},
		{
			name: "add_request_logs_status",
			sql:  `ALTER TABLE request_logs ADD COLUMN status TEXT NOT NULL DEFAULT 'success'`,
		},
		{
			name: "add_request_logs_updated_at",
			sql:  `ALTER TABLE request_logs ADD COLUMN updated_at DATETIME`,
		},
		{
			name: "add_request_logs_status_index",
			sql:  `CREATE INDEX IF NOT EXISTS idx_request_logs_status ON request_logs(status)`,
		},
		{
			name: "add_original_model_index",
			sql:  `CREATE INDEX IF NOT EXISTS idx_request_logs_original_model_time ON request_logs(original_model, created_at DESC)`,
		},
		{
			name: "add_request_logs_response_text",
			sql:  `ALTER TABLE request_logs ADD COLUMN response_text TEXT`,
		},
		{
			name: "add_channels_enabled_priority_index",
			sql:  `CREATE INDEX IF NOT EXISTS idx_channels_enabled_priority ON channels(enabled, priority ASC, weight DESC)`,
		},
		{
			name: "add_web_search_mode",
			sql:  `ALTER TABLE user_amp_settings ADD COLUMN web_search_mode TEXT NOT NULL DEFAULT 'upstream'`,
		},
		{
			name: "add_request_logs_cost_micros",
			sql:  `ALTER TABLE request_logs ADD COLUMN cost_micros INTEGER`,
		},
		{
			name: "add_request_logs_cost_usd",
			sql:  `ALTER TABLE request_logs ADD COLUMN cost_usd TEXT`,
		},
		{
			name: "add_request_logs_pricing_model",
			sql:  `ALTER TABLE request_logs ADD COLUMN pricing_model TEXT`,
		},
		{
			name: "add_request_logs_thinking_level",
			sql:  `ALTER TABLE request_logs ADD COLUMN thinking_level TEXT`,
		},
		{
			name: "add_native_mode",
			sql:  `ALTER TABLE user_amp_settings ADD COLUMN native_mode INTEGER NOT NULL DEFAULT 0`,
		},
		{
			name: "add_user_group_id",
			sql:  `ALTER TABLE users ADD COLUMN group_id TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: "add_channel_group_id",
			sql:  `ALTER TABLE channels ADD COLUMN group_id TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: "create_user_groups_table",
			sql: `CREATE TABLE IF NOT EXISTS user_groups (
				user_id TEXT NOT NULL,
				group_id TEXT NOT NULL,
				PRIMARY KEY (user_id, group_id),
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
				FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
			)`,
		},
		{
			name: "create_user_groups_indexes",
			sql: `CREATE INDEX IF NOT EXISTS idx_user_groups_user ON user_groups(user_id);
				  CREATE INDEX IF NOT EXISTS idx_user_groups_group ON user_groups(group_id)`,
		},
		{
			name: "create_channel_groups_table",
			sql: `CREATE TABLE IF NOT EXISTS channel_groups (
				channel_id TEXT NOT NULL,
				group_id TEXT NOT NULL,
				PRIMARY KEY (channel_id, group_id),
				FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
				FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
			)`,
		},
		{
			name: "create_channel_groups_indexes",
			sql: `CREATE INDEX IF NOT EXISTS idx_channel_groups_channel ON channel_groups(channel_id);
				  CREATE INDEX IF NOT EXISTS idx_channel_groups_group ON channel_groups(group_id)`,
		},
		{
			name: "add_group_rate_multiplier",
			sql:  `ALTER TABLE groups ADD COLUMN rate_multiplier REAL NOT NULL DEFAULT 1.0`,
		},
		{
			name: "add_user_balance_micros",
			sql:  `ALTER TABLE users ADD COLUMN balance_micros INTEGER NOT NULL DEFAULT 0`,
		},
		{
			name: "add_request_logs_rate_multiplier",
			sql:  `ALTER TABLE request_logs ADD COLUMN rate_multiplier REAL`,
		},
		{
			name: "add_request_logs_charged_subscription_micros",
			sql:  `ALTER TABLE request_logs ADD COLUMN charged_subscription_micros INTEGER NOT NULL DEFAULT 0`,
		},
		{
			name: "add_request_logs_charged_balance_micros",
			sql:  `ALTER TABLE request_logs ADD COLUMN charged_balance_micros INTEGER NOT NULL DEFAULT 0`,
		},
		{
			name: "add_request_logs_billing_status",
			sql:  `ALTER TABLE request_logs ADD COLUMN billing_status TEXT NOT NULL DEFAULT 'none'`,
		},
		{
			name: "add_billing_events_usage_window_index",
			sql:  `CREATE INDEX IF NOT EXISTS idx_billing_events_usage_window ON billing_events(user_subscription_id, source, created_at)`,
		},
		{
			name: "add_show_balance_in_ad",
			sql:  `ALTER TABLE user_amp_settings ADD COLUMN show_balance_in_ad INTEGER NOT NULL DEFAULT 0`,
		},
	}

	for _, m := range migrations {
		_, err := db.Exec(m.sql)
		if err != nil {
			// ALTER TABLE 和 INSERT 迁移可能因为已存在而失败，这是正常的
			// 只有 CREATE INDEX IF NOT EXISTS 类型的迁移失败才是真正的错误
			if m.name == "add_original_model_index" || m.name == "add_channels_enabled_priority_index" || m.name == "add_request_logs_status_index" || m.name == "create_user_groups_indexes" || m.name == "create_channel_groups_indexes" || m.name == "add_billing_events_usage_window_index" {
				return fmt.Errorf("migration '%s' failed: %w", m.name, err)
			}
		}
	}

	if err := migrateTimestampsToUTC(db); err != nil {
		return fmt.Errorf("migrate timestamps to UTC failed: %w", err)
	}

	return nil
}

// migrateTimestampsToUTC 将数据库中所有带时区偏移的 RFC3339 时间戳转换为 UTC
func migrateTimestampsToUTC(db *sql.DB) error {
	var done int
	err := db.QueryRow(`SELECT COUNT(*) FROM system_config WHERE key = 'migration_timestamps_utc'`).Scan(&done)
	if err != nil {
		return err
	}
	if done > 0 {
		return nil
	}

	type tableCol struct {
		table  string
		column string
	}
	targets := []tableCol{
		{"request_logs", "created_at"},
		{"request_logs", "updated_at"},
		{"request_log_details", "created_at"},
		{"users", "created_at"},
		{"users", "updated_at"},
		{"channels", "created_at"},
		{"channels", "updated_at"},
		{"groups", "created_at"},
		{"groups", "updated_at"},
		{"user_api_keys", "created_at"},
		{"user_api_keys", "last_used_at"},
		{"user_amp_settings", "created_at"},
		{"user_amp_settings", "updated_at"},
		{"channel_models", "created_at"},
		{"model_metadata", "created_at"},
		{"model_metadata", "updated_at"},
		{"model_prices", "created_at"},
		{"model_prices", "updated_at"},
		{"system_config", "updated_at"},
	}

	for _, tc := range targets {
		rows, err := db.Query(fmt.Sprintf(
			`SELECT rowid, %s FROM %s WHERE %s IS NOT NULL AND %s NOT LIKE '%%Z' AND %s LIKE '%%+%%'`,
			tc.column, tc.table, tc.column, tc.column, tc.column,
		))
		if err != nil {
			continue
		}

		type row struct {
			rowid int64
			ts    string
		}
		var toUpdate []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.rowid, &r.ts); err != nil {
				continue
			}
			toUpdate = append(toUpdate, r)
		}
		rows.Close()

		if len(toUpdate) == 0 {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		stmt, err := tx.Prepare(fmt.Sprintf(`UPDATE %s SET %s = ? WHERE rowid = ?`, tc.table, tc.column))
		if err != nil {
			tx.Rollback()
			return err
		}
		for _, r := range toUpdate {
			t, err := time.Parse(time.RFC3339, r.ts)
			if err != nil {
				continue
			}
			_, _ = stmt.Exec(t.UTC().Format(time.RFC3339), r.rowid)
		}
		stmt.Close()
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	_, err = db.Exec(`INSERT OR REPLACE INTO system_config (key, value, updated_at) VALUES ('migration_timestamps_utc', '1', ?)`, time.Now().UTC())
	return err
}

func Close() error {
	mu.Lock()
	defer mu.Unlock()
	if db != nil {
		err := db.Close()
		db = nil
		return err
	}
	return nil
}
