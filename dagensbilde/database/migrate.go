package database

import (
	"fmt"
	"os"

	"github.com/pressly/goose/v3"
)

// Migrate runs all pending goose migrations from the given directory against
// the database.  Pass the path to the folder containing the .sql files
// (e.g. "sql").
func (db *DB) Migrate(migrationsDir string) error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	goose.SetLogger(goose.NopLogger())

	if _, err := os.Stat(migrationsDir); err != nil {
		return fmt.Errorf("migrations dir: %w", err)
	}

	if err := goose.Up(db.DB.DB, migrationsDir); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
