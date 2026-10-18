//go:build unit

package jsonlog

import (
	"bytes"
	"strings"
	"testing"
)

// TestLoggerRespectsMinLevel verifies that only messages at or above the
// configured minimum level are emitted, in particular that the default Info
// level suppresses debug output.
func TestLoggerRespectsMinLevel(t *testing.T) {
	tests := []struct {
		name     string
		minLevel Level
		log      func(*Logger)
		want     []string
		notWant  []string
	}{
		{
			name:     "info level suppresses debug",
			minLevel: LevelInfo,
			log: func(l *Logger) {
				l.Debug("debug message")
				l.Info("info message")
				l.Warn("warn message")
			},
			want:    []string{`"level":"INFO"`, `"msg":"info message"`, `"level":"WARNING"`},
			notWant: []string{`"level":"DEBUG"`, "debug message"},
		},
		{
			name:     "debug level includes debug",
			minLevel: LevelDebug,
			log: func(l *Logger) {
				l.Debug("debug message")
			},
			want: []string{`"level":"DEBUG"`, "debug message"},
		},
		{
			name:     "error level suppresses info and warnings",
			minLevel: LevelError,
			log: func(l *Logger) {
				l.Info("info message")
				l.Warn("warn message")
				l.Error("error message")
			},
			want:    []string{`"level":"ERROR"`, `"msg":"error message"`},
			notWant: []string{"info message", "warn message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := New(&buf, tt.minLevel)

			tt.log(logger)

			out := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("output %q does not contain %q", out, want)
				}
			}
			for _, notWant := range tt.notWant {
				if strings.Contains(out, notWant) {
					t.Errorf("output %q should not contain %q", out, notWant)
				}
			}
		})
	}
}
