package cli

import (
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"heoscli/internal/heos"
)

func newSearchCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search Tidal or SoundCloud",
	}
	cmd.AddCommand(newSearchTidalCmd(flags))
	cmd.AddCommand(newSearchSoundCloudCmd(flags))
	return cmd
}

func newSearchTidalCmd(flags *rootFlags) *cobra.Command {
	return newSearchServiceCmd(flags, "tidal", heos.SourceIDTidal, "Tidal")
}

func newSearchSoundCloudCmd(flags *rootFlags) *cobra.Command {
	return newSearchServiceCmd(flags, "soundcloud", heos.SourceIDSoundCloud, "SoundCloud")
}

func newSearchServiceCmd(flags *rootFlags, use string, sourceID int, label string) *cobra.Command {
	var doPlay, doEnqueue bool
	var index int
	var showRaw bool
	cmd := &cobra.Command{
		Use:   use + " <query>",
		Short: "Search " + label,
		Long:  "Searches " + label + " and prints results. Use --play or --enqueue to play/add on the target player.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if doPlay && doEnqueue {
				return errors.New("use only one of --play or --enqueue")
			}
			if (doPlay || doEnqueue) && flags.IP == "" && flags.Name == "" {
				return errors.New("--play/--enqueue require --ip or --name")
			}
			query := strings.TrimSpace(strings.Join(args, " "))
			ctx := cmd.Context()
			c, err := targetClient(ctx, flags)
			if err != nil {
				return err
			}
			defer func() { _ = c.Disconnect() }()
			if err := c.Connect(ctx); err != nil {
				return err
			}
			criteria, err := c.GetSearchCriteria(ctx, sourceID)
			if err != nil {
				return err
			}
			var scid int
			for _, cr := range criteria {
				if strings.EqualFold(cr.Name, "Track") {
					scid = cr.Scid
					break
				}
			}
			if scid == 0 && len(criteria) > 0 {
				scid = criteria[0].Scid
			}
			result, err := c.Search(ctx, sourceID, query, scid)
			if err != nil {
				return err
			}
			if len(result.Items) == 0 {
				return errors.New("no results")
			}
			if showRaw {
				_, _ = cmd.OutOrStdout().Write(result.RawPayload)
				_, _ = cmd.OutOrStdout().Write([]byte("\n"))
				return nil
			}
			if doPlay || doEnqueue {
				c2, pid, err := resolvePlayerID(ctx, flags)
				if err != nil {
					return err
				}
				if index <= 0 {
					index = 1
				}
				if index > len(result.Items) {
					return fmt.Errorf("--index %d out of range (got %d results)", index, len(result.Items))
				}
				item := result.Items[index-1]
				containerID := heos.SearchCriteriaTracksPrefix + query
				addCrit := heos.AddCriteriaPlayNow
				if doEnqueue {
					addCrit = 3
				}
				if err := c2.AddToQueue(ctx, pid, sourceID, containerID, item.MediaID, addCrit); err != nil {
					return err
				}
				if doPlay {
					_ = c2.SetPlayState(ctx, pid, heos.PlayStatePlay)
				}
			}
			if isJSON(flags) {
				return writeJSON(cmd, result)
			}
			if isTSV(flags) {
				for i, r := range result.Items {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%s\t%s\t%s\t%s\n", i+1, r.Type, r.Name, r.Artist, r.MediaID, r.Quality)
				}
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "INDEX\tTYPE\tTITLE\tARTIST\tMID\tQUALITY")
			for i, r := range result.Items {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n", i+1, r.Type, r.Name, r.Artist, r.MediaID, r.Quality)
			}
			return w.Flush()
		},
	}
	cmd.Flags().BoolVar(&doPlay, "play", false, "Play the selected result on target player")
	cmd.Flags().BoolVar(&doEnqueue, "enqueue", false, "Add the selected result to queue")
	cmd.Flags().IntVar(&index, "index", 1, "Which result to use with --play/--enqueue (1-based)")
	cmd.Flags().BoolVar(&showRaw, "show-raw", false, "Print raw JSON payload from device (to inspect available keys, e.g. quality)")
	return cmd
}
