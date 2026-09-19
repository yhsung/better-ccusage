// Package shim implements deprecated forwarder binaries that delegate to
// better-ccusage. Each shim is a thin main supplying a Config.
package shim

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Config parameterizes one shim binary.
type Config struct {
	// Target is the binary to exec, e.g. "better-ccusage".
	Target string
	// OwnName is this shim's binary name, stripped from argv (npx scenario).
	OwnName string
	// NoticeLines are written to stderr on TTY runs (already prefixed).
	NoticeLines []string
	// OptOutEnv, when "1", suppresses the notice (e.g. CODEX_NO_DEPRECATION_NOTICE).
	OptOutEnv string
	// Set env vars are always applied.
	Set map[string]string
	// SuppressIfUnset entries apply only when the parent env lacks the key
	// or holds an empty value.
	SuppressIfUnset map[string]string
}

// NonexistentTmp returns a path inside os.TempDir that must not exist,
// used to neuter unrelated data-dir env vars.
func NonexistentTmp(prefix string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("nonexistent-%s-%d", prefix, os.Getpid()))
}

// ResolveBinary locates target on PATH, falling back to the shim's own
// directory. It never falls back silently: missing binaries are an error.
func ResolveBinary(name string) (string, error) {
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), name)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return p, nil
		}
	}
	return "", fmt.Errorf("could not resolve %q: not on PATH nor next to this binary", name)
}

// Run forwards args to the target and returns the process exit code.
// It never calls os.Exit itself so mains stay testable.
func Run(cfg Config, args []string) int {
	filtered := make([]string, 0, len(args))
	for _, a := range args {
		if a != cfg.OwnName {
			filtered = append(filtered, a)
		}
	}
	printNotice(os.Stderr, cfg, isCharDevice(os.Stdout))
	bin, err := ResolveBinary(cfg.Target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cmd := exec.Command(bin, filtered...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = buildEnv(os.Environ(), cfg)
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func printNotice(w io.Writer, cfg Config, isTTY bool) {
	if !isTTY {
		return
	}
	if os.Getenv(cfg.OptOutEnv) == "1" {
		return
	}
	for _, line := range cfg.NoticeLines {
		fmt.Fprintln(w, line)
	}
}

func isCharDevice(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func buildEnv(parent []string, cfg Config) []string {
	out := make([]string, 0, len(parent)+len(cfg.Set)+len(cfg.SuppressIfUnset))
	out = append(out, parent...)
	set := func(k, v string) {
		prefix := k + "="
		for i, e := range out {
			if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
				out[i] = prefix + v
				return
			}
		}
		out = append(out, prefix+v)
	}
	for k, v := range cfg.Set {
		set(k, v)
	}
	for k, v := range cfg.SuppressIfUnset {
		if os.Getenv(k) == "" {
			set(k, v)
		}
	}
	return out
}
