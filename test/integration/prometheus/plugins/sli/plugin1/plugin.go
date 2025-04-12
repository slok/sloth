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
	SLIPluginID      = "integration_test"
)

var tpl = template.Must(template.New("").Parse(`
sum(rate(integration_test{ {{.filter}}job="{{.job}}",code=~"(5..|429)" }[{{"{{.window}}"}}]))
/
sum(rate(integration_test{ {{.filter}}job="{{.job}}" }[{{"{{.window}}"}}]))`))

var filterRegex = regexp.MustCompile(`([^=]+="[^=,"]+",)+`)

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
	err = tpl.Execute(&b, data)
	if err != nil {
		return "", fmt.Errorf("could not execute template: %w", err)
	}

	return b.String(), nil
}

func validateLabels(labels map[string]string, requiredKeys ...string) error {
	for _, k := range requiredKeys {
		v, ok := labels[k]
		if !ok || (ok && v == "") {
			return fmt.Errorf("%q label is required", k)
		}
	}

	return nil
}
