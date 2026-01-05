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

func (r *UserRepository) List() ([]*model.User, error) {
	db := database.GetDB()
	rows, err := db.Query(
		`SELECT id, username, password_hash, is_admin, created_at, updated_at FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		user := &model.User{}
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *UserRepository) UpdatePassword(id string, passwordHash string) error {
	db := database.GetDB()
	_, err := db.Exec(
		`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		passwordHash, time.Now(), id,
	)
	return err
}

func (r *UserRepository) UpdateUsername(id string, username string) error {
	db := database.GetDB()
	_, err := db.Exec(
		`UPDATE users SET username = ?, updated_at = ? WHERE id = ?`,
		username, time.Now(), id,
	)
	return err
}

func (r *UserRepository) SetAdmin(id string, isAdmin bool) error {
	db := database.GetDB()
	_, err := db.Exec(
		`UPDATE users SET is_admin = ?, updated_at = ? WHERE id = ?`,
		isAdmin, time.Now(), id,
	)
	return err
}

func (r *UserRepository) Delete(id string) error {
	db := database.GetDB()
	_, err := db.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}
