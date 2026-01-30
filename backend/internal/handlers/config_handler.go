package handlers

import (
	"encoding/json"
	"net/http"
)

// ConfigHandler handles GET /api/config to expose server config (e.g. value_range for frontend defaults).
type ConfigHandler struct {
	minValue float64
	maxValue float64
}

// NewConfigHandler creates a new ConfigHandler with value range from config.
func NewConfigHandler(minValue, maxValue float64) *ConfigHandler {
	return &ConfigHandler{minValue: minValue, maxValue: maxValue}
}

// ValueRangeResponse is the JSON shape for value_range.
type ValueRangeResponse struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// ConfigResponse is the JSON response for GET /api/config.
type ConfigResponse struct {
	ValueRange ValueRangeResponse `json:"value_range"`
}

// Handle responds with config suitable for frontend (e.g. value_range defaults).
func (h *ConfigHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	resp := ConfigResponse{
		ValueRange: ValueRangeResponse{Min: h.minValue, Max: h.maxValue},
	}
	_ = json.NewEncoder(w).Encode(resp)
}
