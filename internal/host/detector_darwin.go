//go:build darwin

package host

import (
	"errors"
	"log"
	"strings"
)

type darwinDetector struct {
	commandRunner commandRunner
}

func newDetector(cmdRunner commandRunner) IDetector {
	return &darwinDetector{commandRunner: cmdRunner}
}

func (d *darwinDetector) GetUsername() string {
	cmd := d.commandRunner("whoami")
	if cmd.Err != nil {
		panic(cmd.Err)
		//jsonlog.Fatal(cmd.Err, nil)
	}

	output, err := cmd.Output()
	if err != nil {
		log.Fatal(errors.New("error producing standard output"), nil)
	}

	return strings.TrimSuffix(string(output), "\n")
}
