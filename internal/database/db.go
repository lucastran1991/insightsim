package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the database connection
type DB struct {
	conn *sql.DB
}

// NewDB creates a new database connection and runs migrations
func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}

	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// GetConn returns the underlying database connection
func (db *DB) GetConn() *sql.DB {
	return db.conn
}

// migrate creates the necessary tables if they don't exist
func (db *DB) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS insight_raws (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tag TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		value REAL NOT NULL,
		quality INTEGER NOT NULL,
		UNIQUE(tag, timestamp)
	);

	CREATE INDEX IF NOT EXISTS idx_tag_timestamp ON insight_raws(tag, timestamp);
	`

	_, err := db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}
