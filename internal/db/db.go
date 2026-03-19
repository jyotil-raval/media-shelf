// internal/db/db.go
package db

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

// Sentinel errors — compared with errors.Is(), never with ==
var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("duplicate entry")
)

func Open(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connecting to db: %w", err)
	}

	return db, nil
}
