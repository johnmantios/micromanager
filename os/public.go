package os

import (
	"github.com/johnmantios/micromanager/internal/jsonlog"
	"os/exec"
)

type commandRunner func(name string, arg ...string) *exec.Cmd

type Host struct {
	Logger        *jsonlog.Logger
	commandRunner commandRunner
	UserID        string
}

func (h *Host) IsLocked() bool {
	return h.isLocked()
}

func (h *Host) WhoAmI() string {
	return h.whoAmI()
}

func (h *Host) getCommandRunner() commandRunner {
	if h.commandRunner != nil {
		return h.commandRunner
	}
	return exec.Command
}
