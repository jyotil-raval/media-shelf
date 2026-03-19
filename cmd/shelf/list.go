package shelf

import (
	"github.com/jyotil-raval/media-shelf/internal/db"
	"github.com/spf13/cobra"
)

func newListCommand(app *App) *cobra.Command {
	var status, mediaType, subType, sort string
	var minScore int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List anime on your shelf",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.List(cmd.Context(), db.Filter{
				Status:    status,
				MediaType: mediaType,
				SubType:   subType,
				MinScore:  minScore,
				Sort:      sort,
			})
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().StringVar(&mediaType, "type", "", "Filter by media type")
	cmd.Flags().StringVar(&subType, "subtype", "", "Filter by subtype: tv|movie|ova|special")
	cmd.Flags().IntVar(&minScore, "score", 0, "Minimum score filter")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort by: title|score|updated_at")

	return cmd
}
