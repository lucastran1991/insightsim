package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"insightsim/internal/services"
)

// UploadHandler handles POST /api/upload-csv (multipart: file, mode).
type UploadHandler struct {
	uploadService *services.UploadService
}

// NewUploadHandler creates a new UploadHandler.
func NewUploadHandler(uploadService *services.UploadService) *UploadHandler {
	return &UploadHandler{uploadService: uploadService}
}

// UploadResponse is the JSON response for upload-csv.
type UploadResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	Count        int    `json:"count,omitempty"`
	TagsAffected int    `json:"tags_affected,omitempty"`
}

// Handle handles the upload request.
func (h *UploadHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit body size (e.g. 50MB)
	r.ParseMultipartForm(50 << 20)

	file, _, err := r.FormFile("file")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(UploadResponse{
			Success: false,
			Message: "missing or invalid file: " + err.Error(),
		})
		return
	}
	defer file.Close()

	modeStr := strings.TrimSpace(strings.ToLower(r.FormValue("mode")))
	if modeStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(UploadResponse{
			Success: false,
			Message: "missing mode (required: override or replace)",
		})
		return
	}
	var mode services.ImportMode
	switch modeStr {
	case "override":
		mode = services.ImportModeOverride
	case "replace":
		mode = services.ImportModeReplace
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(UploadResponse{
			Success: false,
			Message: "invalid mode (must be override or replace)",
		})
		return
	}

	result, err := h.uploadService.ImportFromCSV(file, mode)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(UploadResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(UploadResponse{
		Success:      true,
		Message:      "CSV imported successfully",
		Count:        result.Count,
		TagsAffected: result.TagsAffected,
	})
}
