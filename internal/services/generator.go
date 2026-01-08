package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"insightsim/internal/database"
	"insightsim/internal/models"
)

// Generator handles generating dummy timeseries data
type Generator struct {
	db *database.DB
}

// NewGenerator creates a new Generator instance
func NewGenerator(db *database.DB) *Generator {
	return &Generator{db: db}
}

// GenerateDummyData generates dummy data for all tags from tag_list.json
func (g *Generator) GenerateDummyData(tagListFile string, minValue, maxValue float64) (int, int, error) {
	// Read tag_list.json from config
	tagListPath := tagListFile
	if !filepath.IsAbs(tagListPath) {
		tagListPath = filepath.Join(".", tagListPath)
	}

	data, err := os.ReadFile(tagListPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read tag_list.json: %w", err)
	}

	var tagList models.TagList
	if err := json.Unmarshal(data, &tagList); err != nil {
		return 0, 0, fmt.Errorf("failed to parse tag_list.json: %w", err)
	}

	// Parse tags
	tags := strings.Split(tagList.TagList, ",")
	for i, tag := range tags {
		tags[i] = strings.TrimSpace(tag)
	}

	// Time range: 2025-12-01 00:00:00 to 2026-01-31 23:59:59
	startTime := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	// Calculate total minutes
	totalMinutes := int(endTime.Sub(startTime).Minutes()) + 1

	// Delete all existing records before generating new batch
	conn := g.db.GetConn()
	if err := g.db.DeleteAllRecords(); err != nil {
		return 0, 0, fmt.Errorf("failed to delete existing records: %w", err)
	}

	tx, err := conn.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare statements
	insertStmt, err := tx.Prepare(`
		INSERT INTO insight_raws (tag, timestamp, value, quality)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer insertStmt.Close()

	checkStmt, err := tx.Prepare(`
		SELECT quality FROM insight_raws
		WHERE tag = ? AND timestamp = ?
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to prepare check statement: %w", err)
	}
	defer checkStmt.Close()

	updateStmt, err := tx.Prepare(`
		UPDATE insight_raws
		SET value = ?, quality = ?
		WHERE tag = ? AND timestamp = ?
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer updateStmt.Close()

	totalRecords := 0
	rand.Seed(time.Now().UnixNano())

	// Process each tag
	for tagIdx, tag := range tags {
		if tag == "" {
			continue
		}

		// Random base value within configured range for this tag
		baseValue := minValue + rand.Float64()*(maxValue-minValue)
		currentValue := baseValue

		// Generate records for each minute
		for minute := 0; minute < totalMinutes; minute++ {
			currentTime := startTime.Add(time.Duration(minute) * time.Minute)
			timestamp := currentTime.UnixMilli()

			// Generate new value with change < 30%
			// Change can be between -30% and +30%
			changePercent := (rand.Float64()*2 - 1) * 0.3 // -0.3 to 0.3
			newValue := currentValue * (1 + changePercent)

			// Ensure value stays within reasonable bounds
			if newValue < 0 {
				newValue = 0
			}

			quality := 3 // Fixed quality value

			// Check if record exists
			var existingQuality int
			err = checkStmt.QueryRow(tag, timestamp).Scan(&existingQuality)

			if err != nil {
				// Record doesn't exist, insert new
				if err == sql.ErrNoRows {
					_, err = insertStmt.Exec(tag, timestamp, newValue, quality)
					if err != nil {
						return 0, 0, fmt.Errorf("failed to insert record for tag %s at %v: %w", tag, currentTime, err)
					}
					totalRecords++
				} else {
					return 0, 0, fmt.Errorf("failed to check existing record: %w", err)
				}
			} else {
				// Record exists, check quality
				// Only update if new quality is higher or equal
				if quality >= existingQuality {
					_, err = updateStmt.Exec(newValue, quality, tag, timestamp)
					if err != nil {
						return 0, 0, fmt.Errorf("failed to update record: %w", err)
					}
					totalRecords++
				}
			}

			// Update current value for next iteration
			currentValue = newValue

			// Commit in batches to avoid memory issues
			if totalRecords%10000 == 0 {
				// Close old statements
				insertStmt.Close()
				checkStmt.Close()
				updateStmt.Close()

				if err := tx.Commit(); err != nil {
					return 0, 0, fmt.Errorf("failed to commit batch: %w", err)
				}
				// Start new transaction
				tx, err = conn.Begin()
				if err != nil {
					return 0, 0, fmt.Errorf("failed to begin new transaction: %w", err)
				}
				// Re-prepare statements for new transaction
				insertStmt, err = tx.Prepare(`
					INSERT INTO insight_raws (tag, timestamp, value, quality)
					VALUES (?, ?, ?, ?)
				`)
				if err != nil {
					return 0, 0, fmt.Errorf("failed to prepare insert statement: %w", err)
				}
				checkStmt, err = tx.Prepare(`
					SELECT quality FROM insight_raws
					WHERE tag = ? AND timestamp = ?
				`)
				if err != nil {
					return 0, 0, fmt.Errorf("failed to prepare check statement: %w", err)
				}
				updateStmt, err = tx.Prepare(`
					UPDATE insight_raws
					SET value = ?, quality = ?
					WHERE tag = ? AND timestamp = ?
				`)
				if err != nil {
					return 0, 0, fmt.Errorf("failed to prepare update statement: %w", err)
				}
			}
		}

		// Log progress every 10 tags
		if (tagIdx+1)%10 == 0 {
			fmt.Printf("Processed %d/%d tags...\n", tagIdx+1, len(tags))
		}
	}

	// Final commit
	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit final transaction: %w", err)
	}

	return totalRecords, len(tags), nil
}
