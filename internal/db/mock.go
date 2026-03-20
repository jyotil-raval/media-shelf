package db

import (
	"context"
	"sync"

	"github.com/jyotil-raval/media-shelf/internal/models"
)

type MockStore struct {
	mu    sync.Mutex
	items map[int64]models.MediaItem
	next  int64
}

func NewMockStore() *MockStore {
	return &MockStore{
		items: make(map[int64]models.MediaItem),
		next:  1,
	}
}

func (m *MockStore) Add(ctx context.Context, item models.MediaItem) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.items {
		if existing.Source == item.Source && existing.SourceID == item.SourceID {
			return 0, ErrDuplicate
		}
	}

	item.ID = m.next
	m.items[m.next] = item
	m.next++
	return item.ID, nil
}

func (m *MockStore) GetByID(ctx context.Context, id int64) (*models.MediaItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &item, nil
}

func (m *MockStore) List(ctx context.Context, filter Filter) ([]models.MediaItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []models.MediaItem
	for _, item := range m.items {
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.SubType != "" && item.SubType != filter.SubType {
			continue
		}
		if filter.MinScore > 0 && item.Score < filter.MinScore {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func (m *MockStore) Update(ctx context.Context, item models.MediaItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.items[item.ID]; !ok {
		return ErrNotFound
	}
	m.items[item.ID] = item
	return nil
}

func (m *MockStore) Delete(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.items[id]; !ok {
		return ErrNotFound
	}
	delete(m.items, id)
	return nil
}

func (m *MockStore) Stats(ctx context.Context) ([]StatRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	counts := make(map[string]int)
	for _, item := range m.items {
		key := item.SubType + "|" + item.Status
		counts[key]++
	}

	var result []StatRow
	for key, count := range counts {
		parts := splitKey(key)
		result = append(result, StatRow{
			SubType: parts[0],
			Status:  parts[1],
			Count:   count,
		})
	}
	return result, nil
}

func splitKey(key string) []string {
	for i, c := range key {
		if c == '|' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{key, ""}
}
