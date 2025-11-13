package authkit

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/techmaster-vietnam/authkit/config"
	"github.com/techmaster-vietnam/authkit/handlers"
	"github.com/techmaster-vietnam/authkit/middleware"
	"github.com/techmaster-vietnam/authkit/models"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/authkit/router"
	"github.com/techmaster-vietnam/authkit/service"
	"gorm.io/gorm"
)

// Config là alias cho config.Config để tránh conflict với package config khác
type Config = config.Config

// Models - Export các models
type (
	User = models.User
	Role = models.Role
	Rule = models.Rule
)

// RuleType constants
const (
	RuleTypePublic        = models.RuleTypePublic
	RuleTypeAllow         = models.RuleTypeAllow
	RuleTypeForbid        = models.RuleTypeForbid
	RuleTypeAuth          = models.RuleTypeAuth
	RuleTypeAuthenticated = models.RuleTypeAuth
)

// Repositories - Export repository types và constructors
type (
	UserRepository = repository.UserRepository
	RoleRepository = repository.RoleRepository
	RuleRepository = repository.RuleRepository
)

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return repository.NewUserRepository(db)
}

// NewRoleRepository creates a new role repository
func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return repository.NewRoleRepository(db)
}

// NewRuleRepository creates a new rule repository
func NewRuleRepository(db *gorm.DB) *RuleRepository {
	return repository.NewRuleRepository(db)
}

// Services - Export service types và constructors
type (
	AuthService = service.AuthService
	RoleService = service.RoleService
	RuleService = service.RuleService
)

// Service request/response types
type (
	LoginRequest      = service.LoginRequest
	LoginResponse     = service.LoginResponse
	RegisterRequest   = service.RegisterRequest
	AddRoleRequest    = service.AddRoleRequest
	AddRuleRequest    = service.AddRuleRequest
	UpdateRuleRequest = service.UpdateRuleRequest
)

// NewAuthService creates a new auth service
func NewAuthService(userRepo *UserRepository, cfg *Config) *AuthService {
	return service.NewAuthService(userRepo, cfg)
}

// NewRoleService creates a new role service
func NewRoleService(roleRepo *RoleRepository) *RoleService {
	return service.NewRoleService(roleRepo)
}

// NewRuleService creates a new rule service
func NewRuleService(ruleRepo *RuleRepository) *RuleService {
	return service.NewRuleService(ruleRepo)
}

// Middleware - Export middleware types và constructors
type (
	AuthMiddleware          = middleware.AuthMiddleware
	AuthorizationMiddleware = middleware.AuthorizationMiddleware
)

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(cfg *Config, userRepo *UserRepository) *AuthMiddleware {
	return middleware.NewAuthMiddleware(cfg, userRepo)
}

// NewAuthorizationMiddleware creates a new authorization middleware
func NewAuthorizationMiddleware(
	ruleRepo *RuleRepository,
	roleRepo *RoleRepository,
	userRepo *UserRepository,
) *AuthorizationMiddleware {
	return middleware.NewAuthorizationMiddleware(ruleRepo, roleRepo, userRepo)
}

// Middleware helper functions
func GetUserFromContext(c *fiber.Ctx) (*User, bool) {
	return middleware.GetUserFromContext(c)
}

func GetUserIDFromContext(c *fiber.Ctx) (uuid.UUID, bool) {
	return middleware.GetUserIDFromContext(c)
}

// Handlers - Export handler types và constructors
type (
	AuthHandler = handlers.AuthHandler
	RoleHandler = handlers.RoleHandler
	RuleHandler = handlers.RuleHandler
)

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *AuthService) *AuthHandler {
	return handlers.NewAuthHandler(authService)
}

// NewRoleHandler creates a new role handler
func NewRoleHandler(roleService *RoleService) *RoleHandler {
	return handlers.NewRoleHandler(roleService)
}

// NewRuleHandler creates a new rule handler
func NewRuleHandler(ruleService *RuleService, authzMiddleware *AuthorizationMiddleware) *RuleHandler {
	return handlers.NewRuleHandler(ruleService, authzMiddleware)
}

// LoadConfig loads configuration from environment variables
// Đây là wrapper function để tránh conflict với package config của ứng dụng chính
func LoadConfig() *Config {
	return config.LoadConfig()
}

// SetupRoutes sets up all routes for AuthKit
func SetupRoutes(app *fiber.App, db *gorm.DB, cfg *Config) error {
	return router.SetupRoutes(app, db, cfg)
}
