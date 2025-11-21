package ui_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/slok/sloth/internal/http/ui"
	"github.com/slok/sloth/internal/http/ui/uimock"
)

var trimSpaceMultilineRegexp = regexp.MustCompile(`(?m)(^\s+|\s+$)`)

func assertContainsHTTPResponseBody(t *testing.T, exp []string, resp *httptest.ResponseRecorder) {
	// Sanitize got HTML so we make easier to check content.
	got := resp.Body.String()
	got = trimSpaceMultilineRegexp.ReplaceAllString(got, "")
	got = strings.ReplaceAll(got, "\n", " ")

	// Check each expected snippet.
	for _, e := range exp {
		assert.Contains(t, got, e)
	}
}

type mocks struct {
	ServiceApp *uimock.ServiceApp
}

func newMocks(t *testing.T) mocks {
	return mocks{
		ServiceApp: &uimock.ServiceApp{},
	}
}

// Always now is an specific time for tests idempotency.
var testTimeNow, _ = time.Parse(time.RFC3339, "2025-11-15T01:02:03Z")

func newTestUIHandler(t *testing.T, m mocks) http.Handler {
	h, err := ui.NewUI(ui.UIConfig{
		ServiceApp:  m.ServiceApp,
		TimeNowFunc: func() time.Time { return testTimeNow },
	})
	require.NoError(t, err)

	return h
}
