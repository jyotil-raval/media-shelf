// cmd/shelf/app.go
package shelf

import (
	"context"
	"fmt"

	"github.com/jyotil-raval/media-shelf/internal/db"
)

type App struct {
	store db.Store
}

func NewApp(store db.Store) *App {
	return &App{store: store}
}

func (a *App) Add(ctx context.Context, source, id, status string) error {
	fmt.Printf("TODO: add anime %s from %s with status %s\n", id, source, status)
	return nil
}

func (a *App) List(ctx context.Context, filter db.Filter) error {
	fmt.Println("TODO: list anime")
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
