// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// SLIEventsApplyConfiguration represents a declarative configuration of the SLIEvents type for use
// with apply.
type SLIEventsApplyConfiguration struct {
	ErrorQuery *string `json:"errorQuery,omitempty"`
	TotalQuery *string `json:"totalQuery,omitempty"`
}

// SLIEventsApplyConfiguration constructs a declarative configuration of the SLIEvents type for use with
// apply.
func SLIEvents() *SLIEventsApplyConfiguration {
	return &SLIEventsApplyConfiguration{}
}

// WithErrorQuery sets the ErrorQuery field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ErrorQuery field is set to the value of the last call.
func (b *SLIEventsApplyConfiguration) WithErrorQuery(value string) *SLIEventsApplyConfiguration {
	b.ErrorQuery = &value
	return b
}

// WithTotalQuery sets the TotalQuery field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the TotalQuery field is set to the value of the last call.
func (b *SLIEventsApplyConfiguration) WithTotalQuery(value string) *SLIEventsApplyConfiguration {
	b.TotalQuery = &value
	return b
}
