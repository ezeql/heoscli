package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/ezeql/heoscli/internal/heos"
)

func newQueueCmd(flags *rootFlags) *cobra.Command {
	var maxItems int
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "List the current play queue",
		Long:  "Lists tracks in the player's queue (including SoundCloud, Tidal, and other sources).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTarget(flags); err != nil {
				return err
			}
			ctx := cmd.Context()
			c, pid, err := resolvePlayerID(ctx, flags)
			if err != nil {
				return err
			}
			defer func() { _ = c.Disconnect() }()

			if maxItems <= 0 {
				maxItems = 200
			}
			items, err := c.GetQueue(ctx, pid, 0, maxItems)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				if isJSON(flags) {
					return writeJSON(cmd, []heos.QueueItem{})
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Queue is empty.")
				return nil
			}
			if isJSON(flags) {
				return writeJSON(cmd, items)
			}
			if isTSV(flags) {
				for i, q := range items {
					title := q.Name
					if title == "" {
						title = q.Song
					}
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%d\t%d\t%s\t%s\t%s\t%s\t%s\n", i+1, q.QueueID, title, q.Artist, q.Album, sourceName(q.SourceID), q.Quality)
				}
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "#\tQID\tTITLE\tARTIST\tALBUM\tSOURCE\tQUALITY")
			for i, q := range items {
				title := q.Name
				if title == "" {
					title = q.Song
				}
				_, _ = fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%s\t%s\t%s\n", i+1, q.QueueID, title, q.Artist, q.Album, sourceName(q.SourceID), q.Quality)
			}
			return w.Flush()
		},
	}
	cmd.Flags().IntVar(&maxItems, "max", 200, "Maximum number of queue items to return")
	return cmd
}

func sourceName(sid int) string {
	switch sid {
	case heos.SourceIDSoundCloud:
		return "SoundCloud"
	case heos.SourceIDTidal:
		return "Tidal"
	default:
		if sid != 0 {
			return fmt.Sprintf("sid:%d", sid)
		}
		return ""
	}
}
