package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"insightsim/internal/services"
)

// QueryHandler handles GET /api/timeseriesdata/{start}/{end} requests
type QueryHandler struct {
	queryService *services.QueryService
}

// NewQueryHandler creates a new QueryHandler instance
func NewQueryHandler(queryService *services.QueryService) *QueryHandler {
	return &QueryHandler{queryService: queryService}
}

// Handle handles the query request
func (h *QueryHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/timeseriesdata/{start}/{end}
	path := strings.TrimPrefix(r.URL.Path, "/api/timeseriesdata/")
	// Remove query string from path if present
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	parts := strings.SplitN(path, "/", 2)

	var startTime, endTime string
	if len(parts) >= 2 {
		startTime = parts[0]
		endTime = strings.TrimSuffix(parts[1], "/")
	}

	// If not in path, try query parameters
	if startTime == "" {
		startTime = r.URL.Query().Get("start")
	}
	if endTime == "" {
		endTime = r.URL.Query().Get("end")
	}

	if startTime == "" || endTime == "" {
		http.Error(w, "Invalid path format. Expected: /api/timeseriesdata/{start}/{end} or ?start=...&end=...", http.StatusBadRequest)
		return
	}

	// Parse tags from query parameter
	var tags []string
	tagsParam := r.URL.Query().Get("tags")
	if tagsParam != "" {
		tags = strings.Split(tagsParam, ",")
		// Trim whitespace from each tag
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
	}

	// Log request
	tagsInfo := "all tags"
	if len(tags) > 0 {
		tagsInfo = fmt.Sprintf("%d tag(s)", len(tags))
	}
	fmt.Printf("[API] GET /api/timeseriesdata - Query: %s to %s, tags: %s\n", startTime, endTime, tagsInfo)
	
	// Query the data
	result, err := h.queryService.QueryTimeseriesData(startTime, endTime, tags)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
