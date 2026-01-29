package services

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"insightsim/internal/database"
)

const importQuality = 3

// ImportMode is override (upsert within CSV) or replace (delete tag then insert).
type ImportMode string

const (
	ImportModeOverride ImportMode = "override"
	ImportModeReplace  ImportMode = "replace"
)

// UploadService handles CSV import into insight_raws.
type UploadService struct {
	db *database.DB
}

// NewUploadService creates a new UploadService.
func NewUploadService(db *database.DB) *UploadService {
	return &UploadService{db: db}
}

// ImportResult holds count and tags affected after import.
type ImportResult struct {
	Count        int
	TagsAffected int
}

// ImportFromCSV parses CSV from reader and imports per mode. CSV format: header "timestamp",<tag1>,<tag2>,...; rows: timestamp,value1,value2,...
func (u *UploadService) ImportFromCSV(reader io.Reader, mode ImportMode) (*ImportResult, error) {
	r := csv.NewReader(reader)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("CSV is empty")
	}
	header := records[0]
	if len(header) < 2 {
		return nil, fmt.Errorf("CSV must have timestamp column and at least one tag column")
	}
	if strings.TrimSpace(strings.ToLower(header[0])) != "timestamp" {
		return nil, fmt.Errorf("first column must be 'timestamp', got %q", header[0])
	}
	tags := make([]string, 0, len(header)-1)
	for i := 1; i < len(header); i++ {
		tag := strings.TrimSpace(header[i])
		if tag == "" {
			return nil, fmt.Errorf("empty tag name in column %d", i+1)
		}
		tags = append(tags, tag)
	}
	// data rows
	rows := records[1:]
	if len(rows) == 0 {
		// No data rows: still ensure every header tag exists in tags table
		now := time.Now().UTC().Format(time.RFC3339)
		for _, tag := range tags {
			if err := u.db.InsertTagIfNotExists(tag, now, now, "upload"); err != nil {
				return nil, fmt.Errorf("failed to register tag %s: %w", tag, err)
			}
		}
		return &ImportResult{Count: 0, TagsAffected: len(tags)}, nil
	}

	conn := u.db.GetConn()
	tx, err := conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if mode == ImportModeReplace {
		for _, tag := range tags {
			_, err = tx.Exec("DELETE FROM insight_raws WHERE tag = ?", tag)
			if err != nil {
				return nil, fmt.Errorf("failed to delete records for tag %s: %w", tag, err)
			}
		}
	}

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO insight_raws (tag, timestamp, value, quality)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insert: %w", err)
	}
	defer stmt.Close()

	totalCount := 0
	for rowIdx, row := range rows {
		if len(row) == 0 || allEmpty(row) {
			continue
		}
		if len(row) != len(header) {
			return nil, fmt.Errorf("row %d: expected %d columns, got %d", rowIdx+2, len(header), len(row))
		}
		tsStr := strings.TrimSpace(row[0])
		if tsStr == "" {
			return nil, fmt.Errorf("row %d: empty timestamp", rowIdx+2)
		}
		tsMs, err := parseTimestampCSV(tsStr)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowIdx+2, err)
		}
		for i, tag := range tags {
			valStr := ""
			if i+1 < len(row) {
				valStr = strings.TrimSpace(row[i+1])
			}
			var value float64
			if valStr != "" {
				value, err = strconv.ParseFloat(valStr, 64)
				if err != nil {
					return nil, fmt.Errorf("row %d column %s: invalid number %q: %w", rowIdx+2, tag, valStr, err)
				}
			}
			_, err = stmt.Exec(tag, tsMs, value, importQuality)
			if err != nil {
				return nil, fmt.Errorf("failed to insert row: %w", err)
			}
			totalCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	// Ensure every CSV tag exists in tags table (create if not existed)
	now := time.Now().UTC().Format(time.RFC3339)
	for _, tag := range tags {
		if err := u.db.InsertTagIfNotExists(tag, now, now, "upload"); err != nil {
			return nil, fmt.Errorf("failed to register tag %s: %w", tag, err)
		}
	}

	return &ImportResult{
		Count:        totalCount,
		TagsAffected: len(tags),
	}, nil
}

func allEmpty(ss []string) bool {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return false
		}
	}
	return true
}

func parseTimestampCSV(isoTime string) (int64, error) {
	formats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05Z",
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05.000Z",
	}
	for _, f := range formats {
		t, err := time.Parse(f, isoTime)
		if err == nil {
			return t.UnixMilli(), nil
		}
	}
	return 0, fmt.Errorf("unable to parse timestamp: %s", isoTime)
}
