package main

import (
	"fmt"
	"github.com/johnmantios/micromanager/daemon"
	"github.com/johnmantios/micromanager/internal/jsonlog"
	micromanagerOS "github.com/johnmantios/micromanager/os"
	"os"
	"runtime"
)

func main() {
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	host := micromanagerOS.Host{Logger: logger}

	logger.PrintInfo("Starting micromanagement...", map[string]string{
		"os": runtime.GOOS,
	})

	for event := range daemon.StartDaemon(host) {
		logger.PrintInfo("status", map[string]string{
			"is locked": fmt.Sprintf("%t", event.IsLocked),
		})
	}

}
