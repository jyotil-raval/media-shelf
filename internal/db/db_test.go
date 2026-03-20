package db

import (
	"context"
	"testing"

	"github.com/jyotil-raval/media-shelf/internal/models"
)

func TestMockStore_Add(t *testing.T) {
	tests := []struct {
		name    string
		item    models.MediaItem
		wantErr error
	}{
		{
			name: "valid anime",
			item: models.MediaItem{
				Title:     "Death Note",
				MediaType: "anime",
				SubType:   "tv",
				Source:    "mal",
				SourceID:  "1535",
				Status:    "watching",
			},
			wantErr: nil,
		},
		{
			name: "duplicate source entry",
			item: models.MediaItem{
				Title:     "Death Note",
				MediaType: "anime",
				SubType:   "tv",
				Source:    "mal",
				SourceID:  "1535",
				Status:    "watching",
			},
			wantErr: ErrDuplicate,
		},
		{
			name: "different source ID — not a duplicate",
			item: models.MediaItem{
				Title:     "AoT",
				MediaType: "anime",
				SubType:   "tv",
				Source:    "mal",
				SourceID:  "16498",
				Status:    "completed",
			},
			wantErr: nil,
		},
	}

	store := NewMockStore()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.Add(ctx, tt.item)
			if err != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestMockStore_GetByID(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	id, _ := store.Add(ctx, models.MediaItem{
		Title: "Death Note", Source: "mal", SourceID: "1535", Status: "watching",
	})

	tests := []struct {
		name    string
		id      int64
		wantErr error
	}{
		{"existing id", id, nil},
		{"non-existing id", 9999, ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.GetByID(ctx, tt.id)
			if err != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestMockStore_List(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	store.Add(ctx, models.MediaItem{Title: "Death Note", Source: "mal", SourceID: "1535", Status: "watching", SubType: "tv"})
	store.Add(ctx, models.MediaItem{Title: "AoT", Source: "mal", SourceID: "16498", Status: "completed", SubType: "tv"})

	tests := []struct {
		name      string
		filter    Filter
		wantCount int
	}{
		{"no filter", Filter{}, 2},
		{"filter by status watching", Filter{Status: "watching"}, 1},
		{"filter by status completed", Filter{Status: "completed"}, 1},
		{"filter by non-existent status", Filter{Status: "dropped"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := store.List(ctx, tt.filter)
			if err != nil {
				t.Fatalf("List() unexpected error: %v", err)
			}
			if len(items) != tt.wantCount {
				t.Errorf("List() got %d items, want %d", len(items), tt.wantCount)
			}
		})
	}
}

func TestMockStore_Delete(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	id, _ := store.Add(ctx, models.MediaItem{
		Title: "Death Note", Source: "mal", SourceID: "1535", Status: "watching",
	})

	tests := []struct {
		name    string
		id      int64
		wantErr error
	}{
		{"existing id", id, nil},
		{"non-existing id", 9999, ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Delete(ctx, tt.id)
			if err != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
