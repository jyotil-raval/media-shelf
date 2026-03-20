package shelf

import (
	"context"
	"fmt"

	"github.com/jyotil-raval/media-shelf/internal/db"
	"github.com/jyotil-raval/media-shelf/internal/providers/mal"
)

type App struct {
	store     db.Store
	malClient *mal.Client
}

func NewApp(store db.Store, malClient *mal.Client) *App {
	return &App{store: store, malClient: malClient}
}

func (a *App) Add(ctx context.Context, source, id, status string) error {
	item, err := a.malClient.GetAnime(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching anime: %w", err)
	}

	item.Status = status

	newID, err := a.store.Add(ctx, *item)
	if err != nil {
		return fmt.Errorf("storing anime: %w", err)
	}

	fmt.Printf("✓ Added [%d]: %s (%s)\n", newID, item.Title, item.SubType)
	return nil
}

func (a *App) List(ctx context.Context, filter db.Filter) error {
	items, err := a.store.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("listing anime: %w", err)
	}

	if len(items) == 0 {
		fmt.Println("Your shelf is empty.")
		return nil
	}

	fmt.Printf("%-6s %-40s %-10s %-10s %-6s\n", "ID", "Title", "Type", "Status", "Score")
	fmt.Println("----------------------------------------------------------------------")
	for _, item := range items {
		fmt.Printf("%-6d %-40s %-10s %-10s %-6d\n",
			item.ID, item.Title, item.SubType, item.Status, item.Score)
	}
	fmt.Printf("\nTotal: %d\n", len(items))
	return nil
}

func (a *App) Stats(ctx context.Context) error {
	fmt.Println("TODO: stats")
	return nil
}

func (a *App) Export(ctx context.Context, format, output string) error {
	fmt.Printf("TODO: export as %s to %s\n", format, output)
	return nil
}
