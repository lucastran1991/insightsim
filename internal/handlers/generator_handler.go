package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"insightsim/internal/services"
)

// GeneratorHandler handles POST /api/generate-dummy requests
type GeneratorHandler struct {
	generator       *services.Generator
	tagListFile     string
	minValue        float64
	maxValue        float64
	useSequential   bool
	startTime       string
	endTime         string
}

// NewGeneratorHandler creates a new GeneratorHandler instance
func NewGeneratorHandler(generator *services.Generator, tagListFile string, minValue, maxValue float64, useSequential bool, startTime, endTime string) *GeneratorHandler {
	return &GeneratorHandler{
		generator:     generator,
		tagListFile:   tagListFile,
		minValue:      minValue,
		maxValue:      maxValue,
		useSequential: useSequential,
		startTime:     startTime,
		endTime:       endTime,
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

	mode := "random"
	if h.useSequential {
		mode = "sequential (30% restriction)"
	}
	fmt.Printf("[API] POST /api/generate-dummy - Starting generation (value range: %.2f-%.2f, mode: %s, time range: %s to %s)\n", h.minValue, h.maxValue, mode, h.startTime, h.endTime)
	
	count, tagsCount, err := h.generator.GenerateDummyData(h.tagListFile, h.minValue, h.maxValue, h.useSequential, h.startTime, h.endTime)
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
