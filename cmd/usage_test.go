package cmd

import (
	"strings"
	"testing"
)

func TestEnvironmentCommandUsage(t *testing.T) {
	if got := listCmd.UseLine(); !strings.HasSuffix(got, " list") {
		t.Errorf("list usage = %q", got)
	}
	if got := deleteCmd.UseLine(); !strings.HasSuffix(got, " delete <name>") {
		t.Errorf("delete usage = %q", got)
	}
}
