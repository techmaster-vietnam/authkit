package main

import (
	"fmt"
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

	// 7. Seed initial data (roles and users only, rules sẽ được sync từ routes)
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

	// 10. Khởi tạo AuthKit với Generic API - Sử dụng CustomUser với mobile và address
	// CustomUser embed BaseUser nên vẫn tương thích hoàn toàn với AuthKit
	// Lưu ý: Phải dùng pointer types cho generics
	ak, err := authkit.New[*CustomUser, *authkit.BaseRole](app, db).
		WithConfig(cfg).
		WithUserModel(&CustomUser{}).
		WithRoleModel(&authkit.BaseRole{}).
		Initialize()

	if err != nil {
		panic(goerrorkit.WrapWithMessage(err, "Failed to initialize AuthKit").
			WithData(map[string]interface{}{
				"operation": "initialize_authkit",
			}))
	}

	blogHandler := NewBlogHandler() // Application-specific handler
	demoHandler := NewDemoHandler(ak) // Demo handler for new features

	// 11. Setup routes với fluent API
	setupRoutes(app, ak, blogHandler, demoHandler)

	// 12. Sync routes từ code vào database
	if err := ak.SyncRoutes(); err != nil {
		panic(goerrorkit.WrapWithMessage(err, "Failed to sync routes to database").
			WithData(map[string]interface{}{
				"operation": "sync_routes",
			}))
	}

	// 13. Refresh authorization middleware cache sau khi sync routes
	ak.InvalidateCache()

	// 17. Start server
	if err := app.Listen(":" + cfg.Server.Port); err != nil {
		panic(goerrorkit.NewSystemError(err))
	}
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
