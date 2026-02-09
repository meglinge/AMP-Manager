package repository

import (
	"database/sql"
	"errors"
	"time"

	"ampmanager/internal/database"
	"ampmanager/internal/model"

	"github.com/google/uuid"
)

var ErrUserNotFound = errors.New("用户不存在")

type UserRepositoryInterface interface {
	Create(user *model.User) error
	GetByUsername(username string) (*model.User, error)
	ExistsByUsername(username string) (bool, error)
	GetByID(id string) (*model.User, error)
	List() ([]*model.User, error)
	UpdatePassword(id string, passwordHash string) error
	UpdateUsername(id string, username string) error
	SetAdmin(id string, isAdmin bool) error
	SetGroups(id string, groupIDs []string) error
	GetGroupIDs(userID string) ([]string, error)
	Delete(id string) error
	GetBalance(userID string) (int64, error)
	DeductBalance(userID string, amountMicros int64) error
	TopUpBalance(userID string, amountMicros int64) error
}

var _ UserRepositoryInterface = (*UserRepository)(nil)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) Create(user *model.User) error {
	db := database.GetDB()
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = time.Now().UTC()

	_, err := db.Exec(
		`INSERT INTO users (id, username, password_hash, is_admin, balance_micros, created_at, updated_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.PasswordHash, user.IsAdmin, user.BalanceMicros, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	db := database.GetDB()
	user := &model.User{}
	err := db.QueryRow(
		`SELECT id, username, password_hash, is_admin, balance_micros, created_at, updated_at FROM users WHERE username = ?`,
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &user.BalanceMicros, &user.CreatedAt, &user.UpdatedAt)
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
		`SELECT id, username, password_hash, is_admin, balance_micros, created_at, updated_at FROM users WHERE id = ?`,
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &user.BalanceMicros, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func (r *UserRepository) List() ([]*model.User, error) {
	db := database.GetDB()
	rows, err := db.Query(
		`SELECT id, username, password_hash, is_admin, balance_micros, created_at, updated_at FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		user := &model.User{}
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &user.BalanceMicros, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *UserRepository) UpdatePassword(id string, passwordHash string) error {
	db := database.GetDB()
	result, err := db.Exec(
		`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		passwordHash, time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) UpdateUsername(id string, username string) error {
	db := database.GetDB()
	result, err := db.Exec(
		`UPDATE users SET username = ?, updated_at = ? WHERE id = ?`,
		username, time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) SetAdmin(id string, isAdmin bool) error {
	db := database.GetDB()
	_, err := db.Exec(
		`UPDATE users SET is_admin = ?, updated_at = ? WHERE id = ?`,
		isAdmin, time.Now().UTC(), id,
	)
	return err
}

func (r *UserRepository) SetGroups(id string, groupIDs []string) error {
	db := database.GetDB()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM user_groups WHERE user_id = ?`, id)
	if err != nil {
		return err
	}

	for _, gid := range groupIDs {
		if gid == "" {
			continue
		}
		_, err = tx.Exec(`INSERT INTO user_groups (user_id, group_id) VALUES (?, ?)`, id, gid)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *UserRepository) GetGroupIDs(userID string) ([]string, error) {
	db := database.GetDB()
	rows, err := db.Query(`SELECT group_id FROM user_groups WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *UserRepository) Delete(id string) error {
	db := database.GetDB()
	_, err := db.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

func (r *UserRepository) GetBalance(userID string) (int64, error) {
	db := database.GetDB()
	var balance int64
	err := db.QueryRow(`SELECT balance_micros FROM users WHERE id = ?`, userID).Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, ErrUserNotFound
	}
	return balance, err
}

func (r *UserRepository) DeductBalance(userID string, amountMicros int64) error {
	db := database.GetDB()
	result, err := db.Exec(
		`UPDATE users SET balance_micros = balance_micros - ?, updated_at = ? WHERE id = ?`,
		amountMicros, time.Now().UTC(), userID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) TopUpBalance(userID string, amountMicros int64) error {
	db := database.GetDB()
	result, err := db.Exec(
		`UPDATE users SET balance_micros = balance_micros + ?, updated_at = ? WHERE id = ?`,
		amountMicros, time.Now().UTC(), userID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}
