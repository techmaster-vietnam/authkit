package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds application configuration
type Config struct {
	JWT         JWTConfig
	Server      ServerConfig
	ServiceName string // Service name for microservice architecture (max 20 chars, empty = single app mode)
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	jwtExpirationHours, _ := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24"))
	readTimeout, _ := strconv.Atoi(getEnv("READ_TIMEOUT_SECONDS", "10"))
	writeTimeout, _ := strconv.Atoi(getEnv("WRITE_TIMEOUT_SECONDS", "10"))

	serviceName := getEnv("SERVICE_NAME", "")
	// Truncate to max 20 characters if longer
	if len(serviceName) > 20 {
		serviceName = serviceName[:20]
	}

	return &Config{
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			Expiration: time.Duration(jwtExpirationHours) * time.Hour,
		},
		Server: ServerConfig{
			Port:         getEnv("PORT", "3000"),
			ReadTimeout:  time.Duration(readTimeout) * time.Second,
			WriteTimeout: time.Duration(writeTimeout) * time.Second,
		},
		ServiceName: serviceName,
	}
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
