package terminal

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogger_RespectsLevel(t *testing.T) {
	cases := []struct {
		name      string
		level     int
		wantWarn  bool
		wantLog   bool
		wantInfo  bool
		wantDebug bool
		wantTrace bool
	}{
		{"Silent_0_blocks_all", 0, false, false, false, false, false},
		{"Warn_1_allows_warn_only", 1, true, false, false, false, false},
		{"Log_2_allows_warn_log", 2, true, true, false, false, false},
		{"Info_3_allows_info", 3, true, true, true, false, false},
		{"Debug_4_allows_debug", 4, true, true, true, true, false},
		{"Trace_5_allows_all", 5, true, true, true, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := NewLogger(tc.level, &buf)
			l.Warn("w")
			l.Log("l")
			l.Info("i")
			l.Debug("d")
			l.Trace("t")
			out := buf.String()
			if got := strings.Contains(out, "w"); got != tc.wantWarn {
				t.Errorf("warn: got %v, want %v", got, tc.wantWarn)
			}
			if got := strings.Contains(out, "l"); got != tc.wantLog {
				t.Errorf("log: got %v, want %v", got, tc.wantLog)
			}
			if got := strings.Contains(out, "i"); got != tc.wantInfo {
				t.Errorf("info: got %v, want %v", got, tc.wantInfo)
			}
			if got := strings.Contains(out, "d"); got != tc.wantDebug {
				t.Errorf("debug: got %v, want %v", got, tc.wantDebug)
			}
			if got := strings.Contains(out, "t"); got != tc.wantTrace {
				t.Errorf("trace: got %v, want %v", got, tc.wantTrace)
			}
		})
	}
}

func TestLogger_OutputFormat(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger(int(Info), &buf)
	l.Info("hello %s", "world")
	out := buf.String()
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected formatted message, got %q", out)
	}
	// Format: "INFO  hello world\n" — level prefix + space + msg + newline
	if !strings.HasPrefix(out, "INFO ") {
		t.Errorf("expected INFO prefix, got %q", out)
	}
}

func TestLogger_NilWriterDoesNotPanic(t *testing.T) {
	l := NewLogger(int(Trace), nil)
	l.Info("noop")
	// Just verifying no panic
}

func TestNewLoggerFromEnv(t *testing.T) {
	cases := []struct {
		env  string
		want Level
	}{
		{"", Silent},
		{"0", Silent},
		{"1", Warn},
		{"3", Info},
		{"5", Trace},
		{"invalid", Silent}, // graceful fallback
	}
	for _, tc := range cases {
		t.Run("env="+tc.env, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", tc.env)
			if got := NewLoggerFromEnv().Level(); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}
