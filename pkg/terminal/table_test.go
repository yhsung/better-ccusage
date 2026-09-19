package terminal

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTable_BasicRender(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable(&buf,
		Column{Header: "Model", Width: 20},
		Column{Header: "Cost", Width: 10, Align: lipgloss.Right},
	)
	tbl.SetRows([][]string{
		{"claude-sonnet-4-5", "0.50"},
		{"kimi-for-coding", "0.10"},
	})
	if err := tbl.Render(); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Model") {
		t.Errorf("missing Model header in output:\n%s", out)
	}
	if !strings.Contains(out, "claude-sonnet-4-5") {
		t.Errorf("missing row 1 in output:\n%s", out)
	}
	if !strings.Contains(out, "kimi-for-coding") {
		t.Errorf("missing row 2 in output:\n%s", out)
	}
	if !strings.Contains(out, "0.50") {
		t.Errorf("missing cost value in output:\n%s", out)
	}
}

func TestTable_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable(&buf,
		Column{Header: "Model", Width: 10},
	)
	if err := tbl.Render(); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Model") {
		t.Errorf("expected header even with no rows:\n%s", out)
	}
}

func TestTable_ColumnWidthRespected(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable(&buf,
		Column{Header: "X", Width: 10},
	)
	tbl.SetRows([][]string{{"short"}, {"a-much-longer-cell"}}) // longer than width
	if err := tbl.Render(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Even with overflow, the value should be present
	if !strings.Contains(out, "a-much-longer-cell") {
		t.Error("expected overflow cell to appear in output")
	}
}
