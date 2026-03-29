package handler

import (
	"fmt"
	unlockedTime "github.com/johnmantios/micromanager/pkg/unlocked_time"
	"github.com/johnmantios/micromanager/repo"
	log "github.com/sirupsen/logrus"
	"runtime"
)

type UnlockedScreenTime struct {
	repo repo.IEventsModel
	host unlockedTime.Host
}

func NewUnlockedScreenTimeHandler(
	repo repo.IEventsModel,
	host unlockedTime.Host,
) *UnlockedScreenTime {
	return &UnlockedScreenTime{
		repo: repo,
		host: host,
	}
}

func (ut *UnlockedScreenTime) Start() {
	log.Info("Starting screen micromanagement...", map[string]string{
		"os": runtime.GOOS,
	})

	previous := repo.EventEntity{
		IsLocked: true,
	}

	for event := range unlockedTime.StartDaemon(ut.host) {
		if event.IsLocked != previous.IsLocked {
			err := ut.repo.SaveTick(event)
			if err != nil {
				log.Warning("could not save tick!", map[string]string{
					"error": err.Error(),
				})
			}
			log.Info("status", map[string]string{
				"user":      ut.host.UserID,
				"is locked": fmt.Sprintf("%t", event.IsLocked),
			})
			previous = event
		}
	}
}
