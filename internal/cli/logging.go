package cli

import (
	"log/slog"
	"os"
)

func enableDebugLogging() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
}
