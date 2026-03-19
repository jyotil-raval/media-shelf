package shelf

import (
	"github.com/spf13/cobra"
)

func newAddCommand(app *App) *cobra.Command {
	var source, id, status string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add an anime to your shelf",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Add(cmd.Context(), source, id, status)
		},
	}

	cmd.Flags().StringVar(&source, "source", "mal", "Data source (mal)")
	cmd.Flags().StringVar(&id, "id", "", "Anime ID from the source")
	cmd.Flags().StringVar(&status, "status", "", "Watch status: watching|completed|on_hold|dropped|plan_to")

	cmd.MarkFlagRequired("id")
	cmd.MarkFlagRequired("status")

	return cmd
}
