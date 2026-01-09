package repository

import (
	"database/sql"
	"time"

	"ampmanager/internal/database"
	"ampmanager/internal/model"

	"github.com/google/uuid"
)

type ChannelRepositoryInterface interface {
	Create(channel *model.Channel) error
	GetByID(id string) (*model.Channel, error)
	List() ([]*model.Channel, error)
	ListEnabled() ([]*model.Channel, error)
	Update(channel *model.Channel) error
	Delete(id string) error
	SetEnabled(id string, enabled bool) error
}

var _ ChannelRepositoryInterface = (*ChannelRepository)(nil)

type ChannelRepository struct{}

func NewChannelRepository() *ChannelRepository {
	return &ChannelRepository{}
}

func (r *ChannelRepository) Create(channel *model.Channel) error {
	db := database.GetDB()
	channel.ID = uuid.New().String()
	now := time.Now()
	channel.CreatedAt = now
	channel.UpdatedAt = now

	_, err := db.Exec(
		`INSERT INTO channels (id, type, endpoint, name, base_url, api_key, enabled, weight, priority, models_json, headers_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		channel.ID, channel.Type, channel.Endpoint, channel.Name, channel.BaseURL, channel.APIKey,
		channel.Enabled, channel.Weight, channel.Priority, channel.ModelsJSON, channel.HeadersJSON,
		channel.CreatedAt, channel.UpdatedAt,
	)
	return err
}

func (r *ChannelRepository) GetByID(id string) (*model.Channel, error) {
	db := database.GetDB()
	channel := &model.Channel{}

	err := db.QueryRow(
		`SELECT id, type, endpoint, name, base_url, api_key, enabled, weight, priority, models_json, headers_json, created_at, updated_at
		 FROM channels WHERE id = ?`,
		id,
	).Scan(
		&channel.ID, &channel.Type, &channel.Endpoint, &channel.Name, &channel.BaseURL, &channel.APIKey,
		&channel.Enabled, &channel.Weight, &channel.Priority, &channel.ModelsJSON, &channel.HeadersJSON,
		&channel.CreatedAt, &channel.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return channel, nil
}

func (r *ChannelRepository) List() ([]*model.Channel, error) {
	db := database.GetDB()
	rows, err := db.Query(
		`SELECT id, type, endpoint, name, base_url, api_key, enabled, weight, priority, models_json, headers_json, created_at, updated_at
		 FROM channels ORDER BY priority ASC, created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*model.Channel
	for rows.Next() {
		channel := &model.Channel{}
		err := rows.Scan(
			&channel.ID, &channel.Type, &channel.Endpoint, &channel.Name, &channel.BaseURL, &channel.APIKey,
			&channel.Enabled, &channel.Weight, &channel.Priority, &channel.ModelsJSON, &channel.HeadersJSON,
			&channel.CreatedAt, &channel.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	return channels, rows.Err()
}

func (r *ChannelRepository) ListEnabled() ([]*model.Channel, error) {
	db := database.GetDB()
	rows, err := db.Query(
		`SELECT id, type, endpoint, name, base_url, api_key, enabled, weight, priority, models_json, headers_json, created_at, updated_at
		 FROM channels WHERE enabled = 1 ORDER BY priority ASC, weight DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*model.Channel
	for rows.Next() {
		channel := &model.Channel{}
		err := rows.Scan(
			&channel.ID, &channel.Type, &channel.Endpoint, &channel.Name, &channel.BaseURL, &channel.APIKey,
			&channel.Enabled, &channel.Weight, &channel.Priority, &channel.ModelsJSON, &channel.HeadersJSON,
			&channel.CreatedAt, &channel.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	return channels, rows.Err()
}

func (r *ChannelRepository) Update(channel *model.Channel) error {
	db := database.GetDB()
	channel.UpdatedAt = time.Now()

	_, err := db.Exec(
		`UPDATE channels SET type = ?, endpoint = ?, name = ?, base_url = ?, api_key = ?, enabled = ?, weight = ?, priority = ?, models_json = ?, headers_json = ?, updated_at = ?
		 WHERE id = ?`,
		channel.Type, channel.Endpoint, channel.Name, channel.BaseURL, channel.APIKey, channel.Enabled, channel.Weight, channel.Priority, channel.ModelsJSON, channel.HeadersJSON, channel.UpdatedAt,
		channel.ID,
	)
	return err
}

func (r *ChannelRepository) Delete(id string) error {
	db := database.GetDB()
	_, err := db.Exec(`DELETE FROM channels WHERE id = ?`, id)
	return err
}

func (r *ChannelRepository) SetEnabled(id string, enabled bool) error {
	db := database.GetDB()
	_, err := db.Exec(`UPDATE channels SET enabled = ?, updated_at = ? WHERE id = ?`, enabled, time.Now(), id)
	return err
}
