//go:build darwin

package unlocked_time

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"strings"
)

type darwinDetector struct {
	commandRunner commandRunner
}

func newDetector(cmdRunner commandRunner) IDetector {
	return &darwinDetector{commandRunner: cmdRunner}
}

func (d *darwinDetector) IsLocked() bool {
	cmd := d.commandRunner("ioreg", "-n", "Root", "-d1")
	if cmd.Err != nil {
		log.Fatal(cmd.Err, nil)
	}

	output, err := cmd.Output()
	if err != nil {
		log.WithError(err).Error("isLocked: command failed")
	}

	result := strings.Contains(string(output), "\"IOConsoleLocked\" = Yes")
	log.Info(result)
	return result

}

func (d *darwinDetector) GetUsername() string {
	cmd := d.commandRunner("whoami")
	if cmd.Err != nil {
		log.Fatal(cmd.Err, nil)
	}

	output, err := cmd.Output()
	if err != nil {
		log.Fatal(errors.New("error producing standard output"), nil)
	}

	return strings.TrimSuffix(string(output), "\n")
}
