package config

import (
	"os"
)

type Config struct {
	AdminUsername string
	AdminPassword string
	ServerPort    string
	JWTSecret     string
}

var cfg *Config

func Load() *Config {
	cfg = &Config{
		AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		ServerPort:    getEnv("SERVER_PORT", "16823"),
		JWTSecret:     getEnv("JWT_SECRET", "amp-manager-default-secret-change-in-production"),
	}
	return cfg
}

func Get() *Config {
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
