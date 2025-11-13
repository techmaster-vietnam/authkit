package authkit

import (
	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/config"
	"github.com/techmaster-vietnam/authkit/database"
	"github.com/techmaster-vietnam/authkit/router"
	"gorm.io/gorm"
)

// Config là alias cho config.Config để tránh conflict với package config khác
type Config = config.Config

// LoadConfig loads configuration from environment variables
// Đây là wrapper function để tránh conflict với package config của ứng dụng chính
func LoadConfig() *Config {
	return config.LoadConfig()
}

// Migrate runs database migrations for AuthKit models
func Migrate(db *gorm.DB) error {
	return database.Migrate(db)
}

// SetupRoutes sets up all routes for AuthKit
func SetupRoutes(app *fiber.App, db *gorm.DB, cfg *Config) error {
	return router.SetupRoutes(app, db, cfg)
}
