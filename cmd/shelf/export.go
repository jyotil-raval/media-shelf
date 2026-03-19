package shelf

import (
	"github.com/spf13/cobra"
)

func newExportCommand(app *App) *cobra.Command {
	var format, output string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export your shelf to a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Export(cmd.Context(), format, output)
		},
	}

	cmd.Flags().StringVar(&format, "format", "json", "Export format: json|csv")
	cmd.Flags().StringVar(&output, "output", "shelf.json", "Output file path")

	return cmd
}
