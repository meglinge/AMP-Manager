package service

import (
	"errors"
	"time"

	"ampmanager/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken    = errors.New("无效的 Token")
	ErrExpiredToken    = errors.New("Token 已过期")
	ErrInvalidIssuer   = errors.New("无效的 Token 签发者")
	ErrInvalidAudience = errors.New("无效的 Token 受众")
)

type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type JWTService struct{}

func NewJWTService() *JWTService {
	return &JWTService{}
}

func (s *JWTService) GenerateToken(userID, username string) (string, error) {
	cfg := config.Get()

	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.JWTIssuer,
			Audience:  jwt.ClaimStrings{cfg.JWTAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	cfg := config.Get()

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(cfg.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.Issuer != cfg.JWTIssuer {
		return nil, ErrInvalidIssuer
	}

	validAudience := false
	for _, aud := range claims.Audience {
		if aud == cfg.JWTAudience {
			validAudience = true
			break
		}
	}
	if !validAudience {
		return nil, ErrInvalidAudience
	}

	return claims, nil
}
