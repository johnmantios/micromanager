package daemon

import (
	"github.com/johnmantios/micromanager/os"
	"time"
)

type Event struct {
	IsLocked bool
	Time     time.Time
}

func StartDaemon(host os.Host) <-chan Event {

	eventChannel := make(chan Event)

	go ListenForEvents(host, eventChannel)

	return eventChannel
}

func ListenForEvents(host os.Host, ch chan<- Event) {
	defer close(ch)

	ticker := time.NewTicker(5 * time.Second)

	lastTime := time.Now()

	for {
		newTime := <-ticker.C

		missingTime := newTime.Sub(lastTime).Seconds() > 5

		isLocked := host.IsLocked()

		if missingTime || isLocked {
			ch <- Event{
				IsLocked: true,
				Time:     lastTime,
			}
		} else {
			ch <- Event{
				IsLocked: false,
				Time:     newTime,
			}
		}

		lastTime = newTime
	}
}
