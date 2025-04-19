package conventions

import "regexp"

var (
	// NameRegexp is the regex to validate SLO, SLI and in general safe names and IDs.
	// Names must:
	// - Start and end with an alphanumeric.
	// - Contain alphanumeric, `.`, '_', and '-'.
	NameRegexp = regexp.MustCompile(`^[A-Za-z0-9][-A-Za-z0-9_.]*[A-Za-z0-9]$`)

	// TplWindowRegex is the regex to match the {{ .window }} template variable.
	TplWindowRegex = regexp.MustCompile(`{{ *\.window *}}`)
)
