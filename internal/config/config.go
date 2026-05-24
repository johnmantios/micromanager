package config

import (
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type (
	Specification struct {
		LogLevel   jsonlog.Level `envconfig:"LOG_LEVEL"        default:"0"`
		StatusPort string        `envconfig:"STATUS_PORT"      default:"8000"`
		ActivityDB
		APIService
	}
	ActivityDB struct {
		FilePath               string `envconfig:"FILEPATH" default:"/Users/johnmantios/Downloads/knowledgeC.db" required:"true"`
		Driver                 string `envconfig:"DRIVER"             default:"sqlite3" required:"true"`
		MaxIdleConns           int    `envconfig:"MAXIDLECONNS"       default:"10"`
		MaxOpenConns           int    `envconfig:"MAXOPENCONNS"       default:"80"`
		PullPeriodCronSchedule string `envconfig:"PULLPERIODCRONSCHEDULE"        default:"* * * * *" required:"true"`
	}
	APIService struct {
		Port int `envconfig:"PORT"     default:"8080"        required:"true"`
		TimeSeriesDB
	}
	TimeSeriesDB struct {
		Driver       string `envconfig:"DRIVER"             default:"postgres" required:"true"`
		DSN          string `envconfig:"DSN"         default:"postgres://wins:wins@localhost:5432/wins?sslmode=disable&client_encoding=UTF8"    required:"true"`
		MaxIdleConns int    `envconfig:"MAXIDLECONNS"       default:"10"`
		MaxOpenConns int    `envconfig:"MAXOPENCONNS"       default:"1000"`
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
