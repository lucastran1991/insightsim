package services

import (
	"fmt"
	"strconv"
	"strings"
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

// QueryTimeseriesData queries data by date range, tags, and optional aggregation
func (q *QueryService) QueryTimeseriesData(startTime, endTime string, tags []string, aggregate string) (*models.JSONOutput, error) {
	queryStartTime := time.Now()

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

	// Log query parameters
	tagsInfo := "all tags"
	if len(tags) > 0 {
		if len(tags) <= 3 {
			tagsInfo = strings.Join(tags, ", ")
		} else {
			tagsInfo = fmt.Sprintf("%d tags (%s, ...)", len(tags), strings.Join(tags[:3], ", "))
		}
	}
	fmt.Printf("[QUERY] Querying data: time range %s to %s, tags: %s, aggregate: %s\n", startTime, endTime, tagsInfo, aggregate)

	if aggregate == "" || aggregate == "raw" {
		return q.queryRaw(q.db, startTimestamp, endTimestamp, tags, queryStartTime)
	}
	return q.queryAggregated(q.db, startTimestamp, endTimestamp, tags, aggregate, queryStartTime)
}

// queryRaw returns raw rows (no aggregation)
func (q *QueryService) queryRaw(conn *database.DB, startTimestamp, endTimestamp int64, tags []string, queryStartTime time.Time) (*models.JSONOutput, error) {
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

	rows, err := conn.GetConn().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]models.DataPoint)
	for rows.Next() {
		var tag string
		var timestamp int64
		var value float64
		var quality int
		if err := rows.Scan(&tag, &timestamp, &value, &quality); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		isoTime := formatTimestamp(timestamp)
		result[tag] = append(result[tag], models.DataPoint{Timestamp: isoTime, Value: value, Quality: quality})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	totalRecords := 0
	for _, dataPoints := range result {
		totalRecords += len(dataPoints)
	}
	fmt.Printf("[QUERY] Query completed: %d records from %d tags (took %v)\n",
		totalRecords, len(result), time.Since(queryStartTime).Round(time.Millisecond))
	return &models.JSONOutput{Result: result}, nil
}

// queryAggregated runs GROUP BY with date bucket and SUM(value), MAX(quality)
func (q *QueryService) queryAggregated(conn *database.DB, startTimestamp, endTimestamp int64, tags []string, aggregate string, queryStartTime time.Time) (*models.JSONOutput, error) {
	// SQLite: bucket expression and ORDER BY bucket
	// timestamp is stored in milliseconds
	bucketExpr := bucketExpression(aggregate)
	if bucketExpr == "" {
		return q.queryRaw(conn, startTimestamp, endTimestamp, tags, queryStartTime)
	}

	query := fmt.Sprintf(`
		SELECT tag, %s AS bucket, SUM(value) AS value, MAX(quality) AS quality
		FROM insight_raws
		WHERE timestamp >= ? AND timestamp <= ?
	`, bucketExpr)
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
	query += " GROUP BY tag, bucket ORDER BY tag, bucket"

	rows, err := conn.GetConn().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]models.DataPoint)
	for rows.Next() {
		var tag, bucket string
		var value float64
		var quality int
		if err := rows.Scan(&tag, &bucket, &value, &quality); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		isoTime := bucketToISO(bucket, aggregate)
		result[tag] = append(result[tag], models.DataPoint{Timestamp: isoTime, Value: value, Quality: quality})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	totalRecords := 0
	for _, dataPoints := range result {
		totalRecords += len(dataPoints)
	}
	fmt.Printf("[QUERY] Query completed: %d records from %d tags (took %v)\n",
		totalRecords, len(result), time.Since(queryStartTime).Round(time.Millisecond))
	return &models.JSONOutput{Result: result}, nil
}

// bucketExpression returns SQLite expression for grouping (empty = raw)
func bucketExpression(aggregate string) string {
	// datetime(timestamp/1000, 'unixepoch') is UTC in SQLite
	dt := "datetime(timestamp/1000, 'unixepoch')"
	switch aggregate {
	case "daily":
		return fmt.Sprintf("strftime('%%Y-%%m-%%d', %s)", dt)
	case "monthly":
		return fmt.Sprintf("strftime('%%Y-%%m', %s)", dt)
	case "quarterly":
		return fmt.Sprintf("strftime('%%Y', %s) || '-' || ((cast(strftime('%%m', %s) as integer) - 1) / 3 + 1)", dt, dt)
	case "yearly":
		return fmt.Sprintf("strftime('%%Y', %s)", dt)
	default:
		return ""
	}
}

// bucketToISO converts SQL bucket string to ISO 8601 timestamp (bucket start)
func bucketToISO(bucket, aggregate string) string {
	switch aggregate {
	case "daily":
		// bucket e.g. "2026-01-29"
		return bucket + "T00:00:00"
	case "monthly":
		// bucket e.g. "2026-01"
		return bucket + "-01T00:00:00"
	case "quarterly":
		// bucket e.g. "2026-1", "2026-2", "2026-3", "2026-4"
		parts := strings.SplitN(bucket, "-", 2)
		if len(parts) != 2 {
			return bucket + "T00:00:00"
		}
		year, _ := strconv.Atoi(parts[0])
		q, _ := strconv.Atoi(parts[1])
		if q < 1 || q > 4 {
			return bucket + "T00:00:00"
		}
		month := (q-1)*3 + 1
		return fmt.Sprintf("%04d-%02d-01T00:00:00", year, month)
	case "yearly":
		// bucket e.g. "2026"
		return bucket + "-01-01T00:00:00"
	default:
		return bucket + "T00:00:00"
	}
}

// parseTimestampToMillis converts ISO 8601 timestamp string to Unix milliseconds
func parseTimestampToMillis(isoTime string) (int64, error) {
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

	for _, format := range formats {
		t, err := time.Parse(format, isoTime)
		if err == nil {
			return t.UnixMilli(), nil
		}
	}

	return 0, fmt.Errorf("unable to parse timestamp: %s", isoTime)
}

// formatTimestamp converts Unix milliseconds to ISO 8601 string (UTC)
func formatTimestamp(millis int64) string {
	t := time.UnixMilli(millis).UTC()
	return t.Format("2006-01-02T15:04:05")
}
