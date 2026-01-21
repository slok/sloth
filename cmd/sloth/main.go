package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/alecthomas/kingpin/v2"

	"github.com/slok/sloth/cmd/sloth/commands"
	"github.com/slok/sloth/internal/info"
	"github.com/slok/sloth/internal/log"
)

// Run runs the main application.
func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	app := kingpin.New("sloth", "Easy SLO generator.")
	app.DefaultEnvars()
	config := commands.NewRootConfig(app)

	// Setup commands (registers flags).
	generateCmd := commands.NewGenerateCommand(app)
	kubeCtrlCmd := commands.NewKubeControllerCommand(app)
	serverCmd := commands.NewServerCommand(app)
	validateCmd := commands.NewValidateCommand(app)
	versionCmd := commands.NewVersionCommand(app)

	cmds := map[string]commands.Command{
		generateCmd.Name(): generateCmd,
		kubeCtrlCmd.Name(): kubeCtrlCmd,
		serverCmd.Name():   serverCmd,
		validateCmd.Name(): validateCmd,
		versionCmd.Name():  versionCmd,
	}

	// Parse commandline.
	cmdName, err := app.Parse(args[1:])
	if err != nil {
		return fmt.Errorf("invalid command configuration: %w", err)
	}

	// Set up global dependencies.
	config.Stdin = stdin
	config.Stdout = stdout
	config.Stderr = stderr
	config.Logger = getLogger(*config)

	// Execute command.
	err = cmds[cmdName].Run(ctx, *config)
	if err != nil {
		return fmt.Errorf("%q command failed: %w", cmdName, err)
	}

	return nil
}

// getLogger returns the application logger.
func getLogger(config commands.RootConfig) log.Logger {
	if config.NoLog {
		return log.Noop
	}

	// Parse log level.
	var level slog.Level
	switch config.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	slogLogger := slog.New(slog.NewTextHandler(config.Stderr, opts))
	logger := log.NewSlog(slogLogger).WithValues(log.Kv{
		"version": info.Version,
	})

	logger.Debugf("Debug level is enabled") // Will log only when debug enabled.

	return logger
}

func main() {
	ctx := context.Background()
	err := Run(ctx, os.Args, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
}
