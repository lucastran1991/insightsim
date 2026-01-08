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
func (g *Generator) GenerateDummyData(tagListFile string, minValue, maxValue float64, useSequential bool, startTimeStr, endTimeStr string) (int, int, error) {
	generateStartTime := time.Now()

	// Read tag_list.json from config
	tagListPath := tagListFile
	if !filepath.IsAbs(tagListPath) {
		tagListPath = filepath.Join(".", tagListPath)
	}

	fmt.Printf("[GENERATE] Reading tag list from: %s\n", tagListPath)
	data, err := os.ReadFile(tagListPath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read tag_list.json: %w", err)
	}

	var tagList models.TagList
	if err := json.Unmarshal(data, &tagList); err != nil {
		return 0, 0, fmt.Errorf("failed to parse tag_list.json: %w", err)
	}

	// Parse tags (support both "tag_list" and "tag" keys)
	tagListStr := tagList.GetTagList()
	if tagListStr == "" {
		return 0, 0, fmt.Errorf("no tags found in tag_list.json (check 'tag_list' or 'tag' key)")
	}

	tags := strings.Split(tagListStr, ",")
	for i, tag := range tags {
		tags[i] = strings.TrimSpace(tag)
	}

	fmt.Printf("[GENERATE] Parsed %d tags from tag list\n", len(tags))

	// Parse time range from config
	timeFormat := "2006-01-02T15:04:05"
	
	// Set defaults if empty
	if startTimeStr == "" {
		startTimeStr = "2025-12-01T00:00:00"
	}
	if endTimeStr == "" {
		endTimeStr = "2026-01-31T23:59:59"
	}
	
	startTime, err := time.Parse(timeFormat, startTimeStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid generation_start_time format '%s': %w (expected format: %s)", startTimeStr, err, timeFormat)
	}
	startTime = startTime.UTC()
	
	endTime, err := time.Parse(timeFormat, endTimeStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid generation_end_time format '%s': %w (expected format: %s)", endTimeStr, err, timeFormat)
	}
	endTime = endTime.UTC()
	
	// Validate that start time is before end time
	if startTime.After(endTime) || startTime.Equal(endTime) {
		return 0, 0, fmt.Errorf("invalid time range: generation_start_time (%s) must be before generation_end_time (%s)", startTimeStr, endTimeStr)
	}

	// Calculate total minutes
	totalMinutes := int(endTime.Sub(startTime).Minutes()) + 1

	// Delete all existing records before generating new batch
	conn := g.db.GetConn()
	fmt.Printf("[GENERATE] Deleting all existing records...\n")
	deleteStart := time.Now()
	if err := g.db.DeleteAllRecords(); err != nil {
		return 0, 0, fmt.Errorf("failed to delete existing records: %w", err)
	}
	fmt.Printf("[GENERATE] Deleted existing records (took %v)\n", time.Since(deleteStart).Round(time.Millisecond))

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

	// Log generation start
	fmt.Printf("[GENERATE] Starting generation for %d tags, time range: %s to %s, value range: %.2f-%.2f\n",
		len(tags), startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"), minValue, maxValue)

	// Process each tag
	for _, tag := range tags {
		if tag == "" {
			continue
		}

		tagStartTime := time.Now()
		tagRecords := 0

		// Initialize base value and current value for sequential generation
		var baseValue float64
		var currentValue float64
		if useSequential {
			// Random base value within configured range for this tag
			baseValue = minValue + rand.Float64()*(maxValue-minValue)
			currentValue = baseValue
		}

		// Generate records for each minute
		for minute := 0; minute < totalMinutes; minute++ {
			currentTime := startTime.Add(time.Duration(minute) * time.Minute)
			timestamp := currentTime.UnixMilli()

			// Generate value based on generation mode
			var newValue float64
			if useSequential {
				// Sequential generation: change from previous value with 30% restriction
				changePercent := (rand.Float64()*2 - 1) * 0.3 // -0.3 to 0.3
				newValue = currentValue * (1 + changePercent)
			} else {
				// Completely random value within configured range [minValue, maxValue]
				newValue = minValue + rand.Float64()*(maxValue-minValue)
			}

			// Clamp value to configured range [minValue, maxValue] (safety check)
			wasClamped := false
			originalValue := newValue
			if newValue < minValue {
				newValue = minValue
				wasClamped = true
			}
			if newValue > maxValue {
				newValue = maxValue
				wasClamped = true
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
					tagRecords++

					// Log every record generated
					clampInfo := ""
					if wasClamped {
						clampInfo = fmt.Sprintf(" [CLAMPED: %.2f -> %.2f]", originalValue, newValue)
					}
					fmt.Printf("[GENERATE] INSERT tag=%s timestamp=%s value=%.2f quality=%d%s\n",
						tag, currentTime.Format("2006-01-02T15:04:05"), newValue, quality, clampInfo)
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
					tagRecords++

					// Log every record generated
					clampInfo := ""
					if wasClamped {
						clampInfo = fmt.Sprintf(" [CLAMPED: %.2f -> %.2f]", originalValue, newValue)
					}
					fmt.Printf("[GENERATE] UPDATE tag=%s timestamp=%s value=%.2f quality=%d%s\n",
						tag, currentTime.Format("2006-01-02T15:04:05"), newValue, quality, clampInfo)
				} else {
					// Log skipped record (lower quality)
					fmt.Printf("[GENERATE] SKIP tag=%s timestamp=%s (existing quality %d >= new quality %d)\n",
						tag, currentTime.Format("2006-01-02T15:04:05"), existingQuality, quality)
				}
			}

			// Update current value for next iteration (only for sequential mode)
			if useSequential {
				currentValue = newValue
			}

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

		// Log completion for each tag
		tagDuration := time.Since(tagStartTime)
		fmt.Printf("[GENERATE] Completed tag: %s (%d records, took %v)\n", tag, tagRecords, tagDuration.Round(time.Millisecond))
	}

	// Final commit
	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit final transaction: %w", err)
	}

	totalDuration := time.Since(generateStartTime)
	fmt.Printf("[GENERATE] Generation completed: %d total records for %d tags (total time: %v)\n",
		totalRecords, len(tags), totalDuration.Round(time.Second))

	return totalRecords, len(tags), nil
}
