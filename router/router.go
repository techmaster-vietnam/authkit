package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/config"
	"github.com/techmaster-vietnam/authkit/handlers"
	"github.com/techmaster-vietnam/authkit/middleware"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/authkit/service"
	"gorm.io/gorm"
)

// SetupRoutes sets up all routes
func SetupRoutes(app *fiber.App, db *gorm.DB, cfg *config.Config) error {
	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	ruleRepo := repository.NewRuleRepository(db, cfg.ServiceName)

	// Initialize services
	authService := service.NewAuthService(userRepo, roleRepo, cfg)
	roleService := service.NewRoleService(roleRepo)
	ruleService := service.NewRuleService(ruleRepo, roleRepo)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg, userRepo)
	authzMiddleware := middleware.NewAuthorizationMiddleware(ruleRepo, roleRepo, userRepo)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	roleHandler := handlers.NewRoleHandler(roleService)
	ruleHandler := handlers.NewRuleHandler(ruleService, authzMiddleware)

	// API routes
	api := app.Group("/api")

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/login", authHandler.Login)
	auth.Post("/register", authHandler.Register)
	auth.Post("/logout", authHandler.Logout)

	// Protected auth routes
	authProtected := auth.Group("", authMiddleware.RequireAuth(), authzMiddleware.Authorize())
	authProtected.Get("/profile", authHandler.GetProfile)
	authProtected.Put("/profile", authHandler.UpdateProfile)
	authProtected.Delete("/profile", authHandler.DeleteProfile)
	authProtected.Post("/change-password", authHandler.ChangePassword)

	// Role routes (protected)
	roles := api.Group("/roles", authMiddleware.RequireAuth(), authzMiddleware.Authorize())
	roles.Get("/", roleHandler.ListRoles)
	roles.Post("/", roleHandler.AddRole)
	roles.Delete("/:id", roleHandler.RemoveRole)
	roles.Get("/:role_name/users", roleHandler.ListUsersHasRole)

	// User role routes (protected)
	users := api.Group("/users", authMiddleware.RequireAuth(), authzMiddleware.Authorize())
	users.Get("/:user_id/roles", roleHandler.ListRolesOfUser)
	users.Post("/:user_id/roles/:role_id", roleHandler.AddRoleToUser)
	users.Delete("/:user_id/roles/:role_id", roleHandler.RemoveRoleFromUser)
	users.Get("/:user_id/roles/:role_name/check", roleHandler.CheckUserHasRole)

	// Rule routes (protected)
	rules := api.Group("/rules", authMiddleware.RequireAuth(), authzMiddleware.Authorize())
	rules.Get("/", ruleHandler.ListRules)
	rules.Post("/", ruleHandler.AddRule)
	rules.Put("/:id", ruleHandler.UpdateRule)
	rules.Delete("/:id", ruleHandler.RemoveRule)

	return nil
}

