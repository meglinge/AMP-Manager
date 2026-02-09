package repository

import (
	"database/sql"
	"time"

	"ampmanager/internal/database"
	"ampmanager/internal/model"

	"github.com/google/uuid"
)

type GroupRepositoryInterface interface {
	Create(group *model.Group) error
	GetByID(id string) (*model.Group, error)
	GetByName(name string) (*model.Group, error)
	List() ([]*model.Group, error)
	Update(group *model.Group) error
	Delete(id string) error
	CountUsers(groupID string) (int, error)
	CountChannels(groupID string) (int, error)
	GetMinRateMultiplierByUserID(userID string) (float64, []string, error)
}

var _ GroupRepositoryInterface = (*GroupRepository)(nil)

type GroupRepository struct{}

func NewGroupRepository() *GroupRepository {
	return &GroupRepository{}
}

func (r *GroupRepository) Create(group *model.Group) error {
	db := database.GetDB()
	group.ID = uuid.New().String()
	now := time.Now().UTC()
	group.CreatedAt = now
	group.UpdatedAt = now
	if group.RateMultiplier == 0 {
		group.RateMultiplier = 1.0
	}

	_, err := db.Exec(
		`INSERT INTO groups (id, name, description, rate_multiplier, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		group.ID, group.Name, group.Description, group.RateMultiplier, group.CreatedAt, group.UpdatedAt,
	)
	return err
}

func (r *GroupRepository) GetByID(id string) (*model.Group, error) {
	db := database.GetDB()
	group := &model.Group{}
	err := db.QueryRow(
		`SELECT id, name, description, rate_multiplier, created_at, updated_at FROM groups WHERE id = ?`, id,
	).Scan(&group.ID, &group.Name, &group.Description, &group.RateMultiplier, &group.CreatedAt, &group.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return group, err
}

func (r *GroupRepository) GetByName(name string) (*model.Group, error) {
	db := database.GetDB()
	group := &model.Group{}
	err := db.QueryRow(
		`SELECT id, name, description, rate_multiplier, created_at, updated_at FROM groups WHERE name = ?`, name,
	).Scan(&group.ID, &group.Name, &group.Description, &group.RateMultiplier, &group.CreatedAt, &group.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return group, err
}

func (r *GroupRepository) List() ([]*model.Group, error) {
	db := database.GetDB()
	rows, err := db.Query(
		`SELECT id, name, description, rate_multiplier, created_at, updated_at FROM groups ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*model.Group
	for rows.Next() {
		group := &model.Group{}
		if err := rows.Scan(&group.ID, &group.Name, &group.Description, &group.RateMultiplier, &group.CreatedAt, &group.UpdatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func (r *GroupRepository) Update(group *model.Group) error {
	db := database.GetDB()
	group.UpdatedAt = time.Now().UTC()
	_, err := db.Exec(
		`UPDATE groups SET name = ?, description = ?, rate_multiplier = ?, updated_at = ? WHERE id = ?`,
		group.Name, group.Description, group.RateMultiplier, group.UpdatedAt, group.ID,
	)
	return err
}

func (r *GroupRepository) Delete(id string) error {
	db := database.GetDB()
	_, _ = db.Exec(`DELETE FROM user_groups WHERE group_id = ?`, id)
	_, _ = db.Exec(`DELETE FROM channel_groups WHERE group_id = ?`, id)
	_, err := db.Exec(`DELETE FROM groups WHERE id = ?`, id)
	return err
}

func (r *GroupRepository) CountUsers(groupID string) (int, error) {
	db := database.GetDB()
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM user_groups WHERE group_id = ?`, groupID).Scan(&count)
	return count, err
}

func (r *GroupRepository) CountChannels(groupID string) (int, error) {
	db := database.GetDB()
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM channel_groups WHERE group_id = ?`, groupID).Scan(&count)
	return count, err
}

func (r *GroupRepository) GetMinRateMultiplierByUserID(userID string) (float64, []string, error) {
	db := database.GetDB()
	rows, err := db.Query(`
		SELECT g.id, g.rate_multiplier
		FROM groups g
		INNER JOIN user_groups ug ON g.id = ug.group_id
		WHERE ug.user_id = ?
	`, userID)
	if err != nil {
		return 1.0, nil, err
	}
	defer rows.Close()

	var groupIDs []string
	minMultiplier := -1.0
	for rows.Next() {
		var gid string
		var mult float64
		if err := rows.Scan(&gid, &mult); err != nil {
			return 1.0, nil, err
		}
		groupIDs = append(groupIDs, gid)
		if minMultiplier < 0 || mult < minMultiplier {
			minMultiplier = mult
		}
	}
	if err := rows.Err(); err != nil {
		return 1.0, nil, err
	}

	if len(groupIDs) == 0 {
		return 1.0, nil, nil
	}
	return minMultiplier, groupIDs, nil
}
