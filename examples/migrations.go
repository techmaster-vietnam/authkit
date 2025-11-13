package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	pgmigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// runMigrations runs database migrations using golang-migrate
func runMigrations(db *gorm.DB, dbName string) error {
	// Get underlying *sql.DB from GORM
	sqlDB, err := db.DB()
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Failed to get underlying sql.DB from GORM")
	}

	// Create postgres driver instance
	driver, err := pgmigrate.WithInstance(sqlDB, &pgmigrate.Config{
		DatabaseName: dbName,
	})
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Failed to create postgres driver for migrations")
	}

	// Get migrations directory path
	migrationsPath := filepath.Join(".", "migrations")
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		return goerrorkit.NewSystemError(fmt.Errorf("migrations directory not found: %s", migrationsPath))
	}

	// Create file source from migrations directory
	sourceDriver, err := iofs.New(os.DirFS(migrationsPath), ".")
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Failed to create file source driver").
			WithData(map[string]interface{}{
				"migrations_path": migrationsPath,
			})
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance(
		"iofs",
		sourceDriver,
		"postgres",
		driver,
	)
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Failed to create migrate instance")
	}

	// Run migrations up
	if err := m.Up(); err != nil {
		// Ignore "no change" error (migrations already applied)
		if err == migrate.ErrNoChange {
			fmt.Println("Migrations are up to date")
			return nil
		}
		return goerrorkit.WrapWithMessage(err, "Failed to run migrations")
	}

	fmt.Println("Migrations completed successfully")
	return nil
}

