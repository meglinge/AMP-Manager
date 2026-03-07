package database

import (
	"fmt"
	"strings"
)

type DBType string

const (
	DBTypeSQLite   DBType = "sqlite"
	DBTypePostgres DBType = "postgres"
)

type Options struct {
	Type        DBType
	DatabaseURL string
	SQLitePath  string
}

func (options Options) Normalize() (Options, error) {
	normalized := options
	normalized.Type = DBType(strings.TrimSpace(strings.ToLower(string(normalized.Type))))
	if normalized.Type == "" {
		normalized.Type = DBTypeSQLite
	}

	switch normalized.Type {
	case DBTypeSQLite:
		if strings.TrimSpace(normalized.SQLitePath) == "" {
			normalized.SQLitePath = "./data/data.db"
		}
	case DBTypePostgres:
		if strings.TrimSpace(normalized.DatabaseURL) == "" {
			return normalized, fmt.Errorf("DATABASE_URL is required when DB_TYPE=postgres")
		}
	default:
		return normalized, fmt.Errorf("unsupported DB_TYPE: %s", normalized.Type)
	}

	return normalized, nil
}
