package logrus

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/slok/sloth/internal/log"
)

type logger struct {
	*logrus.Entry
}

// NewLogrus returns a new log.Logger for a logrus implementation.
func NewLogrus(l *logrus.Entry) log.Logger {
	return logger{Entry: l}
}

func (l logger) WithValues(kv log.Kv) log.Logger {
	newLogger := l.Entry.WithFields(kv)
	return NewLogrus(newLogger)
}

func (l logger) WithCtxValues(ctx context.Context) log.Logger {
	return l.WithValues(log.ValuesFromCtx(ctx))
}

func (l logger) SetValuesOnCtx(parent context.Context, values log.Kv) context.Context {
	return log.CtxWithValues(parent, values)
}
