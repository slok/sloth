package app

import (
	"fmt"
	"time"

	"github.com/slok/sloth/internal/http/backend/storage"
)

type AppConfig struct {
	ServiceGetter storage.ServiceGetter
	SLOGetter     storage.SLOGetter
	TimeNowFunc   func() time.Time
}

func (c *AppConfig) defaults() error {
	if c.ServiceGetter == nil {
		return fmt.Errorf("service getter is required")
	}
	if c.SLOGetter == nil {
		return fmt.Errorf("slo getter is required")
	}
	if c.TimeNowFunc == nil {
		c.TimeNowFunc = time.Now
	}

	return nil
}

type App struct {
	serviceGetter storage.ServiceGetter
	sloGetter     storage.SLOGetter
	timeNowFunc   func() time.Time
}

func NewApp(config AppConfig) (*App, error) {
	if err := config.defaults(); err != nil {
		return nil, err
	}

	return &App{
		serviceGetter: config.ServiceGetter,
		sloGetter:     config.SLOGetter,
		timeNowFunc:   config.TimeNowFunc,
	}, nil
}
