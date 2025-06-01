package main

import (
	"fmt"
	"github.com/johnmantios/micromanager/daemon"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	micromanagerOS "github.com/johnmantios/micromanager/os"
	repository "github.com/johnmantios/micromanager/repo/timescale"
	"os"
	"runtime"
)

func main() {
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	host := micromanagerOS.Host{Logger: logger}

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

	for event := range daemon.StartDaemon(host) {
		err = repo.Event.SaveTick(event)
		if err != nil {
			logger.PrintWarning("could not save tick!", map[string]string{
				"error": err.Error(),
			})
		}
		logger.PrintInfo("status", map[string]string{
			"is locked": fmt.Sprintf("%t", event.IsLocked),
		})
	}

}
