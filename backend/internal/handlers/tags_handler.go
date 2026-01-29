package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"insightsim/internal/services"
)

// TagsHandler handles GET /api/tags, GET /api/tags/names, DELETE /api/tags?tag=..., POST /api/tags
type TagsHandler struct {
	tagsService *services.TagsService
}

// NewTagsHandler creates a new TagsHandler
func NewTagsHandler(tagsService *services.TagsService) *TagsHandler {
	return &TagsHandler{tagsService: tagsService}
}

// CreateTagRequest is the body for POST /api/tags
type CreateTagRequest struct {
	Tag    string `json:"tag"`
	Source string `json:"source"`
}

// TagsListResponse is the paginated response for GET /api/tags
type TagsListResponse struct {
	Items []services.TagInfo `json:"items"`
	Total int                `json:"total"`
}

// TagsNamesResponse is the response for GET /api/tags/names
type TagsNamesResponse struct {
	Tags []string `json:"tags"`
}

// HandleGet returns a page of tags (GET /api/tags?page=1&limit=9)
func (h *TagsHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 9
	}
	if limit > 100 {
		limit = 100
	}
	items, total, err := h.tagsService.ListTagsPaginated(page, limit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TagsListResponse{Items: items, Total: total})
}

// HandleListNames returns all tag names (GET /api/tags/names)
func (h *TagsHandler) HandleListNames(w http.ResponseWriter, r *http.Request) {
	names, err := h.tagsService.ListTagNames()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TagsNamesResponse{Tags: names})
}

// HandleDelete deletes a tag (DELETE /api/tags?tag=...)
func (h *TagsHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	tag := strings.TrimSpace(r.URL.Query().Get("tag"))
	if tag == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing query param: tag"})
		return
	}
	if err := h.tagsService.DeleteTagData(tag); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HandlePost creates a tag (POST /api/tags)
func (h *TagsHandler) HandlePost(w http.ResponseWriter, r *http.Request) {
	var req CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}
	tag := strings.TrimSpace(req.Tag)
	if tag == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "tag is required and cannot be empty"})
		return
	}
	source := strings.TrimSpace(req.Source)
	if err := h.tagsService.CreateTag(tag, source); err != nil {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "constraint") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "tag already exists"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
}
