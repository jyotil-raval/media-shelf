// internal/db/postgres.go
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jyotil-raval/media-shelf/internal/models"
	"github.com/lib/pq"
)

type PostgreSQLStore struct {
	db *sql.DB
}

func NewPostgreSQLStore(db *sql.DB) *PostgreSQLStore {
	return &PostgreSQLStore{db: db}
}

// Compile-time check — fails to build if PostgreSQLStore stops satisfying Store
var _ Store = (*PostgreSQLStore)(nil)

func (s *PostgreSQLStore) Add(ctx context.Context, item models.MediaItem) (int64, error) {
	query := `
		INSERT INTO media_items
			(title, media_type, sub_type, source, source_id, status, score, progress, total, notes)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	var id int64
	err := s.db.QueryRowContext(ctx, query,
		item.Title, item.MediaType, item.SubType,
		item.Source, item.SourceID, item.Status,
		item.Score, item.Progress, item.Total, item.Notes,
	).Scan(&id)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return 0, fmt.Errorf("add item: %w", ErrDuplicate)
		}
		return 0, fmt.Errorf("add item: %w", err)
	}

	return id, nil
}

func (s *PostgreSQLStore) GetByID(ctx context.Context, id int64) (*models.MediaItem, error) {
	query := `
		SELECT id, title, media_type, sub_type, source, source_id,
		       status, score, progress, total, notes
		FROM media_items
		WHERE id = $1`

	var item models.MediaItem
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&item.ID, &item.Title, &item.MediaType, &item.SubType,
		&item.Source, &item.SourceID, &item.Status,
		&item.Score, &item.Progress, &item.Total, &item.Notes,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get by id: %w", err)
	}

	return &item, nil
}

func (s *PostgreSQLStore) List(ctx context.Context, filter Filter) ([]models.MediaItem, error) {
	query := `
		SELECT id, title, media_type, sub_type, source, source_id,
		       status, score, progress, total, notes
		FROM media_items`

	var conditions []string
	var args []any
	argIdx := 1

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.MediaType != "" {
		conditions = append(conditions, fmt.Sprintf("media_type = $%d", argIdx))
		args = append(args, filter.MediaType)
		argIdx++
	}
	if filter.SubType != "" {
		conditions = append(conditions, fmt.Sprintf("sub_type = $%d", argIdx))
		args = append(args, filter.SubType)
		argIdx++
	}
	if filter.MinScore > 0 {
		conditions = append(conditions, fmt.Sprintf("score >= $%d", argIdx))
		args = append(args, filter.MinScore)
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	switch filter.Sort {
	case "title":
		query += " ORDER BY title ASC"
	case "score":
		query += " ORDER BY score DESC"
	case "updated_at":
		query += " ORDER BY updated_at DESC"
	default:
		query += " ORDER BY created_at DESC"
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	var items []models.MediaItem
	for rows.Next() {
		var item models.MediaItem
		if err := rows.Scan(
			&item.ID, &item.Title, &item.MediaType, &item.SubType,
			&item.Source, &item.SourceID, &item.Status,
			&item.Score, &item.Progress, &item.Total, &item.Notes,
		); err != nil {
			return nil, fmt.Errorf("scanning item: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *PostgreSQLStore) Update(ctx context.Context, item models.MediaItem) error {
	query := `
		UPDATE media_items
		SET title=$1, status=$2, score=$3, progress=$4, total=$5,
		    notes=$6, updated_at=NOW()
		WHERE id=$7`

	result, err := s.db.ExecContext(ctx, query,
		item.Title, item.Status, item.Score,
		item.Progress, item.Total, item.Notes, item.ID,
	)
	if err != nil {
		return fmt.Errorf("update item: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update item: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *PostgreSQLStore) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM media_items WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *PostgreSQLStore) Stats(ctx context.Context) ([]StatRow, error) {
	query := `
		SELECT COALESCE(sub_type, 'unknown'), status, COUNT(*)
		FROM media_items
		GROUP BY sub_type, status
		ORDER BY sub_type, status`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("stats query: %w", err)
	}
	defer rows.Close()

	var stats []StatRow
	for rows.Next() {
		var row StatRow
		if err := rows.Scan(&row.SubType, &row.Status, &row.Count); err != nil {
			return nil, fmt.Errorf("scanning stat row: %w", err)
		}
		stats = append(stats, row)
	}

	return stats, rows.Err()
}
