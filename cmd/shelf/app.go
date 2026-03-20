package shelf

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

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
	stats, err := a.store.Stats(ctx)
	if err != nil {
		return fmt.Errorf("fetching stats: %w", err)
	}

	if len(stats) == 0 {
		fmt.Println("Your shelf is empty.")
		return nil
	}

	fmt.Printf("%-12s %-14s %s\n", "Type", "Status", "Count")
	fmt.Println("----------------------------------------")

	total := 0
	for _, row := range stats {
		fmt.Printf("%-12s %-14s %d\n", row.SubType, row.Status, row.Count)
		total += row.Count
	}

	fmt.Printf("----------------------------------------\n")
	fmt.Printf("%-12s %-14s %d\n", "", "Total", total)
	return nil
}

func (a *App) Export(ctx context.Context, format, output string) error {
	items, err := a.store.List(ctx, db.Filter{})
	if err != nil {
		return fmt.Errorf("fetching items: %w", err)
	}

	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	switch format {
	case "json":
		enc := json.NewEncoder(file)
		enc.SetIndent("", "  ")
		if err := enc.Encode(items); err != nil {
			return fmt.Errorf("encoding json: %w", err)
		}
	case "csv":
		w := csv.NewWriter(file)
		defer w.Flush()

		// header row
		w.Write([]string{"id", "title", "media_type", "sub_type", "source", "status", "score", "progress", "total"})

		for _, item := range items {
			w.Write([]string{
				strconv.FormatInt(item.ID, 10),
				item.Title,
				item.MediaType,
				item.SubType,
				item.Source,
				item.Status,
				strconv.Itoa(item.Score),
				strconv.Itoa(item.Progress),
				strconv.Itoa(item.Total),
			})
		}
	default:
		return fmt.Errorf("unsupported format: %s (use json or csv)", format)
	}

	fmt.Printf("✓ Exported %d items to %s\n", len(items), output)
	return nil
}
