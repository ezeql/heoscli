package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newVolumeCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Get or set volume (0-100)",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Get volume",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, pid, err := resolvePlayerID(ctx, flags)
			if err != nil {
				return err
			}
			defer func() { _ = c.Disconnect() }()
			v, err := c.GetVolume(ctx, pid)
			if err != nil {
				return err
			}
			if isJSON(flags) {
				return writeJSON(cmd, map[string]any{"volume": v, "playerId": pid})
			}
			if isTSV(flags) {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "volume\t%d\n", v)
				return nil
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), v)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "set [0-100]",
		Short: "Set volume",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			c, pid, err := resolvePlayerID(ctx, flags)
			if err != nil {
				return err
			}
			defer func() { _ = c.Disconnect() }()
			v, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}
			if err := c.SetVolume(ctx, pid, v); err != nil {
				return err
			}
			return writeOK(cmd, flags, "volume.set", map[string]any{"playerId": pid, "volume": v})
		},
	})
	return cmd
}
