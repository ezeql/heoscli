package cli

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"heoscli/internal/appconfig"
	"heoscli/internal/heos"
)

type rootFlags struct {
	IP      string
	Name    string
	Timeout time.Duration
	Format  string
	JSON    bool
	Debug   bool
}

var newHeosClient = heos.NewClient
var heosDiscover = heos.Discover

var loadAppConfig = func() (appconfig.Config, error) {
	s, err := appconfig.NewDefaultStore()
	if err != nil {
		return appconfig.Config{}, err
	}
	return s.Load()
}

func Execute() error {
	_ = loadDotEnv(".env")
	rootCmd, _, err := newRootCmd()
	if err != nil {
		return err
	}
	ctx := context.Background()
	rootCmd.SetContext(ctx)
	return rootCmd.Execute()
}

func newRootCmd() (*cobra.Command, *rootFlags, error) {
	flags := &rootFlags{}
	cfg, err := loadAppConfig()
	if err != nil {
		return nil, nil, err
	}
	cfg = cfg.Normalize()

	rootCmd := &cobra.Command{
		Use:     "heos",
		Short:   "Control HEOS devices from the command line",
		Long:    "Control HEOS devices over the local network: discover devices, sign in, search Tidal/SoundCloud, play and control playback.",
		Example: "  heos discover\n  heos status --name \"Living Room\"\n  heos search tidal \"artist\" --play\n  heos play --name \"Living Room\"",
		SilenceUsage: true,
		Version: Version,
	}
	rootCmd.SetVersionTemplate("heos {{.Version}}\n")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if flags.Debug {
			enableDebugLogging()
		}
		format := strings.TrimSpace(flags.Format)
		if format == "" {
			format = formatPlain
		}
		format = strings.ToLower(format)
		if flags.JSON && format == formatPlain {
			format = formatJSON
		}
		norm, err := normalizeFormat(format)
		if err != nil {
			return err
		}
		flags.Format = norm
		return nil
	}

	rootCmd.PersistentFlags().StringVar(&flags.IP, "ip", "", "Target device IP address")
	rootCmd.PersistentFlags().StringVar(&flags.Name, "name", cfg.DefaultDevice, "Target device name")
	rootCmd.PersistentFlags().DurationVar(&flags.Timeout, "timeout", 5*time.Second, "Timeout for discovery and network calls")
	rootCmd.PersistentFlags().StringVar(&flags.Format, "format", cfg.Format, "Output format: plain|json|tsv")
	rootCmd.PersistentFlags().BoolVar(&flags.JSON, "json", false, "Deprecated: use --format json")
	_ = rootCmd.PersistentFlags().MarkDeprecated("json", "use --format json")
	rootCmd.PersistentFlags().BoolVar(&flags.Debug, "debug", false, "Enable debug logging")

	rootCmd.AddCommand(newDiscoverCmd(flags))
	rootCmd.AddCommand(newStatusCmd(flags))
	rootCmd.AddCommand(newPlayCmd(flags))
	rootCmd.AddCommand(newPauseCmd(flags))
	rootCmd.AddCommand(newStopCmd(flags))
	rootCmd.AddCommand(newNextCmd(flags))
	rootCmd.AddCommand(newPrevCmd(flags))
	rootCmd.AddCommand(newVolumeCmd(flags))
	rootCmd.AddCommand(newMuteCmd(flags))
	rootCmd.AddCommand(newQueueCmd(flags))
	rootCmd.AddCommand(newSearchCmd(flags))
	rootCmd.AddCommand(newSignInCmd(flags))
	rootCmd.AddCommand(newSignOutCmd(flags))
	rootCmd.AddCommand(newConfigCmd(flags))

	return rootCmd, flags, nil
}

func validateTarget(flags *rootFlags) error {
	if flags.IP == "" && flags.Name == "" {
		return errors.New("provide --ip or --name (or run `heos discover`)")
	}
	return nil
}

// resolveTargetIP returns the IP to connect to (from --ip or discover by --name).
func resolveTargetIP(ctx context.Context, flags *rootFlags) (string, error) {
	if err := validateTarget(flags); err != nil {
		return "", err
	}
	if flags.IP != "" {
		return flags.IP, nil
	}
	devs, err := heosDiscover(ctx, heos.DiscoverOptions{Timeout: flags.Timeout})
	if err != nil {
		return "", err
	}
	if len(devs) == 0 {
		return "", errors.New("no HEOS devices found")
	}
	name := strings.TrimSpace(flags.Name)
	for _, d := range devs {
		if strings.EqualFold(strings.TrimSpace(d.Name), name) || d.IP == name {
			return d.IP, nil
		}
	}
	return "", errors.New("device name not found: " + flags.Name)
}

// targetClient returns a client connected to the resolved target IP.
func targetClient(ctx context.Context, flags *rootFlags) (*heos.Client, error) {
	ip, err := resolveTargetIP(ctx, flags)
	if err != nil {
		return nil, err
	}
	return newHeosClient(ip, flags.Timeout), nil
}

// resolvePlayerID connects to target, loads players, and returns the player ID matching flags.Name (or first player if by IP).
func resolvePlayerID(ctx context.Context, flags *rootFlags) (*heos.Client, int, error) {
	c, err := targetClient(ctx, flags)
	if err != nil {
		return nil, 0, err
	}
	if err := c.Connect(ctx); err != nil {
		return nil, 0, err
	}
	players, err := c.GetPlayers(ctx)
	if err != nil {
		return nil, 0, err
	}
	if len(players) == 0 {
		return nil, 0, errors.New("no players on device")
	}
	name := strings.TrimSpace(flags.Name)
	if flags.IP != "" && name == "" {
		return c, players[0].PID, nil
	}
	for _, p := range players {
		if strings.EqualFold(strings.TrimSpace(p.Name), name) || (name != "" && p.IP == flags.IP) {
			return c, p.PID, nil
		}
	}
	return c, players[0].PID, nil
}
