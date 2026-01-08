package handlers

import (
	"encoding/json"
	"net/http"

	"insightsim/internal/services"
)

// LoadHandler handles POST /api/load requests
type LoadHandler struct {
	loader        *services.Loader
	rawDataFolder string
}

// NewLoadHandler creates a new LoadHandler instance
func NewLoadHandler(loader *services.Loader, rawDataFolder string) *LoadHandler {
	return &LoadHandler{
		loader:        loader,
		rawDataFolder: rawDataFolder,
	}
}

// LoadRequest represents the request body for load endpoint
type LoadRequest struct {
	FilePath string `json:"file_path"`
}

// LoadResponse represents the response from load endpoint
type LoadResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	Count      int    `json:"count,omitempty"`
	FilesCount int    `json:"files_count,omitempty"`
}

// Handle handles the load request
func (h *LoadHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Load all JSON files from configured folder
	count, filesCount, err := h.loader.LoadFromFolder(h.rawDataFolder)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(LoadResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoadResponse{
		Success:    true,
		Message:    "Data loaded successfully",
		Count:      count,
		FilesCount: filesCount,
	})
}
