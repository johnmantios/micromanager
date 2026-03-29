package config

import (
	"github.com/kelseyhightower/envconfig"
	"time"
)

type (
	Specification struct {
		LogLevel   string `envconfig:"LOG_LEVEL"        default:"info"`
		StatusPort string `envconfig:"STATUS_PORT"      default:"8000"`
		EventsDB
	}
	EventsDB struct {
		DSN              string        `envconfig:"DATABASE_DSN"`
		Driver           string        `envconfig:"POSTGRES_DRIVER"             default:"postgres"`
		MaxIdleConns     int           `envconfig:"POSTGRES_MAXIDLECONNS"       default:"10"`
		MaxOpenConns     int           `envconfig:"POSTGRES_MAXOPENCONNS"       default:"80"`
		MaxConnLifetime  time.Duration `envconfig:"POSTGRES_MAXCONNLIFETIME"    default:"30m"`
		EventsPullPeriod int           `envconfig:"POSTGRES_PULL_EVENTS_PERIOD" default:"5"`
	}
)

func LoadEnv() (*Specification, error) {
	var config Specification

	err := envconfig.Process("", &config)
	if err != nil {
		return nil, err
	}

	return &config, err
}
