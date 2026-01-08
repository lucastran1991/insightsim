package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"insightsim/internal/database"
	"insightsim/internal/models"
)

// Loader handles loading data from JSON files into the database
type Loader struct {
	db *database.DB
}

// NewLoader creates a new Loader instance
func NewLoader(db *database.DB) *Loader {
	return &Loader{db: db}
}

// LoadFromFile loads data from a JSON file into the database
func (l *Loader) LoadFromFile(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return l.LoadFromReader(file)
}

// LoadFromFolder loads all JSON files from a folder into the database
func (l *Loader) LoadFromFolder(folderPath string) (int, int, error) {
	startTime := time.Now()
	
	// Resolve absolute path if relative
	if !filepath.IsAbs(folderPath) {
		folderPath = filepath.Join(".", folderPath)
	}

	fmt.Printf("[LOAD] Loading data from folder: %s\n", folderPath)
	
	// Read all files in folder
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read folder: %w", err)
	}

	totalCount := 0
	filesCount := 0

	// Process each JSON file
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Only process .json files
		if !strings.HasSuffix(strings.ToLower(file.Name()), ".json") {
			continue
		}

		fileStartTime := time.Now()
		filePath := filepath.Join(folderPath, file.Name())
		fmt.Printf("[LOAD] Processing file: %s\n", file.Name())
		
		count, err := l.LoadFromFile(filePath)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to load file %s: %w", file.Name(), err)
		}

		fileDuration := time.Since(fileStartTime)
		fmt.Printf("[LOAD] Completed file: %s (%d records, took %v)\n", file.Name(), count, fileDuration.Round(time.Millisecond))
		
		totalCount += count
		filesCount++
	}

	totalDuration := time.Since(startTime)
	fmt.Printf("[LOAD] Load completed: %d total records from %d files (total time: %v)\n", 
		totalCount, filesCount, totalDuration.Round(time.Millisecond))

	return totalCount, filesCount, nil
}

// LoadFromReader loads data from an io.Reader into the database
func (l *Loader) LoadFromReader(reader io.Reader) (int, error) {
	var input models.JSONInput
	if err := json.NewDecoder(reader).Decode(&input); err != nil {
		return 0, fmt.Errorf("failed to decode JSON: %w", err)
	}

	conn := l.db.GetConn()
	tx, err := conn.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare statements
	insertStmt, err := tx.Prepare(`
		INSERT INTO insight_raws (tag, timestamp, value, quality)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer insertStmt.Close()

	checkStmt, err := tx.Prepare(`
		SELECT quality FROM insight_raws
		WHERE tag = ? AND timestamp = ?
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare check statement: %w", err)
	}
	defer checkStmt.Close()

	updateStmt, err := tx.Prepare(`
		UPDATE insight_raws
		SET value = ?, quality = ?
		WHERE tag = ? AND timestamp = ?
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer updateStmt.Close()

	totalCount := 0

	for tag, dataPoints := range input.Result {
		for _, dp := range dataPoints {
			// Convert ISO timestamp to Unix milliseconds
			timestamp, err := parseTimestamp(dp.Timestamp)
			if err != nil {
				return 0, fmt.Errorf("invalid timestamp %s for tag %s: %w", dp.Timestamp, tag, err)
			}

			// Check if record exists
			var existingQuality int
			err = checkStmt.QueryRow(tag, timestamp).Scan(&existingQuality)

			if err != nil {
				// Record doesn't exist, insert new
				if err == sql.ErrNoRows {
					_, err = insertStmt.Exec(tag, timestamp, dp.Value, dp.Quality)
					if err != nil {
						return 0, fmt.Errorf("failed to insert record: %w", err)
					}
					totalCount++
				} else {
					return 0, fmt.Errorf("failed to check existing record: %w", err)
				}
			} else {
				// Record exists, check quality
				// Only update if new quality is higher or equal
				if dp.Quality >= existingQuality {
					_, err = updateStmt.Exec(dp.Value, dp.Quality, tag, timestamp)
					if err != nil {
						return 0, fmt.Errorf("failed to update record: %w", err)
					}
					totalCount++
				}
				// If new quality is lower, skip (don't update)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return totalCount, nil
}

// parseTimestamp converts ISO 8601 timestamp string to Unix milliseconds
func parseTimestamp(isoTime string) (int64, error) {
	// Try parsing with different formats
	formats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range formats {
		t, err := time.Parse(format, isoTime)
		if err == nil {
			return t.UnixMilli(), nil
		}
	}

	return 0, fmt.Errorf("unable to parse timestamp: %s", isoTime)
}
