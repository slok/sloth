package availability

import (
	"bytes"
	"fmt"
	"text/template"
)

const SLIPluginID = "getting_started_availability"

var tpl = template.Must(template.New("").Parse(`
sum(rate(http_request_duration_seconds_count{from_plugin="true",job="{{.job}}",code=~"(5..|429)"}[{{"{{.window}}"}}]))
/
sum(rate(http_request_duration_seconds_count{from_plugin="true",job="{{.job}}"}[{{"{{.window}}"}}]))
`))

func SLIPlugin(_ map[string]interface{}, options map[string]interface{}) (string, error) {
	_, ok := options["job"]
	if !ok {
		return "", fmt.Errorf("job option is required")
	}

	var b bytes.Buffer
	err := tpl.Execute(&b, options)
	if err != nil {
		return "", fmt.Errorf("could not execute template: %w", err)
	}

	return b.String(), nil
}
