package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	envHEOSUsername = "HEOS_USERNAME"
	envHEOSPassword = "HEOS_PASSWORD"
)

// readPasswordFromStdin reads a line from stdin. If stdin is a TTY, input is not echoed (hidden).
// Use for passwords. When piped (e.g. echo "mypass" | heos sign-in), reads the line as usual.
func readPasswordFromStdin() (string, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	return readLine()
}

// stdin is shared by every prompt: a reader per prompt would buffer past its
// own line and swallow the next answer when input is piped.
var stdin = bufio.NewReader(os.Stdin)

func readLine() (string, error) {
	line, err := stdin.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// readLineFromStdin reads a single line from stdin (echoed). Use for username.
func readLineFromStdin(prompt string) (string, error) {
	if prompt != "" {
		_, _ = fmt.Fprint(os.Stderr, prompt)
	}
	return readLine()
}

func newSignInCmd(flags *rootFlags) *cobra.Command {
	var username, password string
	cmd := &cobra.Command{
		Use:   "sign-in",
		Short: "Sign in to HEOS account",
		Long:  "Signs in to your HEOS account on the target device. Credentials can be stored in config (heos config set heosUsername/user).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTarget(flags); err != nil {
				return err
			}
			if username == "" {
				username = os.Getenv(envHEOSUsername)
			}
			if password == "" {
				password = os.Getenv(envHEOSPassword)
			}
			if username == "" || password == "" {
				cfg, _ := loadAppConfig()
				if username == "" {
					username = cfg.HEOSUsername
				}
				if password == "" {
					password = cfg.HEOSPassword
				}
			}
			if username == "" {
				var err error
				username, err = readLineFromStdin("Username: ")
				if err != nil {
					return err
				}
				if username == "" {
					return errors.New("username is required")
				}
			}
			if password == "" {
				_, _ = fmt.Fprint(os.Stderr, "Password: ")
				var err error
				password, err = readPasswordFromStdin()
				_, _ = fmt.Fprintln(os.Stderr)
				if err != nil {
					return err
				}
				if password == "" {
					return errors.New("password is required")
				}
			}
			ctx := cmd.Context()
			c, err := targetClient(ctx, flags)
			if err != nil {
				return err
			}
			defer func() { _ = c.Disconnect() }()
			u, err := c.SignIn(ctx, username, password)
			if err != nil {
				return err
			}
			if isJSON(flags) {
				return writeJSON(cmd, map[string]any{"signedIn": true, "username": u})
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Signed in as %s\n", u)
			return nil
		},
	}
	cmd.Flags().StringVar(&username, "username", "", "HEOS account username")
	cmd.Flags().StringVar(&password, "password", "", "HEOS account password")
	return cmd
}

func newSignOutCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "sign-out",
		Short: "Sign out of HEOS account",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTarget(flags); err != nil {
				return err
			}
			ctx := cmd.Context()
			c, err := targetClient(ctx, flags)
			if err != nil {
				return err
			}
			defer func() { _ = c.Disconnect() }()
			if err := c.SignOut(ctx); err != nil {
				return err
			}
			return writeOK(cmd, flags, "sign-out", nil)
		},
	}
}
