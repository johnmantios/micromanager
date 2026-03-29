package main

import (
	"github.com/johnmantios/micromanager/pkg/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{
		FieldMap: log.FieldMap{
			log.FieldKeyTime: "@timestamp",
			log.FieldKeyMsg:  "message",
		},
	})

	conf, err := config.LoadEnv()
	if err != nil {
		log.WithError(err).Panic("Loading config from environment failed")
	}

	logLevel, err := log.ParseLevel(conf.LogLevel)
	if err != nil {
		log.WithError(err).Panic("parsing the level of logs failed")
	}

	log.SetLevel(logLevel)

	rootCmd := &cobra.Command{Use: "micromanager"}

	rootCmd.AddCommand(unlockedTimeCmd(conf))

	err = rootCmd.Execute()
	if err != nil {
		log.WithError(err).Panic("Command failed")
	}
}
