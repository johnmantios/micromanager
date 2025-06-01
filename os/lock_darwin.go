package os

import (
	"errors"
	"strings"
)

func (h *Host) isLocked() bool {
	cmd := h.getCommandRunner()("ioreg", "-n", "Root", "-d1")
	if cmd.Err != nil {
		h.Logger.PrintError(cmd.Err, nil)
	}

	output, err := cmd.Output()
	if err != nil {
		h.Logger.PrintError(errors.New("error producing standard output"), nil)
	}

	return strings.Contains(string(output), "CGSSessionScreenIsLocked")

}
