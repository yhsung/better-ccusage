package terminal

import (
	"strings"
	"testing"
)

func TestStyleText_NotEmpty(t *testing.T) {
	cases := []Style{StyleHeader, StyleMuted, StyleCost, StyleWarning, StyleError, StyleSuccess}
	for _, s := range cases {
		t.Run("style", func(t *testing.T) {
			got := StyleText(s, "hello")
			if got == "" {
				t.Errorf("StyleText(%d) returned empty", s)
			}
			if !strings.Contains(got, "hello") {
				t.Errorf("StyleText(%d) lost the input text", s)
			}
		})
	}
}

func TestStyleText_EmptyInput(t *testing.T) {
	got := StyleText(StyleHeader, "")
	if got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
}

func TestStripANSI(t *testing.T) {
	in := "\x1b[31mred\x1b[0m plain \x1b[1;32mgreen\x1b[0m"
	want := "red plain green"
	if got := StripANSI(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}