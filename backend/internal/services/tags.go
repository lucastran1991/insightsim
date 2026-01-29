package services

import (
	"fmt"
	"strings"
	"time"

	"insightsim/internal/database"
)

// TagInfo is the API view of a tag (from tags table)
type TagInfo struct {
	Tag       string `json:"tag"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Source    string `json:"source"`
}

// TagsService handles tag operations (DB only)
type TagsService struct {
	db *database.DB
}

// NewTagsService creates a new TagsService
func NewTagsService(db *database.DB) *TagsService {
	return &TagsService{db: db}
}

// ListTagsPaginated returns a page of tags from the DB and total count. If search is non-empty, filters by tag name (case-insensitive).
func (s *TagsService) ListTagsPaginated(page, limit int, search string) (items []TagInfo, total int, err error) {
	search = strings.TrimSpace(search)
	if limit <= 0 {
		limit = 9
	}
	if page < 1 {
		page = 1
	}
	if search != "" {
		total, err = s.db.CountTagsWithSearch(search)
	} else {
		total, err = s.db.CountTags()
	}
	if err != nil {
		return nil, 0, fmt.Errorf("count tags: %w", err)
	}
	offset := (page - 1) * limit
	if offset >= total {
		return []TagInfo{}, total, nil
	}
	var rows []database.TagRow
	if search != "" {
		rows, err = s.db.ListTagsPaginatedWithSearch(offset, limit, search)
	} else {
		rows, err = s.db.ListTagsPaginated(offset, limit)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("list tags: %w", err)
	}
	items = make([]TagInfo, 0, len(rows))
	for _, r := range rows {
		items = append(items, TagInfo{
			Tag:       r.Tag,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
			Source:    r.Source,
		})
	}
	return items, total, nil
}

// CreateTag inserts a tag into the tags table
func (s *TagsService) CreateTag(tagName string, source string) error {
	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		return fmt.Errorf("tag name is empty")
	}
	if source == "" {
		source = "custom"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return s.db.InsertTag(tagName, now, now, source)
}

// DeleteTagData removes the tag's records from insight_raws and the tag row from tags table
func (s *TagsService) DeleteTagData(tag string) error {
	if err := s.db.DeleteTagRecords(tag); err != nil {
		return err
	}
	return s.db.DeleteTag(tag)
}

// ListTagNames returns all tag names from the tags table (for generator and API)
func (s *TagsService) ListTagNames() ([]string, error) {
	return s.db.ListTagNamesFromTagsTable()
}
