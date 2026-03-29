package main

import (
	"github.com/johnmantios/micromanager/pkg/config"
	"github.com/johnmantios/micromanager/pkg/db"
	"github.com/johnmantios/micromanager/pkg/handler"
	"github.com/johnmantios/micromanager/pkg/unlocked_time"
	"github.com/johnmantios/micromanager/repo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os/exec"
)

func unlockedTimeCmd(configuration *config.Specification) *cobra.Command {
	return &cobra.Command{
		Use:   "unlocked-time",
		Short: "Monitors the active screen time of the host system",
		Long:  "Monitors the active screen time of the host system by saving the timestamps with a flag in a file",
		Run: func(cmd *cobra.Command, args []string) {

			dbConnection, err := db.OpenDB(configuration.EventsDB)
			if err != nil {
				log.Fatal(err, nil)
			}

			eventsModel, err := repo.NewEventsModel(dbConnection)
			if err != nil {
				log.Fatal(err, nil)
			}

			host := unlocked_time.NewHost(exec.Command)

			unlockTimeHandler := handler.NewUnlockedScreenTimeHandler(eventsModel, *host)

			unlockTimeHandler.Start()
		},
	}
}
