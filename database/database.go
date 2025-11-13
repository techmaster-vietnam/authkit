package database

import (
	"github.com/techmaster-vietnam/authkit/models"
	"gorm.io/gorm"
)

// Migrate runs database migrations for AuthKit models
// This function creates the necessary tables (User, Role, Rule, Blog) for authentication and authorization
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Rule{},
		&models.Blog{},
	)
}
