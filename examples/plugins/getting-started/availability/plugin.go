package availability

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

const (
	SLIPluginVersion = "prometheus/v1"
	SLIPluginID      = "getting_started_availability"
)

var queryTpl = template.Must(template.New("").Parse(`
sum(rate(http_request_duration_seconds_count{ {{.filter}}job="{{.job}}",code=~"(5..|429)" }[{{"{{.window}}"}}]))
/
sum(rate(http_request_duration_seconds_count{ {{.filter}}job="{{.job}}" }[{{"{{.window}}"}}]))`))

var filterRegex = regexp.MustCompile(`([^=]+="[^=,"]+",)+`)

// SLIPlugin is the getting started plugin example.
//
// It will return an Sloth error ratio raw query that returns the error ratio of HTTP requests based
// on the HTTP response status code, taking 5xx and 429 as error events.
func SLIPlugin(ctx context.Context, meta, labels, options map[string]string) (string, error) {
	// Get job.
	job, ok := options["job"]
	if !ok {
		return "", fmt.Errorf("job options is required")
	}

	// Validate labels.
	err := validateLabels(labels, "owner", "tier")
	if err != nil {
		return "", fmt.Errorf("invalid labels: %w", err)
	}

	// Sanitize filter.
	filter := options["filter"]
	if filter != "" {
		filter = strings.Trim(filter, "{}")
		filter = strings.Trim(filter, ",")
		filter = filter + ","
		match := filterRegex.MatchString(filter)
		if !match {
			return "", fmt.Errorf("invalid prometheus filter: %s", filter)
		}
	}

	// Create query.
	var b bytes.Buffer
	data := map[string]string{
		"job":    job,
		"filter": filter,
	}
	err = queryTpl.Execute(&b, data)
	if err != nil {
		return "", fmt.Errorf("could not execute template: %w", err)
	}

	return b.String(), nil
}

// validateLabels will check the labels exist.
func validateLabels(labels map[string]string, requiredKeys ...string) error {
	for _, k := range requiredKeys {
		v, ok := labels[k]
		if !ok || (ok && v == "") {
			return fmt.Errorf("%q label is required", k)
		}
	}

	return nil
}
