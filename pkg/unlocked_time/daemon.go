package unlocked_time

import (
	"github.com/johnmantios/micromanager/repo"
	log "github.com/sirupsen/logrus"
	"time"
)

func StartDaemon(host Host) <-chan repo.EventEntity {

	eventChannel := make(chan repo.EventEntity)

	log.Debug("StartDaemon: launching goroutine")
	go ListenForEvents(host, eventChannel)

	return eventChannel
}

func ListenForEvents(host Host, ch chan<- repo.EventEntity) {
	defer close(ch)

	log.Debug("ListenForEvents: goroutine started")

	ticker := time.NewTicker(1 * time.Second)

	lastTime := time.Now().UTC()

	for {
		newTime := <-ticker.C

		isLocked := host.IsLocked()

		if isLocked {
			ch <- repo.EventEntity{
				IsLocked: true,
				Tick:     lastTime,
				UserID:   host.UserID,
			}
		} else {
			ch <- repo.EventEntity{
				IsLocked: false,
				Tick:     newTime.UTC(),
				UserID:   host.UserID,
			}
		}

		lastTime = newTime
	}
}
