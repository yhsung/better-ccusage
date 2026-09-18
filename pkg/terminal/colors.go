package terminal

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Style is a named text style matching the consola color tags used in TS.
type Style int

const (
	StyleHeader Style = iota
	StyleMuted
	StyleCost
	StyleWarning
	StyleError
	StyleSuccess
)

var styles = map[Style]lipgloss.Style{
	StyleHeader:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),  // bright blue
	StyleMuted:   lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("245")), // gray
	StyleCost:    lipgloss.NewStyle().Foreground(lipgloss.Color("10")),              // green
	StyleWarning: lipgloss.NewStyle().Foreground(lipgloss.Color("11")),              // yellow
	StyleError:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")),    // red
	StyleSuccess: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")),   // green
}

// StyleText returns the styled form of text. Returns "" for empty input so
// callers can chain without checking.
func StyleText(s Style, text string) string {
	if text == "" {
		return ""
	}
	return styles[s].Render(text)
}

// ansiEscape matches ANSI CSI sequences for stripping in tests and NormalizeForTest.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes all ANSI escape sequences from s.
func StripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// TrimTrailingSpaces removes trailing spaces from each line of s. Used by
// normalize helpers to make golden files stable across terminal widths.
func TrimTrailingSpaces(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}