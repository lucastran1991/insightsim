package models

// InsightRaw represents a single timeseries data point
type InsightRaw struct {
	ID        int64   `json:"id"`
	Tag       string  `json:"tag"`
	Timestamp int64   `json:"timestamp"` // Unix timestamp in milliseconds
	Value     float64 `json:"value"`
	Quality   int     `json:"quality"`
}

// JSONInput represents the input JSON structure from the feed
type JSONInput struct {
	Result map[string][]DataPoint `json:"result"`
}

// DataPoint represents a single data point in the JSON input
type DataPoint struct {
	Timestamp string  `json:"timestamp"` // ISO 8601 format: "2025-01-01T23:59:12"
	Value     float64 `json:"value"`
	Quality   int     `json:"quality"`
}

// JSONOutput represents the output JSON structure for API responses
type JSONOutput struct {
	Result map[string][]DataPoint `json:"result"`
}
