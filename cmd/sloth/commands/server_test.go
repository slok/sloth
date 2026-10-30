package commands_test

import (
	"testing"

	"github.com/alecthomas/kingpin/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/cmd/sloth/commands"
)

// TestServerCommandPrometheusHeaderFlag checks that the `server --prometheus-header`
// flag can be parsed without panicking. Regression test for the nil map panic
// ("assignment to entry in nil map") reported when the backing map was not
// initialized before being used by kingpin's StringMap value.
func TestServerCommandPrometheusHeaderFlag(t *testing.T) {
	app := kingpin.New("sloth", "test")
	commands.NewServerCommand(app)

	cmd, err := app.Parse([]string{"server", "--prometheus-header=Some-Header=Value"})
	require.NoError(t, err)
	assert.Equal(t, "server", cmd)
}
