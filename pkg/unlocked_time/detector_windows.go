//go:build windows

package unlocked_time

import (
	log "github.com/sirupsen/logrus"
)

type windowsDetector struct {
	commandRunner commandRunner
}

func newDetector(cmdRunner commandRunner) IDetector {
	return &windowsDetector{commandRunner: cmdRunner}
}

func (d *windowsDetector) IsLocked() bool {
	log.Info("windows is unimplemented")
	return false
}

func (d *windowsDetector) GetUsername() string {
	// TODO: implement for Windows
	return "unknown"
}
