package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStatusCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Aliases: []string{"now"},
		Short:   "Show current playback status",
		Long:    "Prints player status: transport state, now playing, volume, mute.",
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

			state, err := c.GetPlayState(ctx, pid)
			if err != nil {
				return err
			}
			np, err := c.GetNowPlayingMedia(ctx, pid)
			if err != nil {
				return err
			}
			vol, err := c.GetVolume(ctx, pid)
			if err != nil {
				return err
			}
			mute, err := c.GetMute(ctx, pid)
			if err != nil {
				return err
			}

			out := map[string]any{
				"playerId": pid,
				"state":    state,
				"volume":   vol,
				"mute":     mute,
			}
			if np != nil {
				out["song"] = np.Song
				out["artist"] = np.Artist
				out["album"] = np.Album
				if np.Quality != "" {
					out["quality"] = np.Quality
				}
			}
			if isJSON(flags) {
				return writeJSON(cmd, out)
			}
			if isTSV(flags) {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "state\t%s\n", state)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "volume\t%d\n", vol)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "mute\t%v\n", mute)
				if np != nil {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "song\t%s\n", np.Song)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "artist\t%s\n", np.Artist)
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "album\t%s\n", np.Album)
					if np.Quality != "" {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(), "quality\t%s\n", np.Quality)
					}
				}
				return nil
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Player ID:\t%d\n", pid)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "State:\t\t%s\n", state)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Volume:\t\t%d\n", vol)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Mute:\t\t%v\n", mute)
			if np != nil {
				if np.Song != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Song:\t\t%s\n", np.Song)
				}
				if np.Artist != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Artist:\t\t%s\n", np.Artist)
				}
				if np.Album != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Album:\t\t%s\n", np.Album)
				}
				if np.Quality != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Quality:\t%s\n", np.Quality)
				}
			}
			return nil
		},
	}
}
