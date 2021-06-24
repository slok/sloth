package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/slok/sloth/cmd/sloth/commands"
	"github.com/slok/sloth/internal/info"
	"github.com/slok/sloth/internal/log"
	loglogrus "github.com/slok/sloth/internal/log/logrus"
)

// Run runs the main application.
func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	app := kingpin.New("sloth", "Easy SLO generator.")
	app.DefaultEnvars()
	config := commands.NewRootConfig(app)

	// Setup commands (registers flags).
	generateCmd := commands.NewGenerateCommand(app)
	kubeCtrlCmd := commands.NewKubeControllerCommand(app)
	validateCmd := commands.NewValidateCommand(app)
	versionCmd := commands.NewVersionCommand(app)

	cmds := map[string]commands.Command{
		generateCmd.Name(): generateCmd,
		kubeCtrlCmd.Name(): kubeCtrlCmd,
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

	// If not logger disabled use logrus logger.
	logrusLog := logrus.New()
	logrusLog.Out = config.Stderr // By default logger goes to stderr (so it can split stdout prints).
	logrusLogEntry := logrus.NewEntry(logrusLog)

	if config.Debug {
		logrusLogEntry.Logger.SetLevel(logrus.DebugLevel)
	}

	// Log format.
	switch config.LoggerType {
	case commands.LoggerTypeDefault:
		logrusLogEntry.Logger.SetFormatter(&logrus.TextFormatter{
			ForceColors:   !config.NoColor,
			DisableColors: config.NoColor,
		})
	case commands.LoggerTypeJSON:
		logrusLogEntry.Logger.SetFormatter(&logrus.JSONFormatter{})
	}

	logger := loglogrus.NewLogrus(logrusLogEntry).WithValues(log.Kv{
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
