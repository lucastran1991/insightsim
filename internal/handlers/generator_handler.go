package handlers

import (
	"encoding/json"
	"net/http"

	"insightsim/internal/services"
)

// GeneratorHandler handles POST /api/generate-dummy requests
type GeneratorHandler struct {
	generator    *services.Generator
	tagListFile  string
	minValue     float64
	maxValue     float64
}

// NewGeneratorHandler creates a new GeneratorHandler instance
func NewGeneratorHandler(generator *services.Generator, tagListFile string, minValue, maxValue float64) *GeneratorHandler {
	return &GeneratorHandler{
		generator:   generator,
		tagListFile: tagListFile,
		minValue:    minValue,
		maxValue:    maxValue,
	}
}

// GenerateResponse represents the response from generate-dummy endpoint
type GenerateResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	Count      int    `json:"count,omitempty"`
	TagsCount  int    `json:"tags_count,omitempty"`
}

// Handle handles the generate-dummy request
func (h *GeneratorHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	count, tagsCount, err := h.generator.GenerateDummyData(h.tagListFile, h.minValue, h.maxValue)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(GenerateResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GenerateResponse{
		Success:   true,
		Message:   "Dummy data generated successfully",
		Count:     count,
		TagsCount: tagsCount,
	})
}
