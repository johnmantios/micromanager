//go:build windows

package host

type windowsDetector struct {
	commandRunner commandRunner
}

func newDetector(cmdRunner commandRunner) IDetector {
	return &windowsDetector{commandRunner: cmdRunner}
}

func (d *windowsDetector) IsLocked() bool {
	panic("windows is unimplemented")
}

func (d *windowsDetector) GetUsername() string {
	// TODO: implement for Windows
	return "unknown"
}
