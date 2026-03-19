// internal/db/filter.go
package db

// Filter holds the query parameters for List()
type Filter struct {
	Status    string
	MediaType string
	SubType   string
	MinScore  int
	Sort      string // "title" | "score" | "updated_at"
}
