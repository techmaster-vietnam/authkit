package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

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
		_ = goerrorkit.WrapWithMessage(err, "Warning: .env file not found, using default values or environment variables")
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
		panic(goerrorkit.NewSystemError(err).
			WithData(map[string]interface{}{
				"host":     dbHost,
				"port":     dbPort,
				"user":     dbUser,
				"database": dbName,
				"sslmode":  dbSSLMode,
			}))
	}
	// 5. Reset database (only if RESET_DB=true)
	if err := resetDatabase(db); err != nil {
		panic(err)
	}

	// 6. Run migrations
	if err := runMigrations(db, dbName); err != nil {
		panic(goerrorkit.NewSystemError(err).
			WithData(map[string]interface{}{
				"operation": "migration",
				"database":  dbName,
			}))
	}

	// 7. Seed initial data (roles and rules)
	if err := SeedData(db); err != nil {
		panic(goerrorkit.WrapWithMessage(err, "Failed to seed initial data").
			WithData(map[string]interface{}{
				"operation": "seed_data",
			}))
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

	// 11. Initialize services
	authService := authkit.NewAuthService(userRepo, roleRepo, cfg)
	roleService := authkit.NewRoleService(roleRepo)
	ruleService := authkit.NewRuleService(ruleRepo)

	// 12. Initialize middleware
	authMiddleware := authkit.NewAuthMiddleware(cfg, userRepo)
	authzMiddleware := authkit.NewAuthorizationMiddleware(ruleRepo, roleRepo, userRepo)

	// 13. Initialize handlers
	authHandler := authkit.NewAuthHandler(authService)
	roleHandler := authkit.NewRoleHandler(roleService)
	ruleHandler := authkit.NewRuleHandler(ruleService, authzMiddleware)
	blogHandler := NewBlogHandler() // Application-specific handler

	// 14. Setup routes
	setupRoutes(app, authHandler, roleHandler, ruleHandler, blogHandler, authMiddleware, authzMiddleware)

	// 15. Check if port is available before starting server
	port := cfg.Server.Port
	if isPortInUse(port) {
		processInfo := getPortProcessInfo(port)
		errorMsg := fmt.Sprintf("Cổng %s đang được sử dụng", port)
		if processInfo != "" {
			errorMsg += fmt.Sprintf("\nThông tin process đang sử dụng cổng:\n%s", processInfo)
			errorMsg += "\n\nĐể giải phóng cổng, bạn có thể:\n"
			errorMsg += "1. Dừng process đang sử dụng cổng\n"
			errorMsg += "2. Hoặc thay đổi cổng trong file .env (SERVER_PORT=<cổng_khác>)"
		}
		panic(goerrorkit.WrapWithMessage(fmt.Errorf("port %s is already in use", port), errorMsg).
			WithData(map[string]interface{}{
				"port":         port,
				"process_info": processInfo,
			}))
	}

	// 16. Start server
	fmt.Printf("Server starting on port %s\n", port)
	if err := app.Listen(":" + port); err != nil {
		// Check if error is related to port already in use
		if strings.Contains(err.Error(), "address already in use") || strings.Contains(err.Error(), "bind: address already in use") {
			processInfo := getPortProcessInfo(port)
			errorMsg := fmt.Sprintf("Không thể khởi động server: Cổng %s đang được sử dụng", port)
			if processInfo != "" {
				errorMsg += fmt.Sprintf("\nThông tin process đang sử dụng cổng:\n%s", processInfo)
			}
			errorMsg += "\n\nĐể giải phóng cổng, bạn có thể:\n"
			errorMsg += "1. Dừng process đang sử dụng cổng\n"
			errorMsg += "2. Hoặc thay đổi cổng trong file .env (SERVER_PORT=<cổng_khác>)"
			panic(goerrorkit.WrapWithMessage(err, errorMsg).
				WithData(map[string]interface{}{
					"port":         port,
					"process_info": processInfo,
				}))
		}
		panic(goerrorkit.NewSystemError(err).
			WithData(map[string]interface{}{
				"port": port,
			}))
	}
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
	// Serve favicon
	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		return c.SendFile("favicon.png")
	})

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
	blogs.Get("/", blogHandler.List) // Public: list blogs

	blogsProtected := blogs.Group("", authMiddleware.RequireAuth(), authzMiddleware.Authorize())
	blogsProtected.Get("/:id", blogHandler.GetByID) // Protected: view detail
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

// isPortInUse checks if a port is already in use
func isPortInUse(port string) bool {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return true
	}
	ln.Close()
	return false
}

// getPortProcessInfo gets information about the process using the port
func getPortProcessInfo(port string) string {
	// Try to get process info using lsof command (works on macOS and Linux)
	cmd := exec.Command("lsof", "-i", ":"+port)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return ""
	}

	// Skip header line and format the output
	var info strings.Builder
	info.WriteString("COMMAND    PID    USER    FD    TYPE    DEVICE    SIZE/OFF    NODE    NAME\n")
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			info.WriteString(lines[i])
			info.WriteString("\n")
		}
	}

	return info.String()
}
