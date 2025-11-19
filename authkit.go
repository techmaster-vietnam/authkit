package authkit

import (
	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/config"
	"github.com/techmaster-vietnam/authkit/core"
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
	User    = models.User
	BaseUser = models.BaseUser
	Role    = models.Role
	BaseRole = models.BaseRole
	Rule    = models.Rule
)

// Interfaces - Export interfaces
type (
	UserInterface = core.UserInterface
	RoleInterface = core.RoleInterface
)

// AuthKit là main struct chứa tất cả dependencies
// TUser phải implement UserInterface, TRole phải implement RoleInterface
type AuthKit[TUser UserInterface, TRole RoleInterface] struct {
	DB     *gorm.DB
	Config *Config

	// Repositories
	UserRepo *BaseUserRepository[TUser]
	RoleRepo *BaseRoleRepository[TRole]
	RuleRepo *RuleRepository

	// Services
	AuthService *BaseAuthService[TUser, TRole]
	RoleService *BaseRoleService[TRole]
	RuleService *RuleService

	// Middleware
	AuthMiddleware          *BaseAuthMiddleware[TUser]
	AuthorizationMiddleware *BaseAuthorizationMiddleware[TUser, TRole]

	// Handlers
	AuthHandler *BaseAuthHandler[TUser, TRole]
	RoleHandler *BaseRoleHandler[TRole]
	RuleHandler *BaseRuleHandler[TUser, TRole]

	// Route registry
	RouteRegistry *router.RouteRegistry
}

// AuthKitBuilder là builder để tạo AuthKit
type AuthKitBuilder[TUser UserInterface, TRole RoleInterface] struct {
	app       *fiber.App
	db        *gorm.DB
	config    *Config
	userModel TUser
	roleModel TRole
}

// New tạo mới AuthKitBuilder với generics
func New[TUser UserInterface, TRole RoleInterface](
	app *fiber.App,
	db *gorm.DB,
) *AuthKitBuilder[TUser, TRole] {
	return &AuthKitBuilder[TUser, TRole]{
		app: app,
		db:  db,
	}
}

// WithConfig set config cho builder
func (b *AuthKitBuilder[TUser, TRole]) WithConfig(cfg *Config) *AuthKitBuilder[TUser, TRole] {
	b.config = cfg
	return b
}

// WithUserModel set user model cho builder (để auto migrate)
func (b *AuthKitBuilder[TUser, TRole]) WithUserModel(userModel TUser) *AuthKitBuilder[TUser, TRole] {
	b.userModel = userModel
	return b
}

// WithRoleModel set role model cho builder (để auto migrate)
func (b *AuthKitBuilder[TUser, TRole]) WithRoleModel(roleModel TRole) *AuthKitBuilder[TUser, TRole] {
	b.roleModel = roleModel
	return b
}

// Initialize khởi tạo AuthKit với tất cả dependencies
func (b *AuthKitBuilder[TUser, TRole]) Initialize() (*AuthKit[TUser, TRole], error) {
	// Load config nếu chưa có
	if b.config == nil {
		b.config = LoadConfig()
	}

	// Auto migrate với custom models
	if err := b.db.AutoMigrate(&b.userModel, &b.roleModel, &models.Rule{}); err != nil {
		return nil, err
	}

	// Initialize repositories
	userRepo := repository.NewBaseUserRepository[TUser](b.db)
	roleRepo := repository.NewBaseRoleRepository[TRole](b.db)
	ruleRepo := repository.NewRuleRepository(b.db)

	// Initialize services
	authService := service.NewBaseAuthService[TUser, TRole](userRepo, roleRepo, b.config)
	roleService := service.NewBaseRoleService[TRole](roleRepo)
	ruleService := service.NewRuleService(ruleRepo, repository.NewRoleRepository(b.db))

	// Initialize middleware với generic types
	authMiddleware := middleware.NewBaseAuthMiddleware[TUser](b.config, userRepo)
	authzMiddleware := middleware.NewBaseAuthorizationMiddleware[TUser, TRole](ruleRepo, roleRepo, userRepo)

	// Initialize handlers với generic types
	authHandler := handlers.NewBaseAuthHandler[TUser, TRole](authService)
	roleHandler := handlers.NewBaseRoleHandler[TRole](roleService)
	ruleHandler := handlers.NewBaseRuleHandler[TUser, TRole](ruleService, authzMiddleware)

	// Create route registry
	routeRegistry := router.NewRouteRegistry()

	return &AuthKit[TUser, TRole]{
		DB:                     b.db,
		Config:                 b.config,
		UserRepo:               userRepo,
		RoleRepo:               roleRepo,
		RuleRepo:               ruleRepo,
		AuthService:            authService,
		RoleService:            roleService,
		RuleService:            ruleService,
		AuthMiddleware:         authMiddleware,
		AuthorizationMiddleware: authzMiddleware,
		AuthHandler:            authHandler,
		RoleHandler:            roleHandler,
		RuleHandler:            ruleHandler,
		RouteRegistry:          routeRegistry,
	}, nil
}

// SyncRoutes sync routes từ registry vào database
func (ak *AuthKit[TUser, TRole]) SyncRoutes() error {
	return router.SyncRoutesToDatabase(ak.RouteRegistry, ak.RuleRepo, repository.NewRoleRepository(ak.DB))
}

// InvalidateCache invalidates authorization middleware cache
func (ak *AuthKit[TUser, TRole]) InvalidateCache() {
	ak.AuthorizationMiddleware.InvalidateCache()
}

// AccessType constants
const (
	AccessPublic = models.AccessPublic
	AccessAllow  = models.AccessAllow
	AccessForbid = models.AccessForbid
)

// Repositories - Export repository types và constructors
type (
	UserRepository         = repository.UserRepository
	RoleRepository         = repository.RoleRepository
	RuleRepository         = repository.RuleRepository
	BaseUserRepository[T core.UserInterface] = repository.BaseUserRepository[T]
	BaseRoleRepository[T core.RoleInterface] = repository.BaseRoleRepository[T]
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

// NewBaseUserRepository creates a new base user repository with generic type
func NewBaseUserRepository[TUser UserInterface](db *gorm.DB) *BaseUserRepository[TUser] {
	return repository.NewBaseUserRepository[TUser](db)
}

// NewBaseRoleRepository creates a new base role repository with generic type
func NewBaseRoleRepository[TRole RoleInterface](db *gorm.DB) *BaseRoleRepository[TRole] {
	return repository.NewBaseRoleRepository[TRole](db)
}

// Services - Export service types và constructors
type (
	AuthService = service.AuthService
	RoleService = service.RoleService
	RuleService = service.RuleService
	BaseAuthService[TUser core.UserInterface, TRole core.RoleInterface] = service.BaseAuthService[TUser, TRole]
	BaseRoleService[TRole core.RoleInterface] = service.BaseRoleService[TRole]
)

// Middleware - Export generic middleware types
type (
	BaseAuthMiddleware[TUser core.UserInterface] = middleware.BaseAuthMiddleware[TUser]
	BaseAuthorizationMiddleware[TUser core.UserInterface, TRole core.RoleInterface] = middleware.BaseAuthorizationMiddleware[TUser, TRole]
)

// Handlers - Export generic handler types
type (
	BaseAuthHandler[TUser core.UserInterface, TRole core.RoleInterface] = handlers.BaseAuthHandler[TUser, TRole]
	BaseRoleHandler[TRole core.RoleInterface] = handlers.BaseRoleHandler[TRole]
	BaseRuleHandler[TUser core.UserInterface, TRole core.RoleInterface] = handlers.BaseRuleHandler[TUser, TRole]
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
func NewAuthService(userRepo *UserRepository, roleRepo *RoleRepository, cfg *Config) *AuthService {
	return service.NewAuthService(userRepo, roleRepo, cfg)
}

// NewRoleService creates a new role service
func NewRoleService(roleRepo *RoleRepository) *RoleService {
	return service.NewRoleService(roleRepo)
}

// NewRuleService creates a new rule service
func NewRuleService(ruleRepo *RuleRepository, roleRepo *RoleRepository) *RuleService {
	return service.NewRuleService(ruleRepo, roleRepo)
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

// GetUserFromContextGeneric gets user from context với generic type
func GetUserFromContextGeneric[TUser UserInterface](c *fiber.Ctx) (TUser, bool) {
	return middleware.GetUserFromContextGeneric[TUser](c)
}

func GetUserIDFromContext(c *fiber.Ctx) (string, bool) {
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
