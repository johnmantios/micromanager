package jsonlog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type Level int8

const (
	LevelInfo Level = iota
	LevelDebug
	LevelWarning
	LevelError
	LevelFatal
	LevelOff
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarning:
		return "WARNING"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return ""
	}
}

type Logger struct {
	out      io.Writer
	minLevel Level
	mu       sync.Mutex
}

func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
	}
}

func (l *Logger) Debug(msg string, args ...any) { _, _ = l.print(LevelDebug, msg, args...) }
func (l *Logger) Info(msg string, args ...any)  { _, _ = l.print(LevelInfo, msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { _, _ = l.print(LevelWarning, msg, args...) }
func (l *Logger) Error(msg string, args ...any) { _, _ = l.print(LevelError, msg, args...) }

func (l *Logger) Fatal(err error, args ...any) {
	_, _ = l.print(LevelFatal, err.Error(), args...)
	os.Exit(1)
}

func (l *Logger) print(level Level, msg string, args ...any) (int, error) {
	if level < l.minLevel || level == LevelOff {
		return 0, nil
	}

	aux := struct {
		Level      string `json:"level"`
		Time       string `json:"time"`
		Message    string `json:"msg"`
		Properties string `json:"args,omitempty"`
		Trace      string `json:"trace,omitempty"`
	}{
		Level:      level.String(),
		Time:       time.Now().UTC().Format(time.RFC3339),
		Message:    strings.TrimSpace(msg),
		Properties: logFormatArgs(args...),
	}

	if level >= LevelError {
		aux.Trace = string(debug.Stack())
	}

	line, err := json.Marshal(aux)
	if err != nil {
		line = []byte(`{"level":"ERROR","msg":"unable to marshal log message"}` + "\n")
	} else {
		line = append(line, '\n')
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	return l.out.Write(line)
}

func (l *Logger) Write(message []byte) (n int, err error) {
	return l.print(LevelInfo, strings.TrimSpace(string(message)))
}

func (l *Logger) Print(v ...any) {
	_, _ = l.print(LevelInfo, fmt.Sprint(v...))
}

func logFormatArgs(args ...any) string {
	if len(args) == 0 {
		return ""
	}
	if len(args)%2 != 0 {
		return fmt.Sprint(args...)
	}

	pairs := make([]string, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		pairs = append(pairs, fmt.Sprintf("%v=%v", args[i], args[i+1]))
	}
	return strings.Join(pairs, ", ")
}
