package shelf

import (
	"github.com/spf13/cobra"
)

func newStatsCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show shelf statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Stats(cmd.Context())
		},
	}
}
