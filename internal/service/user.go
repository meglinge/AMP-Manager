package service

import (
	"errors"
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
	repo *repository.UserRepository
}

func NewUserService() *UserService {
	return &UserService{
		repo: repository.NewUserRepository(),
	}
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
