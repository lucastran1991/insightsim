package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"insightsim/internal/services"
)

// Stream event types for NDJSON response
type streamEvent struct {
	Event     string `json:"event"`
	Tag       string `json:"tag,omitempty"`
	Records   int    `json:"records,omitempty"`
	Count     int    `json:"count,omitempty"`
	TagsCount int    `json:"tags_count,omitempty"`
	Message   string `json:"message,omitempty"`
}

// GeneratorHandler handles POST /api/generate-dummy requests
type GeneratorHandler struct {
	generator     *services.Generator
	minValue      float64
	maxValue      float64
	useSequential bool
	startTime     string
	endTime       string
}

// NewGeneratorHandler creates a new GeneratorHandler instance
func NewGeneratorHandler(generator *services.Generator, minValue, maxValue float64, useSequential bool, startTime, endTime string) *GeneratorHandler {
	return &GeneratorHandler{
		generator:     generator,
		minValue:      minValue,
		maxValue:      maxValue,
		useSequential: useSequential,
		startTime:     startTime,
		endTime:       endTime,
	}
}

// GenerateRequest represents the request body for generate-dummy endpoint
type GenerateRequest struct {
	Tag   string `json:"tag,omitempty"`   // Optional: if provided, only generate for this tag
	Start string `json:"start,omitempty"` // Optional: start time (ISO 8601). If set, overrides config.
	End   string `json:"end,omitempty"`   // Optional: end time (ISO 8601). If set, overrides config.
}

// GenerateResponse represents the response from generate-dummy endpoint
type GenerateResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Count     int    `json:"count,omitempty"`
	TagsCount int    `json:"tags_count,omitempty"`
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

	// Use request start/end if provided; otherwise use config defaults
	startTime := h.startTime
	endTime := h.endTime
	if req.Start != "" {
		startTime = req.Start
	}
	if req.End != "" {
		endTime = req.End
	}

	fmt.Printf("[API] POST /api/generate-dummy - Starting generation for %s (value range: %.2f-%.2f, mode: %s, time range: %s to %s)\n",
		tagInfo, h.minValue, h.maxValue, mode, startTime, endTime)

	// Stream NDJSON: set headers and prepare flusher
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)
	var flusher http.Flusher
	if f, ok := w.(http.Flusher); ok {
		flusher = f
	}
	writeEvent := func(ev streamEvent) {
		data, _ := json.Marshal(ev)
		w.Write(append(data, '\n'))
		if flusher != nil {
			flusher.Flush()
		}
	}

	onTagComplete := func(tag string, records int) {
		writeEvent(streamEvent{Event: "tag_complete", Tag: tag, Records: records})
	}

	count, tagsCount, err := h.generator.GenerateDummyData(h.minValue, h.maxValue, h.useSequential, startTime, endTime, req.Tag, onTagComplete)
	if err != nil {
		writeEvent(streamEvent{Event: "error", Message: err.Error()})
		return
	}
	writeEvent(streamEvent{Event: "done", Count: count, TagsCount: tagsCount})
}
