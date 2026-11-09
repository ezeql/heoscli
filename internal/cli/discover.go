package cli

import (
	"errors"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"heoscli/internal/heos"
)

func newDiscoverCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "Discover HEOS devices on the local network",
		Long:  "Sends SSDP M-SEARCH and optionally scans port 1255. Use --timeout to wait longer.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			devs, err := heosDiscover(ctx, heos.DiscoverOptions{Timeout: flags.Timeout})
			if err != nil {
				return err
			}
			if len(devs) == 0 {
				if isJSON(flags) {
					return writeJSON(cmd, []heos.Device{})
				}
				return errors.New("no HEOS devices found (try increasing --timeout)")
			}
			sort.Slice(devs, func(i, j int) bool {
				if devs[i].Name != devs[j].Name {
					return devs[i].Name < devs[j].Name
				}
				return devs[i].IP < devs[j].IP
			})
			if isJSON(flags) {
				return writeJSON(cmd, devs)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "NAME\tIP\tUDN\n")
			for _, d := range devs {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", d.Name, d.IP, d.UDN)
			}
			return nil
		},
	}
}
