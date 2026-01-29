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

// DeleteAllRecords deletes all records from insight_raws table
func (db *DB) DeleteAllRecords() error {
	_, err := db.conn.Exec("DELETE FROM insight_raws")
	if err != nil {
		return fmt.Errorf("failed to delete all records: %w", err)
	}
	return nil
}

// DeleteTagRecords deletes all records for a specific tag
func (db *DB) DeleteTagRecords(tag string) error {
	_, err := db.conn.Exec("DELETE FROM insight_raws WHERE tag = ?", tag)
	if err != nil {
		return fmt.Errorf("failed to delete records for tag %s: %w", tag, err)
	}
	return nil
}

// TagStats holds per-tag aggregate stats from the database
type TagStats struct {
	Tag   string
	Count int
	MinTs int64
	MaxTs int64
}

// ListTagsWithStats returns all tags with record count and min/max timestamp
func (db *DB) ListTagsWithStats() ([]TagStats, error) {
	rows, err := db.conn.Query("SELECT tag, COUNT(*) as count, MIN(timestamp) as min_ts, MAX(timestamp) as max_ts FROM insight_raws GROUP BY tag")
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	defer rows.Close()
	var result []TagStats
	for rows.Next() {
		var s TagStats
		if err := rows.Scan(&s.Tag, &s.Count, &s.MinTs, &s.MaxTs); err != nil {
			return nil, fmt.Errorf("scan tag stats: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// ListTagNames returns distinct tag names that have at least one record (fast; no aggregation)
func (db *DB) ListTagNames() ([]string, error) {
	rows, err := db.conn.Query("SELECT DISTINCT tag FROM insight_raws ORDER BY tag")
	if err != nil {
		return nil, fmt.Errorf("failed to list tag names: %w", err)
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("scan tag name: %w", err)
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}

// TagLastRecord holds tag and its latest timestamp
type TagLastRecord struct {
	Tag   string
	MaxTs int64
}

// ListTagsLastRecord returns each tag and its max timestamp (lightweight, no COUNT/MIN)
func (db *DB) ListTagsLastRecord() ([]TagLastRecord, error) {
	rows, err := db.conn.Query("SELECT tag, MAX(timestamp) as max_ts FROM insight_raws GROUP BY tag")
	if err != nil {
		return nil, fmt.Errorf("failed to list tags last record: %w", err)
	}
	defer rows.Close()
	var result []TagLastRecord
	for rows.Next() {
		var r TagLastRecord
		if err := rows.Scan(&r.Tag, &r.MaxTs); err != nil {
			return nil, fmt.Errorf("scan tag last record: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// ListTagsLastRecordForTags returns max timestamp only for the given tags (fast: uses WHERE tag IN (...))
func (db *DB) ListTagsLastRecordForTags(tags []string) ([]TagLastRecord, error) {
	if len(tags) == 0 {
		return nil, nil
	}
	// Build placeholder list for IN clause
	args := make([]interface{}, len(tags))
	for i, t := range tags {
		args[i] = t
	}
	placeholders := ""
	for i := range tags {
		if i > 0 {
			placeholders += ",?"
		} else {
			placeholders = "?"
		}
	}
	query := "SELECT tag, MAX(timestamp) as max_ts FROM insight_raws WHERE tag IN (" + placeholders + ") GROUP BY tag"
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags last record for tags: %w", err)
	}
	defer rows.Close()
	var result []TagLastRecord
	for rows.Next() {
		var r TagLastRecord
		if err := rows.Scan(&r.Tag, &r.MaxTs); err != nil {
			return nil, fmt.Errorf("scan tag last record: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// TagRow is a row from the tags table
type TagRow struct {
	Tag       string
	CreatedAt string
	UpdatedAt string
	Source    string
}

// InsertTag inserts a tag into the tags table
func (db *DB) InsertTag(tag, createdAt, updatedAt, source string) error {
	_, err := db.conn.Exec("INSERT INTO tags (tag, created_at, updated_at, source) VALUES (?, ?, ?, ?)",
		tag, createdAt, updatedAt, source)
	if err != nil {
		return fmt.Errorf("insert tag: %w", err)
	}
	return nil
}

// InsertTagIfNotExists inserts a tag into the tags table if it does not already exist (for upload/load)
func (db *DB) InsertTagIfNotExists(tag, createdAt, updatedAt, source string) error {
	_, err := db.conn.Exec("INSERT OR IGNORE INTO tags (tag, created_at, updated_at, source) VALUES (?, ?, ?, ?)",
		tag, createdAt, updatedAt, source)
	if err != nil {
		return fmt.Errorf("insert tag if not exists: %w", err)
	}
	return nil
}

// UpdateTagUpdatedAt sets updated_at for a tag
func (db *DB) UpdateTagUpdatedAt(tag, updatedAt string) error {
	_, err := db.conn.Exec("UPDATE tags SET updated_at = ? WHERE tag = ?", updatedAt, tag)
	if err != nil {
		return fmt.Errorf("update tag: %w", err)
	}
	return nil
}

// DeleteTag removes a tag from the tags table
func (db *DB) DeleteTag(tag string) error {
	_, err := db.conn.Exec("DELETE FROM tags WHERE tag = ?", tag)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	return nil
}

// ListTagsPaginated returns a page of rows from the tags table
func (db *DB) ListTagsPaginated(offset, limit int) ([]TagRow, error) {
	rows, err := db.conn.Query("SELECT tag, created_at, updated_at, source FROM tags ORDER BY tag LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()
	var result []TagRow
	for rows.Next() {
		var r TagRow
		if err := rows.Scan(&r.Tag, &r.CreatedAt, &r.UpdatedAt, &r.Source); err != nil {
			return nil, fmt.Errorf("scan tag row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// CountTags returns the total number of tags
func (db *DB) CountTags() (int, error) {
	var n int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM tags").Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count tags: %w", err)
	}
	return n, nil
}

// CountTagsWithSearch returns the number of tags whose name contains the search term (case-insensitive)
func (db *DB) CountTagsWithSearch(search string) (int, error) {
	pattern := "%" + search + "%"
	var n int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM tags WHERE LOWER(tag) LIKE LOWER(?)", pattern).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count tags with search: %w", err)
	}
	return n, nil
}

// ListTagsPaginatedWithSearch returns a page of tag rows filtered by search term (case-insensitive)
func (db *DB) ListTagsPaginatedWithSearch(offset, limit int, search string) ([]TagRow, error) {
	pattern := "%" + search + "%"
	rows, err := db.conn.Query("SELECT tag, created_at, updated_at, source FROM tags WHERE LOWER(tag) LIKE LOWER(?) ORDER BY tag LIMIT ? OFFSET ?", pattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tags with search: %w", err)
	}
	defer rows.Close()
	var result []TagRow
	for rows.Next() {
		var r TagRow
		if err := rows.Scan(&r.Tag, &r.CreatedAt, &r.UpdatedAt, &r.Source); err != nil {
			return nil, fmt.Errorf("scan tag row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// ListTagNamesFromTagsTable returns tag names from the tags table (for generator and API)
func (db *DB) ListTagNamesFromTagsTable() ([]string, error) {
	rows, err := db.conn.Query("SELECT tag FROM tags ORDER BY tag")
	if err != nil {
		return nil, fmt.Errorf("list tag names: %w", err)
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("scan tag name: %w", err)
		}
		result = append(result, tag)
	}
	return result, rows.Err()
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

	CREATE TABLE IF NOT EXISTS tags (
		tag TEXT PRIMARY KEY,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		source TEXT NOT NULL DEFAULT 'custom'
	);
	`

	_, err := db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}
