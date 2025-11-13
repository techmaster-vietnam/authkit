package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/joho/godotenv"
	"github.com/techmaster-vietnam/authkit"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 0. Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using default values or environment variables")
	}

	// 1. Initialize goerrorkit logger
	goerrorkit.InitLogger(goerrorkit.LoggerOptions{
		ConsoleOutput: true,
		FileOutput:    true,
		FilePath:      "logs/errors.log",
		JSONFormat:    true,
		MaxFileSize:   10,
		MaxBackups:    5,
		MaxAge:        30,
		LogLevel:      "info",
	})

	// 2. Configure stack trace for this application
	goerrorkit.ConfigureForApplication("main")

	// 3. Load configuration
	cfg := authkit.LoadConfig()

	// 4. Connect to database
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "authkit")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 5. Run migrations
	// Migrate AuthKit models
	if err := db.AutoMigrate(
		&authkit.User{},
		&authkit.Role{},
		&authkit.Rule{},
	); err != nil {
		log.Fatal("Failed to migrate AuthKit models:", err)
	}

	// Migrate application models (Blog)
	if err := db.AutoMigrate(
		&Blog{},
	); err != nil {
		log.Fatal("Failed to migrate Blog model:", err)
	}

	// 6. Initialize roles
	if err := initRoles(db); err != nil {
		log.Fatal("Failed to initialize roles:", err)
	}

	// 7. Initialize rules
	if err := initRules(db); err != nil {
		log.Fatal("Failed to initialize rules:", err)
	}

	// 8. Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Blog Management System",
	})

	// 9. Add middleware (RequestID must be before ErrorHandler)
	app.Use(requestid.New())
	app.Use(logger.New())
	app.Use(goerrorkit.FiberErrorHandler()) // goerrorkit error handler
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// 10. Initialize repositories
	userRepo := authkit.NewUserRepository(db)
	roleRepo := authkit.NewRoleRepository(db)
	ruleRepo := authkit.NewRuleRepository(db)
	blogRepo := NewBlogRepository(db) // Application-specific repository

	// 11. Initialize services
	authService := authkit.NewAuthService(userRepo, cfg)
	roleService := authkit.NewRoleService(roleRepo)
	ruleService := authkit.NewRuleService(ruleRepo)
	blogService := NewBlogService(blogRepo, userRepo, roleRepo) // Application-specific service

	// 12. Initialize middleware
	authMiddleware := authkit.NewAuthMiddleware(cfg, userRepo)
	authzMiddleware := authkit.NewAuthorizationMiddleware(ruleRepo, roleRepo, userRepo)

	// 13. Initialize handlers
	authHandler := authkit.NewAuthHandler(authService)
	roleHandler := authkit.NewRoleHandler(roleService)
	ruleHandler := authkit.NewRuleHandler(ruleService, authzMiddleware)
	blogHandler := NewBlogHandler(blogService, roleRepo) // Application-specific handler

	// 14. Setup routes
	setupRoutes(app, authHandler, roleHandler, ruleHandler, blogHandler, authMiddleware, authzMiddleware)

	// 15. Start server
	log.Printf("Server starting on port %s", cfg.Server.Port)
	if err := app.Listen(":" + cfg.Server.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// initRoles initializes default roles
func initRoles(db *gorm.DB) error {
	roleRepo := authkit.NewRoleRepository(db)

	roles := []string{"admin", "editor", "author", "reader"}

	for _, roleName := range roles {
		_, err := roleRepo.GetByName(roleName)
		if err == nil {
			// Role already exists
			continue
		}

		role := &authkit.Role{
			Name: roleName,
		}

		if err := roleRepo.Create(role); err != nil {
			log.Printf("Failed to create role %s: %v", roleName, err)
			return err
		}
		log.Printf("Created role: %s", roleName)
	}

	return nil
}

// initRules initializes default rules for blog management
func initRules(db *gorm.DB) error {
	ruleRepo := authkit.NewRuleRepository(db)

	rules := []authkit.Rule{
		// Public endpoints
		{
			Method:   "POST",
			Path:     "/api/auth/login",
			Type:     authkit.RuleTypePublic,
			Roles:    []string{},
			Priority: 100,
		},
		{
			Method:   "POST",
			Path:     "/api/auth/register",
			Type:     authkit.RuleTypePublic,
			Roles:    []string{},
			Priority: 100,
		},
		{
			Method:   "GET",
			Path:     "/api/blogs",
			Type:     authkit.RuleTypePublic,
			Roles:    []string{},
			Priority: 100,
		},
		{
			Method:   "GET",
			Path:     "/",
			Type:     authkit.RuleTypePublic,
			Roles:    []string{},
			Priority: 100,
		},

		// Authenticated endpoints
		{
			Method:   "GET",
			Path:     "/api/auth/profile",
			Type:     authkit.RuleTypeAuth,
			Roles:    []string{},
			Priority: 90,
		},
		{
			Method:   "PUT",
			Path:     "/api/auth/profile",
			Type:     authkit.RuleTypeAuth,
			Roles:    []string{},
			Priority: 90,
		},
		{
			Method:   "DELETE",
			Path:     "/api/auth/profile",
			Type:     authkit.RuleTypeAuth,
			Roles:    []string{},
			Priority: 90,
		},
		{
			Method:   "POST",
			Path:     "/api/auth/change-password",
			Type:     authkit.RuleTypeAuth,
			Roles:    []string{},
			Priority: 90,
		},
		{
			Method:   "GET",
			Path:     "/api/blogs/my",
			Type:     authkit.RuleTypeAuth,
			Roles:    []string{},
			Priority: 90,
		},

		// Reader can view blog details
		{
			Method:   "GET",
			Path:     "/api/blogs/*",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"reader", "author", "editor", "admin"},
			Priority: 80,
		},

		// Author can create, edit, delete their own blogs
		{
			Method:   "POST",
			Path:     "/api/blogs",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"author", "editor", "admin"},
			Priority: 80,
		},
		{
			Method:   "PUT",
			Path:     "/api/blogs/*",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"author", "editor", "admin"},
			Priority: 80,
		},
		{
			Method:   "DELETE",
			Path:     "/api/blogs/*",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"author", "editor", "admin"},
			Priority: 80,
		},

		// Admin endpoints
		{
			Method:   "GET",
			Path:     "/api/roles",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"admin"},
			Priority: 70,
		},
		{
			Method:   "POST",
			Path:     "/api/roles",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"admin"},
			Priority: 70,
		},
		{
			Method:   "DELETE",
			Path:     "/api/roles/*",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"admin"},
			Priority: 70,
		},
		{
			Method:   "GET",
			Path:     "/api/rules",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"admin"},
			Priority: 70,
		},
		{
			Method:   "POST",
			Path:     "/api/rules",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"admin"},
			Priority: 70,
		},
		{
			Method:   "PUT",
			Path:     "/api/rules/*",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"admin"},
			Priority: 70,
		},
		{
			Method:   "DELETE",
			Path:     "/api/rules/*",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"admin"},
			Priority: 70,
		},
		{
			Method:   "GET",
			Path:     "/api/users",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"admin"},
			Priority: 70,
		},
		{
			Method:   "GET",
			Path:     "/api/users/*/roles",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"admin"},
			Priority: 70,
		},
		{
			Method:   "POST",
			Path:     "/api/users/*/roles/*",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"admin"},
			Priority: 70,
		},
		{
			Method:   "DELETE",
			Path:     "/api/users/*/roles/*",
			Type:     authkit.RuleTypeAllow,
			Roles:    []string{"admin"},
			Priority: 70,
		},
	}

	// Create rules
	for _, rule := range rules {
		// Check if rule already exists
		_, err := ruleRepo.GetByMethodAndPath(rule.Method, rule.Path)
		if err == nil {
			// Rule already exists
			continue
		}

		if err := ruleRepo.Create(&rule); err != nil {
			log.Printf("Failed to create rule %s %s: %v", rule.Method, rule.Path, err)
			return err
		}
		log.Printf("Created rule: %s %s", rule.Method, rule.Path)
	}

	return nil
}

// setupRoutes sets up all routes
func setupRoutes(
	app *fiber.App,
	authHandler *authkit.AuthHandler,
	roleHandler *authkit.RoleHandler,
	ruleHandler *authkit.RuleHandler,
	blogHandler *BlogHandler, // Application-specific handler
	authMiddleware *authkit.AuthMiddleware,
	authzMiddleware *authkit.AuthorizationMiddleware,
) {
	// Serve static HTML file
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("index.html")
	})

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

	// Blog routes
	blogs := api.Group("/blogs")
	blogs.Get("/", blogHandler.List)                                                                  // Public: list blogs
	blogs.Get("/:id", blogHandler.GetByID, authMiddleware.RequireAuth(), authzMiddleware.Authorize()) // Protected: view detail

	blogsProtected := blogs.Group("", authMiddleware.RequireAuth(), authzMiddleware.Authorize())
	blogsProtected.Post("/", blogHandler.Create)
	blogsProtected.Put("/:id", blogHandler.Update)
	blogsProtected.Delete("/:id", blogHandler.Delete)
	blogsProtected.Get("/my", blogHandler.ListMyBlogs)

	// Role routes (admin only)
	roles := api.Group("/roles", authMiddleware.RequireAuth(), authzMiddleware.Authorize())
	roles.Get("/", roleHandler.ListRoles)
	roles.Post("/", roleHandler.AddRole)
	roles.Delete("/:id", roleHandler.RemoveRole)
	roles.Get("/:role_name/users", roleHandler.ListUsersHasRole)

	// User role routes (admin only)
	users := api.Group("/users", authMiddleware.RequireAuth(), authzMiddleware.Authorize())
	users.Get("/:user_id/roles", roleHandler.ListRolesOfUser)
	users.Post("/:user_id/roles/:role_id", roleHandler.AddRoleToUser)
	users.Delete("/:user_id/roles/:role_id", roleHandler.RemoveRoleFromUser)
	users.Get("/:user_id/roles/:role_name/check", roleHandler.CheckUserHasRole)

	// Rule routes (admin only)
	rules := api.Group("/rules", authMiddleware.RequireAuth(), authzMiddleware.Authorize())
	rules.Get("/", ruleHandler.ListRules)
	rules.Post("/", ruleHandler.AddRule)
	rules.Put("/:id", ruleHandler.UpdateRule)
	rules.Delete("/:id", ruleHandler.RemoveRule)
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
