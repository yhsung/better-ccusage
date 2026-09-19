package commands

import (
	"testing"
)

func TestNewBlocksLiveCmd_Flags(t *testing.T) {
	cmd, opts := NewBlocksLiveCmd(testPriceTable(t))
	if cmd.Use != "blocks.live" {
		t.Errorf("expected Use blocks.live, got %q", cmd.Use)
	}
	if cmd.Flags().Lookup("token-limit") == nil {
		t.Error("expected --token-limit flag to be registered")
	}
	_ = opts
}
