package repository

import (
	"database/sql"
	"time"

	"ampmanager/internal/database"
	"ampmanager/internal/model"

	"github.com/google/uuid"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) Create(user *model.User) error {
	db := database.GetDB()
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := db.Exec(
		`INSERT INTO users (id, username, password_hash, is_admin, created_at, updated_at) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.PasswordHash, user.IsAdmin, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	db := database.GetDB()
	user := &model.User{}
	err := db.QueryRow(
		`SELECT id, username, password_hash, is_admin, created_at, updated_at FROM users WHERE username = ?`,
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func (r *UserRepository) ExistsByUsername(username string) (bool, error) {
	db := database.GetDB()
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE username = ?`, username).Scan(&count)
	return count > 0, err
}

func (r *UserRepository) GetByID(id string) (*model.User, error) {
	db := database.GetDB()
	user := &model.User{}
	err := db.QueryRow(
		`SELECT id, username, password_hash, is_admin, created_at, updated_at FROM users WHERE id = ?`,
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}
