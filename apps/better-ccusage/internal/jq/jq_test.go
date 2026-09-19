package jq

import (
	"strings"
	"testing"
)

func TestProcess_Identity(t *testing.T) {
	out, err := Process(".", []byte(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"a"`) {
		t.Errorf("expected identity output, got %q", out)
	}
}

func TestProcess_FieldAccess(t *testing.T) {
	out, err := Process(".a", []byte(`{"a":42}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "42" {
		t.Errorf("expected '42', got %q", out)
	}
}
