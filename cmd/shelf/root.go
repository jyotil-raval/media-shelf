// cmd/shelf/root.go
package shelf

import (
	"github.com/spf13/cobra"
)

func NewRootCommand(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:   "shelf",
		Short: "Track your anime shelf",
		Long:  "A local CLI tool to track anime — powered by MAL and PostgreSQL.",
	}

	root.AddCommand(newAddCommand(app))
	root.AddCommand(newListCommand(app))
	root.AddCommand(newStatsCommand(app))
	root.AddCommand(newExportCommand(app))

	return root
}
