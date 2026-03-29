package unlocked_time

import (
	"fmt"
	"github.com/johnmantios/micromanager/internal/assert"
	"os"
	"os/exec"
	"testing"
)

func fakeCommandDarwinLocked(name string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcessDarwinLocked", "--", name}
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestHelperProcessDarwinLocked(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, `"IOConsoleLocked" = Yes`)
	os.Exit(0)
}

func fakeCommandDarwinUnLocked(name string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcessDarwinUnLocked", "--", name}
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestHelperProcessDarwinUnLocked(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, ``)
	os.Exit(0)
}

func TestDarwinIsLocked(t *testing.T) {
	cases := []struct {
		name     string
		host     Host
		expected bool
	}{
		{
			name: "Is not locked",
			host: Host{
				detector: newDetector(fakeCommandDarwinUnLocked),
				UserID:   "",
			},
			expected: false,
		},
		{
			name: "Is locked",
			host: Host{
				detector: newDetector(fakeCommandDarwinLocked),
				UserID:   "",
			},
			expected: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {

			actual := tt.host.detector.IsLocked()

			assert.Equal(t, actual, tt.expected)
		})
	}
}
