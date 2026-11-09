package cli

import (
	"github.com/spf13/cobra"
	"heoscli/internal/heos"
)

func newPlayCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "play",
		Short: "Resume playback",
		RunE:  runTransport(flags, heos.PlayStatePlay),
	}
}

func newPauseCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "pause",
		Short: "Pause playback",
		RunE:  runTransport(flags, heos.PlayStatePause),
	}
}

func newStopCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop playback",
		RunE:  runTransport(flags, heos.PlayStateStop),
	}
}

func newNextCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "next",
		Short: "Skip to next track",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, pid, err := resolvePlayerID(ctx, flags)
			if err != nil {
				return err
			}
			defer func() { _ = c.Disconnect() }()
			if err := c.PlayNext(ctx, pid); err != nil {
				return err
			}
			return writeOK(cmd, flags, "next", map[string]any{"playerId": pid})
		},
	}
}

func newPrevCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "prev",
		Short: "Go to previous track",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, pid, err := resolvePlayerID(ctx, flags)
			if err != nil {
				return err
			}
			defer func() { _ = c.Disconnect() }()
			if err := c.PlayPrevious(ctx, pid); err != nil {
				return err
			}
			return writeOK(cmd, flags, "prev", map[string]any{"playerId": pid})
		},
	}
}

func runTransport(flags *rootFlags, state string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		c, pid, err := resolvePlayerID(ctx, flags)
		if err != nil {
			return err
		}
		defer func() { _ = c.Disconnect() }()
		if err := c.SetPlayState(ctx, pid, state); err != nil {
			return err
		}
		return writeOK(cmd, flags, state, map[string]any{"playerId": pid})
	}
}
