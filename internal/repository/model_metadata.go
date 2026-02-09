package repository

import (
	"database/sql"
	"strings"
	"time"

	"ampmanager/internal/database"
	"ampmanager/internal/model"

	"github.com/google/uuid"
)

type ModelMetadataRepository struct{}

func NewModelMetadataRepository() *ModelMetadataRepository {
	return &ModelMetadataRepository{}
}

func (r *ModelMetadataRepository) Create(meta *model.ModelMetadata) error {
	db := database.GetDB()
	meta.ID = uuid.New().String()
	now := time.Now().UTC()
	meta.CreatedAt = now
	meta.UpdatedAt = now

	_, err := db.Exec(
		`INSERT INTO model_metadata (id, model_pattern, display_name, context_length, max_completion_tokens, provider, is_builtin, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?)`,
		meta.ID, meta.ModelPattern, meta.DisplayName, meta.ContextLength, meta.MaxCompletionTokens,
		meta.Provider, meta.CreatedAt, meta.UpdatedAt,
	)
	return err
}

func (r *ModelMetadataRepository) GetByID(id string) (*model.ModelMetadata, error) {
	db := database.GetDB()
	meta := &model.ModelMetadata{}

	err := db.QueryRow(
		`SELECT id, model_pattern, display_name, context_length, max_completion_tokens, provider, created_at, updated_at
		 FROM model_metadata WHERE id = ?`,
		id,
	).Scan(
		&meta.ID, &meta.ModelPattern, &meta.DisplayName, &meta.ContextLength, &meta.MaxCompletionTokens,
		&meta.Provider, &meta.CreatedAt, &meta.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return meta, nil
}

func (r *ModelMetadataRepository) GetByPattern(pattern string) (*model.ModelMetadata, error) {
	db := database.GetDB()
	meta := &model.ModelMetadata{}

	err := db.QueryRow(
		`SELECT id, model_pattern, display_name, context_length, max_completion_tokens, provider, created_at, updated_at
		 FROM model_metadata WHERE model_pattern = ?`,
		pattern,
	).Scan(
		&meta.ID, &meta.ModelPattern, &meta.DisplayName, &meta.ContextLength, &meta.MaxCompletionTokens,
		&meta.Provider, &meta.CreatedAt, &meta.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return meta, nil
}

func (r *ModelMetadataRepository) List() ([]*model.ModelMetadata, error) {
	db := database.GetDB()
	rows, err := db.Query(
		`SELECT id, model_pattern, display_name, context_length, max_completion_tokens, provider, created_at, updated_at
		 FROM model_metadata ORDER BY provider, model_pattern`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.ModelMetadata
	for rows.Next() {
		meta := &model.ModelMetadata{}
		err := rows.Scan(
			&meta.ID, &meta.ModelPattern, &meta.DisplayName, &meta.ContextLength, &meta.MaxCompletionTokens,
			&meta.Provider, &meta.CreatedAt, &meta.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, meta)
	}
	return list, rows.Err()
}

func (r *ModelMetadataRepository) Update(meta *model.ModelMetadata) error {
	db := database.GetDB()
	meta.UpdatedAt = time.Now().UTC()

	_, err := db.Exec(
		`UPDATE model_metadata SET model_pattern = ?, display_name = ?, context_length = ?, max_completion_tokens = ?, provider = ?, updated_at = ?
		 WHERE id = ?`,
		meta.ModelPattern, meta.DisplayName, meta.ContextLength, meta.MaxCompletionTokens, meta.Provider, meta.UpdatedAt,
		meta.ID,
	)
	return err
}

func (r *ModelMetadataRepository) Delete(id string) error {
	db := database.GetDB()
	_, err := db.Exec(`DELETE FROM model_metadata WHERE id = ?`, id)
	return err
}

func (r *ModelMetadataRepository) FindMatchingModel(modelName string) (*model.ModelMetadata, error) {
	if modelName == "" {
		return nil, nil
	}

	db := database.GetDB()
	rows, err := db.Query(
		`SELECT id, model_pattern, display_name, context_length, max_completion_tokens, provider, created_at, updated_at
		 FROM model_metadata ORDER BY LENGTH(model_pattern) DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		meta := &model.ModelMetadata{}
		err := rows.Scan(
			&meta.ID, &meta.ModelPattern, &meta.DisplayName, &meta.ContextLength, &meta.MaxCompletionTokens,
			&meta.Provider, &meta.CreatedAt, &meta.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if modelName == meta.ModelPattern || strings.HasPrefix(modelName, meta.ModelPattern) {
			return meta, nil
		}
	}

	return nil, rows.Err()
}
