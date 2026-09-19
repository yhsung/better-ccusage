package terminal

import (
	"regexp"
	"strings"
)

// blankLineRun matches 3+ consecutive newlines for collapse.
var blankLineRun = regexp.MustCompile(`\n{3,}`)

// NormalizeForTest prepares a string for stable golden-file comparison:
//  1. Strips ANSI escape sequences
//  2. Normalizes CRLF and CR to LF
//  3. Trims trailing whitespace per line
//  4. Collapses runs of 3+ blank lines to 2 (one blank line)
//
// Trailing-newline status is preserved: input without a trailing newline
// produces output without one; blank-only input collapses to a single "\n".
func NormalizeForTest(s string) string {
	if s == "" {
		return ""
	}
	s = StripANSI(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = TrimTrailingSpaces(s)
	s = blankLineRun.ReplaceAllString(s, "\n\n")
	// Trim leading blank lines.
	s = strings.TrimLeft(s, "\n")
	// Preserve blank-only input as a single newline so the result is non-empty
	// and still ends with a newline (helps golden-file diffs).
	if s == "" {
		return "\n"
	}
	return s
}