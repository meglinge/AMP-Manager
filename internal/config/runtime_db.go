package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"ampmanager/internal/database"
)

const (
	devRuntimeDatabaseConfigEnv  = "AMP_DEV_RUNTIME_DB_CONFIG"
	devRuntimeDatabaseConfigPath = "./data/config.json"
)

var defaultDevDatabaseOptions = database.Options{
	Type:        database.DBTypePostgres,
	DatabaseURL: "postgres://postgres:mysecretpassword@localhost:5432/ampmanager?sslmode=disable",
	SQLitePath:  "./data/data.db",
}

func loadRuntimeDatabaseOptions() (database.Options, bool, error) {
	if os.Getenv(devRuntimeDatabaseConfigEnv) != "true" {
		return database.Options{}, false, nil
	}

	raw, err := os.ReadFile(devRuntimeDatabaseConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			normalized, normalizeErr := defaultDevDatabaseOptions.Normalize()
			return normalized, true, normalizeErr
		}
		return database.Options{}, false, err
	}

	var options database.Options
	if err := json.Unmarshal(raw, &options); err != nil {
		return database.Options{}, false, err
	}

	normalized, err := options.Normalize()
	if err != nil {
		return database.Options{}, false, err
	}
	return normalized, true, nil
}

func SaveRuntimeDatabaseOptions(options database.Options) error {
	if os.Getenv(devRuntimeDatabaseConfigEnv) != "true" {
		return nil
	}

	normalized, err := options.Normalize()
	if err != nil {
		return err
	}

	directory := filepath.Dir(devRuntimeDatabaseConfigPath)
	if directory != "" && directory != "." {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return err
		}
	}

	encoded, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(devRuntimeDatabaseConfigPath, encoded, 0o644)
}
