package terminal

import (
	"strings"
	"testing"
)

func TestNormalizeForTest_StripsANSI(t *testing.T) {
	in := "\x1b[31mred\x1b[0m"
	want := "red"
	if got := NormalizeForTest(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeForTest_NormalizesLineEndings(t *testing.T) {
	in := "a\r\nb\rc\n"
	want := "a\nb\nc\n"
	if got := NormalizeForTest(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeForTest_TrimsTrailingWhitespace(t *testing.T) {
	in := "a   \nb\t\t\nc\n"
	want := "a\nb\nc\n"
	if got := NormalizeForTest(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeForTest_CollapsesBlankLines(t *testing.T) {
	in := "a\n\n\n\n\nb\n"
	want := "a\n\nb\n"
	if got := NormalizeForTest(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeForTest_Empty(t *testing.T) {
	if got := NormalizeForTest(""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
	if got := NormalizeForTest(strings.Repeat("\n", 5)); got != "\n" {
		t.Errorf("expected single newline for blank-only input, got %q", got)
	}
}