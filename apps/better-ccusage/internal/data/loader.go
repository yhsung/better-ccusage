package data

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Loader reads JSONL entries from one or more Claude config directories.
type Loader struct {
	dirs []string
}

// NewLoader constructs a Loader with the given config directories.
// Each dir is expected to contain projects/*/*.jsonl (Claude Code log layout).
func NewLoader(dirs ...string) *Loader {
	return &Loader{dirs: append([]string(nil), dirs...)}
}

// Load walks all configured dirs, parses each .jsonl line into an Entry,
// and returns them sorted by timestamp ascending. Malformed lines are
// silently skipped (matches TS data-loader.ts).
func (l *Loader) Load(ctx context.Context) ([]Entry, error) {
	if len(l.dirs) == 0 {
		return nil, fmt.Errorf("no data directories configured")
	}
	var entries []Entry
	for _, dir := range l.dirs {
		if err := l.walkDir(ctx, dir, &entries); err != nil {
			return nil, fmt.Errorf("walking %s: %w", dir, err)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})
	return entries, nil
}

// Reload re-reads files whose mtime changed since the previous load.
// For simplicity in v1, Reload calls Load and returns the full result.
// Incremental mtime filtering can be added later.
func (l *Loader) Reload(ctx context.Context, prev []Entry) ([]Entry, error) {
	return l.Load(ctx)
}

func (l *Loader) walkDir(ctx context.Context, root string, out *[]Entry) error {
	projectsDir := filepath.Join(root, "projects")
	return filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if info.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := parseJSONL(f, out); err != nil {
			return err
		}
		return nil
	})
}

func parseJSONL(r io.Reader, out *[]Entry) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024) // up to 16 MiB per line
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			// silently skip malformed lines, matching TS behavior
			continue
		}
		*out = append(*out, e)
	}
	return scanner.Err()
}

// DefaultDirs returns the default Claude config directories in lookup order.
func DefaultDirs() []string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}
	return []string{
		filepath.Join(home, ".config", "claude"),
		filepath.Join(home, ".claude"),
	}
}

// FilterByTime returns entries with Timestamp in [since, until].
// Either bound may be nil for open-ended ranges.
func FilterByTime(entries []Entry, since, until *time.Time) []Entry {
	var out []Entry
	for _, e := range entries {
		if since != nil && e.Timestamp.Before(*since) {
			continue
		}
		if until != nil && e.Timestamp.After(*until) {
			continue
		}
		out = append(out, e)
	}
	return out
}