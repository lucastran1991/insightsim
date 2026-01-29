package models

// TagList represents the tag list JSON structure
type TagList struct {
	TagList string            `json:"tag_list"`          // Comma-separated list of tags (preferred)
	Tag     string            `json:"tag"`               // Alternative key name (for backward compatibility)
	Created map[string]string `json:"created,omitempty"` // Optional: tag name -> ISO created_at
}

// GetTagList returns the tag list string, checking both possible JSON keys
func (t *TagList) GetTagList() string {
	if t.TagList != "" {
		return t.TagList
	}
	return t.Tag
}
