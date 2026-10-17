package config

import (
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type (
	Specification struct {
		LogLevel               jsonlog.Level `envconfig:"LOG_LEVEL"        default:"0"`
		StatusPort             string        `envconfig:"STATUS_PORT"      default:"8000"`
		PullPeriodCronSchedule string        `envconfig:"PULLPERIODCRONSCHEDULE"        default:"* * * * *" required:"true"`
		DB
		APIService
	}
	DB struct {
		DataSourceName string `envconfig:"DATASOURCENAME" required:"true"`
		Driver         string `envconfig:"DRIVER" required:"true"`
		MaxIdleConns   int    `envconfig:"MAXIDLECONNS"       default:"10"`
		MaxOpenConns   int    `envconfig:"MAXOPENCONNS"       default:"80"`
	}
	APIService struct {
		Port int `envconfig:"PORT"     default:"8080"        required:"true"`
		DB
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
