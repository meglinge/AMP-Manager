package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AdminUsername string
	AdminPassword string
	ServerPort    string
	JWTSecret     string
	JWTIssuer     string
	JWTAudience   string

	// CORS 配置
	CORSAllowedOrigins string

	// 速率限制配置
	RateLimitAuthRPS  float64
	RateLimitProxyRPS float64

	// 数据加密密钥 (32 bytes for AES-256)
	DataEncryptionKey string
}

var cfg *Config

var insecureDefaults = map[string]string{
	"ADMIN_PASSWORD": "admin123",
	"JWT_SECRET":     "amp-manager-default-secret-change-in-production",
}

func Load() *Config {
	cfg = &Config{
		AdminUsername:      getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:      getEnv("ADMIN_PASSWORD", "admin123"),
		ServerPort:         getEnv("SERVER_PORT", "16823"),
		JWTSecret:          getEnv("JWT_SECRET", "amp-manager-default-secret-change-in-production"),
		JWTIssuer:          getEnv("JWT_ISSUER", "ampmanager"),
		JWTAudience:        getEnv("JWT_AUDIENCE", "ampmanager-users"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),
		RateLimitAuthRPS:   getEnvFloat("RATE_LIMIT_AUTH_RPS", 5),
		RateLimitProxyRPS:  getEnvFloat("RATE_LIMIT_PROXY_RPS", 100),
		DataEncryptionKey:  getEnv("DATA_ENCRYPTION_KEY", ""),
	}
	return cfg
}

func ValidateSecurityConfig(cfg *Config) error {
	if os.Getenv("ALLOW_INSECURE_DEFAULTS") == "true" {
		return nil
	}

	var issues []string

	if cfg.AdminPassword == insecureDefaults["ADMIN_PASSWORD"] {
		issues = append(issues, "ADMIN_PASSWORD is using insecure default 'admin123'")
	}

	if cfg.JWTSecret == insecureDefaults["JWT_SECRET"] {
		issues = append(issues, "JWT_SECRET is using insecure default value")
	}

	if len(cfg.JWTSecret) < 32 {
		issues = append(issues, "JWT_SECRET should be at least 32 characters")
	}

	if cfg.DataEncryptionKey != "" && len(cfg.DataEncryptionKey) != 32 {
		issues = append(issues, "DATA_ENCRYPTION_KEY must be exactly 32 characters for AES-256")
	}

	if len(issues) > 0 {
		return fmt.Errorf("security configuration errors:\n  - %s\n\nSet ALLOW_INSECURE_DEFAULTS=true to bypass (NOT recommended for production)", strings.Join(issues, "\n  - "))
	}

	return nil
}

func Get() *Config {
	return cfg
}

func (c *Config) GetEncryptionKey() []byte {
	if c.DataEncryptionKey == "" {
		return nil
	}
	return []byte(c.DataEncryptionKey)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}
