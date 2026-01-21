package commands

import (
	"context"
	"io"

	"github.com/alecthomas/kingpin/v2"

	"github.com/slok/sloth/internal/log"
)

// Command represents an application command, all commands that want to be executed
// should implement and setup on main.
type Command interface {
	Name() string
	Run(ctx context.Context, config RootConfig) error
}

// RootConfig represents the root command configuration and global configuration
// for all the commands.
type RootConfig struct {
	// Global flags.
	LogLevel string
	NoLog    bool

	// Global instances.
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Logger log.Logger
}

// NewRootConfig initializes the main root configuration.
func NewRootConfig(app *kingpin.Application) *RootConfig {
	c := &RootConfig{}

	// Register.
	app.Flag("log-level", "Log level.").Default("info").EnumVar(&c.LogLevel, "debug", "info", "warn", "error")
	app.Flag("no-log", "Disable logger.").BoolVar(&c.NoLog)

	return c
}
