package router

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/config"
	"gorm.io/gorm"
)

// SetupRoutes sets up all routes
// DEPRECATED: This function is deprecated and no longer supported.
// Use AuthKit with generic API instead: authkit.New[TUser, TRole](app, db).Initialize()
// This function is kept for backward compatibility but will return an error.
func SetupRoutes(app *fiber.App, db *gorm.DB, cfg *config.Config) error {
	// This function is deprecated. Use AuthKit generic API instead.
	// Example:
	//   ak, err := authkit.New[*CustomUser, *authkit.BaseRole](app, db).
	//       WithConfig(cfg).
	//       WithUserModel(&CustomUser{}).
	//       WithRoleModel(&authkit.BaseRole{}).
	//       Initialize()
	return fmt.Errorf("SetupRoutes is deprecated. Use authkit.New[TUser, TRole](app, db).Initialize() instead")

}

