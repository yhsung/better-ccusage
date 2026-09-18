// Package terminal provides shared CLI rendering primitives (tables, colors,
// loggers) for better-ccusage Go binaries. Mirrors the TS packages/terminal
// package and the cli-table3 + consola output.
package terminal

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"
)

type Level int

const (
	Silent Level = iota
	Warn
	Log
	Info
	Debug
	Trace
)

// Logger writes leveled messages to an io.Writer. Concurrent-safe.
type Logger struct {
	mu  sync.Mutex
	w   io.Writer
	lvl Level
}

// NewLogger returns a Logger that writes to w at the given level.
// Pass Silent (0) to disable all output.
func NewLogger(level int, w io.Writer) *Logger {
	if w == nil {
		w = io.Discard
	}
	return &Logger{w: w, lvl: Level(level)}
}

// NewLoggerFromEnv parses LOG_LEVEL and returns a Logger writing to stderr.
func NewLoggerFromEnv() *Logger {
	return NewLogger(parseLevel(os.Getenv("LOG_LEVEL")), os.Stderr)
}

// Level returns the configured log level.
func (l *Logger) Level() Level { return l.lvl }

func parseLevel(s string) int {
	if s == "" {
		return int(Silent)
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 || v > int(Trace) {
		return int(Silent)
	}
	return v
}

// logf writes a message at the given level if level <= l.lvl.
func (l *Logger) logf(level Level, tag, format string, args ...any) {
	if level > l.lvl || l.w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.w, "%s %s\n", tag, msg)
}

// Warn logs at Warn level.
func (l *Logger) Warn(format string, args ...any) { l.logf(Warn, "WARN ", format, args...) }

// Info logs at Info level.
func (l *Logger) Info(format string, args ...any) { l.logf(Info, "INFO ", format, args...) }

// Log logs at Log level (consola "log" mapping).
func (l *Logger) Log(format string, args ...any) { l.logf(Log, "LOG  ", format, args...) }

// Debug logs at Debug level.
func (l *Logger) Debug(format string, args ...any) { l.logf(Debug, "DEBUG", format, args...) }

// Trace logs at Trace level.
func (l *Logger) Trace(format string, args ...any) { l.logf(Trace, "TRACE", format, args...) }

// Fatal logs the message at Warn level and exits with code 1.
func (l *Logger) Fatal(format string, args ...any) {
	l.Warn(format, args...)
	os.Exit(1)
}

// WithTimestamp returns a copy of the logger that prefixes each message with
// an RFC3339 timestamp. Used by tests; not enabled by default.
func (l *Logger) WithTimestamp() *Logger {
	// Not used in production output — kept for future expansion.
	_ = time.RFC3339
	return l
}
