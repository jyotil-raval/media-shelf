// internal/db/store.go
package db

import (
	"context"

	"github.com/jyotil-raval/media-shelf/internal/models"
)

type Store interface {
	Add(ctx context.Context, item models.MediaItem) (int64, error)
	GetByID(ctx context.Context, id int64) (*models.MediaItem, error)
	List(ctx context.Context, filter Filter) ([]models.MediaItem, error)
	Update(ctx context.Context, item models.MediaItem) error
	Delete(ctx context.Context, id int64) error
	Stats(ctx context.Context) ([]StatRow, error)
}
