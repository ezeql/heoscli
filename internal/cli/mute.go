package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMuteCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mute",
		Short: "Get or set mute",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Get mute state",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, pid, err := resolvePlayerID(ctx, flags)
			if err != nil {
				return err
			}
			defer func() { _ = c.Disconnect() }()
			mute, err := c.GetMute(ctx, pid)
			if err != nil {
				return err
			}
			if isJSON(flags) {
				return writeJSON(cmd, map[string]any{"mute": mute, "playerId": pid})
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), mute)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "on",
		Short: "Mute",
		RunE:  runMute(flags, true),
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "off",
		Short: "Unmute",
		RunE:  runMute(flags, false),
	})
	return cmd
}

func runMute(flags *rootFlags, mute bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		c, pid, err := resolvePlayerID(ctx, flags)
		if err != nil {
			return err
		}
		defer func() { _ = c.Disconnect() }()
		if err := c.SetMute(ctx, pid, mute); err != nil {
			return err
		}
		return writeOK(cmd, flags, "mute", map[string]any{"playerId": pid, "mute": mute})
	}
}
