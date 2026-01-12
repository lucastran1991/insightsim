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

// GenerateRequest represents the request body for generate-dummy endpoint
type GenerateRequest struct {
	Tag string `json:"tag,omitempty"` // Optional: if provided, only generate for this tag
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

	// Parse request body (optional)
	var req GenerateRequest
	if r.Body != nil {
		decoder := json.NewDecoder(r.Body)
		decoder.Decode(&req) // Ignore errors, use empty struct if body is empty or invalid
	}

	mode := "random"
	if h.useSequential {
		mode = "sequential (30% restriction)"
	}

	tagInfo := "all tags"
	if req.Tag != "" {
		tagInfo = fmt.Sprintf("single tag: %s", req.Tag)
	}

	fmt.Printf("[API] POST /api/generate-dummy - Starting generation for %s (value range: %.2f-%.2f, mode: %s, time range: %s to %s)\n",
		tagInfo, h.minValue, h.maxValue, mode, h.startTime, h.endTime)

	count, tagsCount, err := h.generator.GenerateDummyData(h.tagListFile, h.minValue, h.maxValue, h.useSequential, h.startTime, h.endTime, req.Tag)
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
