package main

import (
	"fmt"
	"os"

	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// resetDatabase drops all tables and migrations (useful for development/testing)
// WARNING: This will delete all data! Use with caution.
// Only runs if RESET_DB=true environment variable is set
func resetDatabase(db *gorm.DB) error {
	// Safety check: only reset if explicitly enabled via environment variable
	if os.Getenv("RESET_DB") != "true" {
		return nil // Skip reset if not explicitly enabled
	}

	fmt.Println("⚠️  WARNING: Resetting database (RESET_DB=true is set)")

	// Get underlying *sql.DB from GORM
	sqlDB, err := db.DB()
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Failed to get underlying sql.DB from GORM")
	}

	// Step 1: Drop all tables directly (more reliable than rolling back migrations)
	fmt.Println("Dropping all tables...")
	dropTablesSQL := `
		DROP TABLE IF EXISTS blogs CASCADE;
		DROP TABLE IF EXISTS user_roles CASCADE;
		DROP TABLE IF EXISTS rules CASCADE;
		DROP TABLE IF EXISTS roles CASCADE;
		DROP TABLE IF EXISTS users CASCADE;
		DROP TABLE IF EXISTS schema_migrations CASCADE;
	`
	if _, err := sqlDB.Exec(dropTablesSQL); err != nil {
		return goerrorkit.WrapWithMessage(err, "Failed to drop tables")
	}
	fmt.Println("All tables dropped successfully")

	// Step 2: Reset migration version in schema_migrations table
	// (This ensures migrations will run from the beginning)
	fmt.Println("Resetting migration version...")
	createSchemaMigrationsSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT NOT NULL PRIMARY KEY,
			dirty BOOLEAN NOT NULL
		);
		DELETE FROM schema_migrations;
	`
	if _, err := sqlDB.Exec(createSchemaMigrationsSQL); err != nil {
		// Ignore error if table doesn't exist, it will be created by migrate
		fmt.Println("Note: schema_migrations table will be created by migrate")
	}

	fmt.Println("Database reset completed successfully")
	return nil
}
