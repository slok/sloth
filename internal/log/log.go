package log

import "context"

// Kv is a helper type for structured logging fields usage.
type Kv = map[string]interface{}

// Logger is the interface that the loggers used by the library will use.
type Logger interface {
	Infof(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	WithValues(values map[string]interface{}) Logger
	WithCtxValues(ctx context.Context) Logger
	SetValuesOnCtx(parent context.Context, values map[string]interface{}) context.Context
}

// Noop logger doesn't log anything.
const Noop = noop(0)

type noop int

func (n noop) Infof(format string, args ...interface{})                         {}
func (n noop) Warningf(format string, args ...interface{})                      {}
func (n noop) Errorf(format string, args ...interface{})                        {}
func (n noop) Debugf(format string, args ...interface{})                        {}
func (n noop) WithValues(map[string]interface{}) Logger                         { return n }
func (n noop) WithCtxValues(context.Context) Logger                             { return n }
func (n noop) SetValuesOnCtx(parent context.Context, values Kv) context.Context { return parent }

type contextKey string

// contextLogValuesKey used as unique key to store log values in the context.
const contextLogValuesKey = contextKey("internal-log")

// CtxWithValues returns a copy of parent in which the key values passed have been
// stored ready to be used using log.Logger.
func CtxWithValues(parent context.Context, kv Kv) context.Context {
	// Maybe we have values already set.
	oldValues, ok := parent.Value(contextLogValuesKey).(Kv)
	if !ok {
		oldValues = Kv{}
	}

	// Copy old and received values into the new kv.
	newValues := Kv{}
	for k, v := range oldValues {
		newValues[k] = v
	}
	for k, v := range kv {
		newValues[k] = v
	}

	return context.WithValue(parent, contextLogValuesKey, newValues)
}

// ValuesFromCtx gets the log Key values from a context.
func ValuesFromCtx(ctx context.Context) Kv {
	values, ok := ctx.Value(contextLogValuesKey).(Kv)
	if !ok {
		return Kv{}
	}

	return values
}
