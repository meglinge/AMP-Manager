package repository

import (
	"database/sql"
	"time"

	"ampmanager/internal/database"
)

type SystemConfigRepository struct{}

func NewSystemConfigRepository() *SystemConfigRepository {
	return &SystemConfigRepository{}
}

func (r *SystemConfigRepository) Get(key string) (string, error) {
	db := database.GetDB()
	var value string
	err := db.QueryRow("SELECT value FROM system_config WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (r *SystemConfigRepository) Set(key, value string) error {
	db := database.GetDB()
	_, err := db.Exec(`
		INSERT INTO system_config (key, value, updated_at) 
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, time.Now().UTC())
	return err
}

func (r *SystemConfigRepository) Delete(key string) error {
	db := database.GetDB()
	_, err := db.Exec("DELETE FROM system_config WHERE key = ?", key)
	return err
}
