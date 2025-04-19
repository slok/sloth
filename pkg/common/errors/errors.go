package errors

import "fmt"

var (
	// ErrNoSLORules will be used when there are no rules to store. The upper layer
	// could ignore or handle the error in cases where there wasn't an output.
	ErrNoSLORules = fmt.Errorf("0 SLO Prometheus rules generated")

	// ErrNotFound will be used when a resource has not been found.
	ErrNotFound = fmt.Errorf("resource not found")

	// ErrRequired will be used when a required field is not set.
	ErrRequired = fmt.Errorf("required")
)
