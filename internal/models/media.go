package models

type MediaItem struct {
	ID        int64  `json:"id"         db:"id"`
	Title     string `json:"title"      db:"title"`
	MediaType string `json:"media_type" db:"media_type"` // always "anime"
	SubType   string `json:"sub_type"   db:"sub_type"`   // tv | movie | ova | special
	Source    string `json:"source"     db:"source"`     // always "mal"
	SourceID  string `json:"source_id"  db:"source_id"`
	Status    string `json:"status"     db:"status"` // watching | completed | on_hold | dropped | plan_to
	Score     int    `json:"score"      db:"score"`
	Progress  int    `json:"progress"   db:"progress"`
	Total     int    `json:"total"      db:"total"`
	Notes     string `json:"notes"      db:"notes"`
}
