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
	"github.com/techmaster-vietnam/authkit/utils"
	"gorm.io/gorm"
)

// Config là alias cho config.Config để tránh conflict với package config khác
type Config = config.Config

// Models - Export các models
type (
	User     = models.User
	BaseUser = models.BaseUser
	Role     = models.Role
	BaseRole = models.BaseRole
	Rule     = models.Rule
)

// Interfaces - Export interfaces
type (
	UserInterface = core.UserInterface
	RoleInterface = core.RoleInterface
)

// JWTCustomizer là callback function để tùy chỉnh JWT claims
// Function này được gọi khi tạo JWT token trong quá trình login
// user: User object đang đăng nhập
// roleIDs: Danh sách role IDs của user
// Returns: ClaimsConfig với custom fields để thêm vào JWT token
type JWTCustomizer[TUser UserInterface] = service.JWTCustomizer[TUser]

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
	RoleService *BaseRoleService[TUser, TRole]
	RuleService *RuleService

	// Middleware
	AuthMiddleware          *BaseAuthMiddleware[TUser]
	AuthorizationMiddleware *BaseAuthorizationMiddleware[TUser, TRole]

	// Handlers
	AuthHandler *BaseAuthHandler[TUser, TRole]
	RoleHandler *BaseRoleHandler[TUser, TRole]
	RuleHandler *BaseRuleHandler[TUser, TRole]

	// Route registry
	RouteRegistry *router.RouteRegistry
}

// AuthKitBuilder là builder để tạo AuthKit
type AuthKitBuilder[TUser UserInterface, TRole RoleInterface] struct {
	app           *fiber.App
	db            *gorm.DB
	config        *Config
	userModel     TUser
	roleModel     TRole
	jwtCustomizer JWTCustomizer[TUser]
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

// WithJWTCustomizer set JWT customizer callback để tùy chỉnh JWT claims
// Callback này được gọi khi tạo JWT token trong quá trình login
// Cho phép ứng dụng thêm custom fields vào JWT token (ví dụ: id, full_name, v.v.)
func (b *AuthKitBuilder[TUser, TRole]) WithJWTCustomizer(customizer JWTCustomizer[TUser]) *AuthKitBuilder[TUser, TRole] {
	b.jwtCustomizer = customizer
	return b
}

// Initialize khởi tạo AuthKit với tất cả dependencies
func (b *AuthKitBuilder[TUser, TRole]) Initialize() (*AuthKit[TUser, TRole], error) {
	// Load config nếu chưa có
	if b.config == nil {
		b.config = LoadConfig()
	}

	// Auto migrate với custom models
	if err := b.db.AutoMigrate(&b.userModel, &b.roleModel, &models.Rule{}, &models.RefreshToken{}); err != nil {
		return nil, err
	}

	// Initialize repositories
	userRepo := repository.NewBaseUserRepository[TUser](b.db)
	roleRepo := repository.NewBaseRoleRepository[TRole](b.db)
	ruleRepo := repository.NewRuleRepository(b.db, b.config.ServiceName)
	refreshTokenRepo := repository.NewRefreshTokenRepository(b.db)

	// Initialize middleware với generic types (cần khởi tạo trước để inject vào services)
	authMiddleware := middleware.NewBaseAuthMiddleware(b.config, userRepo)
	authzMiddleware := middleware.NewBaseAuthorizationMiddleware(ruleRepo, roleRepo, userRepo)

	// Initialize services với JWT customizer nếu có
	authService := service.NewBaseAuthService(userRepo, roleRepo, refreshTokenRepo, b.config)
	if b.jwtCustomizer != nil {
		authService.SetJWTCustomizer(b.jwtCustomizer)
	}
	roleService := service.NewBaseRoleService(roleRepo, userRepo)
	ruleService := service.NewRuleService(ruleRepo, repository.NewRoleRepository(b.db))

	// Inject cache invalidator vào services để tự động invalidate cache khi cần
	// Tạo wrapper để implement CacheInvalidator interface
	cacheInvalidator := &cacheInvalidatorWrapper[TUser, TRole]{authzMiddleware: authzMiddleware}
	roleService.SetCacheInvalidator(cacheInvalidator)
	ruleService.SetCacheInvalidator(cacheInvalidator)

	// Initialize handlers với generic types
	authHandler := handlers.NewBaseAuthHandler(authService)
	roleHandler := handlers.NewBaseRoleHandler(roleService)
	ruleHandler := handlers.NewBaseRuleHandler(ruleService, authzMiddleware)

	// Create route registry
	routeRegistry := router.NewRouteRegistry()

	return &AuthKit[TUser, TRole]{
		DB:                      b.db,
		Config:                  b.config,
		UserRepo:                userRepo,
		RoleRepo:                roleRepo,
		RuleRepo:                ruleRepo,
		AuthService:             authService,
		RoleService:             roleService,
		RuleService:             ruleService,
		AuthMiddleware:          authMiddleware,
		AuthorizationMiddleware: authzMiddleware,
		AuthHandler:             authHandler,
		RoleHandler:             roleHandler,
		RuleHandler:             ruleHandler,
		RouteRegistry:           routeRegistry,
	}, nil
}

// SyncRoutes sync routes từ registry vào database
// Tự động invalidate rules cache sau khi sync thành công
func (ak *AuthKit[TUser, TRole]) SyncRoutes() error {
	if err := router.SyncRoutesToDatabase(ak.RouteRegistry, ak.RuleRepo, repository.NewRoleRepository(ak.DB), ak.Config.ServiceName); err != nil {
		return err
	}
	// Tự động invalidate cache sau khi sync routes thành công
	ak.InvalidateCache()
	return nil
}

// InvalidateCache invalidates authorization middleware cache
func (ak *AuthKit[TUser, TRole]) InvalidateCache() {
	ak.AuthorizationMiddleware.InvalidateCache()
}

// RefreshRoleCache refresh role cache bằng cách load lại tất cả roles từ database
// Nên gọi sau khi có thay đổi roles từ bên ngoài (ví dụ: seed data, migration)
func (ak *AuthKit[TUser, TRole]) RefreshRoleCache() error {
	return ak.RoleRepo.RefreshRoleCache()
}

// AccessType constants
const (
	AccessPublic = models.AccessPublic
	AccessAllow  = models.AccessAllow
	AccessForbid = models.AccessForbid
)

// Repositories - Export repository types và constructors
type (
	UserRepository                           = repository.UserRepository
	RoleRepository                           = repository.RoleRepository
	RuleRepository                           = repository.RuleRepository
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
// If serviceName is empty, works in single-app mode (backward compatible)
func NewRuleRepository(db *gorm.DB, serviceName string) *RuleRepository {
	return repository.NewRuleRepository(db, serviceName)
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
	AuthService                                                         = service.AuthService
	RoleService                                                         = service.RoleService
	RuleService                                                         = service.RuleService
	BaseAuthService[TUser core.UserInterface, TRole core.RoleInterface] = service.BaseAuthService[TUser, TRole]
	BaseRoleService[TUser core.UserInterface, TRole core.RoleInterface] = service.BaseRoleService[TUser, TRole]
)

// Middleware - Export generic middleware types
type (
	BaseAuthMiddleware[TUser core.UserInterface]                                    = middleware.BaseAuthMiddleware[TUser]
	BaseAuthorizationMiddleware[TUser core.UserInterface, TRole core.RoleInterface] = middleware.BaseAuthorizationMiddleware[TUser, TRole]
)

// Handlers - Export generic handler types
type (
	BaseAuthHandler[TUser core.UserInterface, TRole core.RoleInterface] = handlers.BaseAuthHandler[TUser, TRole]
	BaseRoleHandler[TUser core.UserInterface, TRole core.RoleInterface] = handlers.BaseRoleHandler[TUser, TRole]
	BaseRuleHandler[TUser core.UserInterface, TRole core.RoleInterface] = handlers.BaseRuleHandler[TUser, TRole]
)

// Service request/response types
type (
	LoginRequest      = service.LoginRequest
	LoginResponse     = service.LoginResponse
	RegisterRequest   = service.RegisterRequest
	AddRoleRequest    = service.AddRoleRequest
	UpdateRuleRequest = service.UpdateRuleRequest
)

// Export ClaimsConfig để ứng dụng có thể sử dụng
type ClaimsConfig = utils.ClaimsConfig

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

// GetRoleIDsFromContext gets role IDs from context (from validated JWT token)
func GetRoleIDsFromContext(c *fiber.Ctx) ([]uint, bool) {
	return middleware.GetRoleIDsFromContext(c)
}

// LoadConfig loads configuration from environment variables
// Đây là wrapper function để tránh conflict với package config của ứng dụng chính
func LoadConfig() *Config {
	return config.LoadConfig()
}

// cacheInvalidatorWrapper wraps BaseAuthorizationMiddleware để implement CacheInvalidator interface
type cacheInvalidatorWrapper[TUser UserInterface, TRole RoleInterface] struct {
	authzMiddleware *BaseAuthorizationMiddleware[TUser, TRole]
}

// InvalidateRulesCache invalidates rules cache trong authorization middleware
func (w *cacheInvalidatorWrapper[TUser, TRole]) InvalidateRulesCache() {
	if w.authzMiddleware != nil {
		w.authzMiddleware.InvalidateCache()
	}
}
