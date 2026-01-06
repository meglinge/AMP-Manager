package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	db   *sql.DB
	once sync.Once
)

func Init(dbPath string) error {
	var err error
	once.Do(func() {
		// 确保数据目录存在
		dir := filepath.Dir(dbPath)
		if dir != "" && dir != "." {
			if err = os.MkdirAll(dir, 0755); err != nil {
				return
			}
		}

		// 添加连接参数：WAL模式、忙等待超时、共享缓存
		dsn := dbPath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
		db, err = sql.Open("sqlite", dsn)
		if err != nil {
			return
		}
		if err = db.Ping(); err != nil {
			return
		}

		// 限制连接池大小，SQLite 单写多读
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		err = createTables()
		if err != nil {
			return
		}
		err = runMigrations()
	})
	return err
}

func GetDB() *sql.DB {
	return db
}

func createTables() error {
	schema := `
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
		request_id TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_request_logs_user_time ON request_logs(user_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_request_logs_apikey_time ON request_logs(api_key_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_request_logs_model_time ON request_logs(mapped_model, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_request_logs_time ON request_logs(created_at DESC);

	CREATE TABLE IF NOT EXISTS system_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := db.Exec(schema)
	return err
}

func runMigrations() error {
	_, _ = db.Exec(`
		ALTER TABLE user_api_keys ADD COLUMN api_key TEXT NOT NULL DEFAULT ''
	`)

	_, _ = db.Exec(`
		INSERT INTO user_api_keys (id, user_id, name, key_hash, prefix, last_used_at, expires_at, revoked_at, created_at)
		SELECT id, user_id, name, token_hash, prefix, last_used_at, expires_at, revoked_at, created_at
		FROM user_proxy_tokens
		WHERE NOT EXISTS (SELECT 1 FROM user_api_keys WHERE user_api_keys.id = user_proxy_tokens.id)
	`)

	_, _ = db.Exec(`ALTER TABLE channels ADD COLUMN endpoint TEXT NOT NULL DEFAULT 'chat_completions'`)

	// Add status and updated_at columns for pending request tracking
	_, _ = db.Exec(`ALTER TABLE request_logs ADD COLUMN status TEXT NOT NULL DEFAULT 'success'`)
	_, _ = db.Exec(`ALTER TABLE request_logs ADD COLUMN updated_at DATETIME`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_request_logs_status ON request_logs(status)`)

	return nil
}

func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
