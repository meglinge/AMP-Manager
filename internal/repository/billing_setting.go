package repository

import (
	"database/sql"
	"time"

	"ampmanager/internal/database"
	"ampmanager/internal/model"
)

type BillingSettingRepositoryInterface interface {
	GetByUserID(userID string) (*model.UserBillingSetting, error)
	Upsert(setting *model.UserBillingSetting) error
}

var _ BillingSettingRepositoryInterface = (*BillingSettingRepository)(nil)

type BillingSettingRepository struct{}

func NewBillingSettingRepository() *BillingSettingRepository {
	return &BillingSettingRepository{}
}

func (r *BillingSettingRepository) GetByUserID(userID string) (*model.UserBillingSetting, error) {
	db := database.GetDB()
	s := &model.UserBillingSetting{}
	err := db.QueryRow(
		`SELECT user_id, primary_source, secondary_source, created_at, updated_at FROM user_billing_settings WHERE user_id = ?`,
		userID,
	).Scan(&s.UserID, &s.PrimarySource, &s.SecondarySource, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return &model.UserBillingSetting{
			UserID:          userID,
			PrimarySource:   model.BillingSourceSubscription,
			SecondarySource: model.BillingSourceBalance,
		}, nil
	}
	return s, err
}

func (r *BillingSettingRepository) Upsert(setting *model.UserBillingSetting) error {
	db := database.GetDB()
	now := time.Now().UTC()
	setting.UpdatedAt = now

	_, err := db.Exec(
		`INSERT INTO user_billing_settings (user_id, primary_source, secondary_source, created_at, updated_at) 
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET primary_source = excluded.primary_source, secondary_source = excluded.secondary_source, updated_at = excluded.updated_at`,
		setting.UserID, setting.PrimarySource, setting.SecondarySource, now, now,
	)
	return err
}
