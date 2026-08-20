package log_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/logsquaredn/blobproxy/internal/log"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestConfig_AddFlags(t *testing.T) {
	cfg := new(log.Config)
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)

	cfg.AddFlags(flagSet)

	// This would pollute the test if set.
	require.NoError(t, os.Unsetenv("DEBUG"), "unset DEBUG environment variable")
	require.Equal(t, slog.LevelInfo, cfg.Level())

	require.NoError(t, flagSet.Parse([]string{"--debug"}), "parse --debug")
	require.Equal(t, slog.LevelDebug, cfg.Level())

	require.NoError(t, flagSet.Parse([]string{"--quiet"}), "parse --quiet")
	require.Equal(t, slog.LevelError, cfg.Level())

	require.NoError(t, flagSet.Parse([]string{"-v"}), "parse -v")
	require.Equal(t, slog.LevelWarn, cfg.Level())
}
