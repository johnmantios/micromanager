package main

import (
	"fmt"
	"github.com/johnmantios/micromanager/daemon"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	micromanagerOS "github.com/johnmantios/micromanager/os"
	repo2 "github.com/johnmantios/micromanager/repo"
	repository "github.com/johnmantios/micromanager/repo/timescale"
	_ "github.com/lib/pq"
	"os"
	"runtime"
)

func main() {
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	host := micromanagerOS.Host{Logger: logger}

	host.UserID = host.WhoAmI()

	logger.PrintInfo("Starting micromanagement...", map[string]string{
		"os": runtime.GOOS,
	})

	db, err := repository.OpenDB()
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	repo, err := repository.NewTimescaleRepo(db)
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	previous := repo2.Event{
		IsLocked: true,
	}

	for event := range daemon.StartDaemon(host) {
		if event.IsLocked != previous.IsLocked {
			err = repo.Event.SaveTick(event)
			if err != nil {
				logger.PrintWarning("could not save tick!", map[string]string{
					"error": err.Error(),
				})
			}
			logger.PrintInfo("status", map[string]string{
				"user":      host.UserID,
				"is locked": fmt.Sprintf("%t", event.IsLocked),
			})
			previous = event
		} else {
			continue
		}

	}

}
