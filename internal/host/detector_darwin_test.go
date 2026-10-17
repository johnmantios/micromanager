package host

import (
	"fmt"
	"os"
	"testing"
)

func TestHelperProcessDarwinLocked(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, `"IOConsoleLocked" = Yes`)
	os.Exit(0)
}

func TestHelperProcessDarwinUnLocked(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, ``)
	os.Exit(0)
}
