package service

import (
	"errors"
	"fmt"
	"log"

	"ampmanager/internal/config"
	"ampmanager/internal/model"
	"ampmanager/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUsernameExists     = errors.New("用户名已存在")
	ErrInvalidCredentials = errors.New("用户名或密码错误")
)

type UserService struct {
	repo repository.UserRepositoryInterface
}

// NewUserServiceWithRepo 使用指定的仓库实现创建 UserService（用于依赖注入和测试）
func NewUserServiceWithRepo(repo repository.UserRepositoryInterface) *UserService {
	return &UserService{
		repo: repo,
	}
}

// NewUserService 创建使用默认仓库的 UserService（便利方法）
func NewUserService() *UserService {
	return NewUserServiceWithRepo(repository.NewUserRepository())
}

func (s *UserService) Register(req *model.RegisterRequest) (*model.User, error) {
	exists, err := s.repo.ExistsByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		IsAdmin:      false,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Login(req *model.LoginRequest) (*model.User, string, error) {
	user, err := s.repo.GetByUsername(req.Username)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	jwtService := NewJWTService()
	token, err := jwtService.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *UserService) EnsureAdmin() error {
	cfg := config.Get()

	exists, err := s.repo.ExistsByUsername(cfg.AdminUsername)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := &model.User{
		Username:     cfg.AdminUsername,
		PasswordHash: string(hashedPassword),
		IsAdmin:      true,
	}

	if err := s.repo.Create(admin); err != nil {
		return err
	}

	log.Printf("管理员账户已创建: %s", cfg.AdminUsername)
	return nil
}

func (s *UserService) ListUsers() ([]*model.UserInfo, error) {
	users, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	groupRepo := repository.NewGroupRepository()
	result := make([]*model.UserInfo, len(users))
	for i, u := range users {
		info := &model.UserInfo{
			ID:            u.ID,
			Username:      u.Username,
			IsAdmin:       u.IsAdmin,
			BalanceMicros: u.BalanceMicros,
			BalanceUsd:    fmt.Sprintf("%.6f", float64(u.BalanceMicros)/1e6),
			GroupIDs:      []string{},
			GroupNames:    []string{},
			CreatedAt:     u.CreatedAt,
			UpdatedAt:     u.UpdatedAt,
		}
		gids, err := s.repo.GetGroupIDs(u.ID)
		if err == nil && len(gids) > 0 {
			info.GroupIDs = gids
			for _, gid := range gids {
				group, err := groupRepo.GetByID(gid)
				if err == nil && group != nil {
					info.GroupNames = append(info.GroupNames, group.Name)
				}
			}
		}
		result[i] = info
	}
	return result, nil
}

func (s *UserService) ChangePassword(userID string, oldPassword, newPassword string) error {
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("用户不存在")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return errors.New("旧密码错误")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.repo.UpdatePassword(userID, string(hashedPassword))
}

func (s *UserService) ChangeUsername(userID string, newUsername string) error {
	exists, err := s.repo.ExistsByUsername(newUsername)
	if err != nil {
		return err
	}
	if exists {
		return ErrUsernameExists
	}
	return s.repo.UpdateUsername(userID, newUsername)
}

func (s *UserService) SetAdmin(userID string, isAdmin bool) error {
	return s.repo.SetAdmin(userID, isAdmin)
}

func (s *UserService) SetGroups(userID string, groupIDs []string) error {
	return s.repo.SetGroups(userID, groupIDs)
}

func (s *UserService) DeleteUser(userID string) error {
	return s.repo.Delete(userID)
}

func (s *UserService) ResetPassword(userID string, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(userID, string(hashedPassword))
}

func (s *UserService) GetBalance(userID string) (int64, error) {
	return s.repo.GetBalance(userID)
}

func (s *UserService) TopUp(userID string, amountMicros int64) error {
	return s.repo.TopUpBalance(userID, amountMicros)
}

func (s *UserService) GetTotalBalanceAndUserCount() (int64, int64, error) {
	return s.repo.GetTotalBalanceAndUserCount()
}
