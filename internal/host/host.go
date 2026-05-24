package host

import (
	"os/exec"
)

type commandRunner func(name string, arg ...string) *exec.Cmd

type Host struct {
	detector IDetector
	Username string
}

func New(cmdRunner commandRunner) *Host {
	detector := newDetector(cmdRunner)
	return &Host{
		detector: detector,
		Username: detector.GetUsername(),
	}
}
