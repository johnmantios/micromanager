package unlocked_time

import (
	"os/exec"
)

type commandRunner func(name string, arg ...string) *exec.Cmd

type Host struct {
	detector IDetector
	UserID   string
}

func NewHost(cmdRunner commandRunner) *Host {
	detector := newDetector(cmdRunner)
	return &Host{
		detector: detector,
		UserID:   detector.GetUsername(),
	}
}

func (h *Host) IsLocked() bool {
	return h.detector.IsLocked()
}
