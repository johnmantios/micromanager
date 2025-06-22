package daemon

import (
	"github.com/johnmantios/micromanager/os"
	"github.com/johnmantios/micromanager/repo"
	"time"
)

func StartDaemon(host os.Host) <-chan repo.Event {

	eventChannel := make(chan repo.Event)

	go ListenForEvents(host, eventChannel)

	return eventChannel
}

func ListenForEvents(host os.Host, ch chan<- repo.Event) {
	defer close(ch)

	ticker := time.NewTicker(1 * time.Second)

	lastTime := time.Now().UTC()

	for {
		newTime := <-ticker.C

		isLocked := host.IsLocked()

		if isLocked {
			ch <- repo.Event{
				IsLocked: true,
				Tick:     lastTime,
				UserID:   "john",
			}
		} else {
			ch <- repo.Event{
				IsLocked: false,
				Tick:     newTime.UTC(),
				UserID:   "john",
			}
		}

		lastTime = newTime
	}
}
