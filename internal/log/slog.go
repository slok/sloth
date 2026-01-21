package log

import (
	"context"
	"fmt"
	"log/slog"
)

type slogLogger struct {
	l     *slog.Logger
	attrs []any
}

// NewSlog returns a new Logger for a slog implementation.
func NewSlog(l *slog.Logger) Logger {
	return &slogLogger{l: l}
}

func (l *slogLogger) Infof(format string, args ...interface{}) {
	l.l.Info(fmt.Sprintf(format, args...), l.attrs...)
}

func (l *slogLogger) Warningf(format string, args ...interface{}) {
	l.l.Warn(fmt.Sprintf(format, args...), l.attrs...)
}

func (l *slogLogger) Errorf(format string, args ...interface{}) {
	l.l.Error(fmt.Sprintf(format, args...), l.attrs...)
}

func (l *slogLogger) Debugf(format string, args ...interface{}) {
	l.l.Debug(fmt.Sprintf(format, args...), l.attrs...)
}

func (l *slogLogger) WithValues(kv Kv) Logger {
	// Convert map to slog key-value pairs and merge with existing attrs.
	newAttrs := make([]any, len(l.attrs), len(l.attrs)+len(kv)*2)
	copy(newAttrs, l.attrs)

	for k, v := range kv {
		newAttrs = append(newAttrs, k, v)
	}

	return &slogLogger{
		l:     l.l,
		attrs: newAttrs,
	}
}

func (l *slogLogger) WithCtxValues(ctx context.Context) Logger {
	return l.WithValues(ValuesFromCtx(ctx))
}

func (l *slogLogger) SetValuesOnCtx(parent context.Context, values Kv) context.Context {
	return CtxWithValues(parent, values)
}
