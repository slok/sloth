package commands

import (
	"context"
	"fmt"

	"gopkg.in/alecthomas/kingpin.v2"
)

type generateCommand struct{}

// NewGenerateCommand returns the generate command.
func NewGenerateCommand(app *kingpin.Application) Command {
	app.Command("generate", "Generates SLOs.")
	return generateCommand{}
}

func (g generateCommand) Name() string { return "generate" }
func (g generateCommand) Run(ctx context.Context, config RootConfig) error {
	return fmt.Errorf("not implemented")
}
