package prometheus

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/slok/sloth/test/integration/testutils"
)

type Config struct {
	Binary string
}

func (c *Config) defaults() error {
	if c.Binary == "" {
		c.Binary = "sloth"
	}

	_, err := exec.LookPath(c.Binary)
	if err != nil {
		return fmt.Errorf("sloth binary missing in %q: %w", c.Binary, err)
	}

	return nil
}

// NewIntegrationConfig prepares the configuration for integration tests, if the configuration is not ready
// it will skip the test.
func NewConfig(t *testing.T) Config {
	const (
		envSlothBin = "SLOTH_INTEGRATION_BINARY"
	)

	c := Config{
		Binary: os.Getenv(envSlothBin),
	}

	err := c.defaults()
	if err != nil {
		t.Skipf("Skipping due to invalid config: %s", err)
	}

	return c
}

func RunSlothGenerate(ctx context.Context, config Config, cmdArgs string) (stdout, stderr []byte, err error) {
	env := []string{
		fmt.Sprintf("SLOTH_SLI_PLUGINS_PATH=%s", "./"),
	}

	return testutils.RunSloth(ctx, env, config.Binary, fmt.Sprintf("generate %s", cmdArgs), true)
}

func RunSlothValidate(ctx context.Context, config Config, cmdArgs string) (stdout, stderr []byte, err error) {
	env := []string{
		fmt.Sprintf("SLOTH_SLI_PLUGINS_PATH=%s", "./"),
	}

	return testutils.RunSloth(ctx, env, config.Binary, fmt.Sprintf("validate %s", cmdArgs), true)
}
