package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sqlx.DB
}

// New opens a connection to the SQLite database at the given path and
// enables WAL mode and foreign key enforcement.
// It creates the parent directory if it does not exist.
func New(dataSourceName string) (*DB, error) {
	dir := filepath.Dir(dataSourceName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sqlx.Open("sqlite3", dataSourceName+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{DB: db}, nil
}
