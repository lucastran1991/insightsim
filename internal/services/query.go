package services

import (
	"fmt"
	"time"

	"insightsim/internal/database"
	"insightsim/internal/models"
)

// QueryService handles querying timeseries data from the database
type QueryService struct {
	db *database.DB
}

// NewQueryService creates a new QueryService instance
func NewQueryService(db *database.DB) *QueryService {
	return &QueryService{db: db}
}

// QueryTimeseriesData queries data by date range and tags
func (q *QueryService) QueryTimeseriesData(startTime, endTime string, tags []string) (*models.JSONOutput, error) {
	// Parse start and end timestamps
	startTimestamp, err := parseTimestampToMillis(startTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start time: %w", err)
	}

	endTimestamp, err := parseTimestampToMillis(endTime)
	if err != nil {
		return nil, fmt.Errorf("invalid end time: %w", err)
	}

	if startTimestamp > endTimestamp {
		return nil, fmt.Errorf("start time must be before or equal to end time")
	}

	conn := q.db.GetConn()

	// Build query
	query := `
		SELECT tag, timestamp, value, quality
		FROM insight_raws
		WHERE timestamp >= ? AND timestamp <= ?
	`

	args := []interface{}{startTimestamp, endTimestamp}

	if len(tags) > 0 {
		query += " AND tag IN ("
		for i, tag := range tags {
			if i > 0 {
				query += ","
			}
			query += "?"
			args = append(args, tag)
		}
		query += ")"
	}

	query += " ORDER BY tag, timestamp"

	rows, err := conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()

	// Group results by tag
	result := make(map[string][]models.DataPoint)

	for rows.Next() {
		var tag string
		var timestamp int64
		var value float64
		var quality int

		if err := rows.Scan(&tag, &timestamp, &value, &quality); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert timestamp from milliseconds to ISO string
		isoTime := formatTimestamp(timestamp)

		dataPoint := models.DataPoint{
			Timestamp: isoTime,
			Value:     value,
			Quality:   quality,
		}

		result[tag] = append(result[tag], dataPoint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return &models.JSONOutput{Result: result}, nil
}

// parseTimestampToMillis converts ISO 8601 timestamp string to Unix milliseconds
func parseTimestampToMillis(isoTime string) (int64, error) {
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

// formatTimestamp converts Unix milliseconds to ISO 8601 string
func formatTimestamp(millis int64) string {
	t := time.UnixMilli(millis)
	return t.Format("2006-01-02T15:04:05")
}
