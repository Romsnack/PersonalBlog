package main

import (
	"os"
	"strings"
	"testing"
)

// run() reads paths relative to the working directory and its FlagSet is
// ExitOnError, so the build and serve branches are covered by the CI build step
// rather than from here. The dispatch error is reachable without touching the
// filesystem.
func TestRunRejectsAnUnknownCommand(t *testing.T) {
	saved := os.Args
	t.Cleanup(func() { os.Args = saved })
	os.Args = []string{"blog", "publish"}

	err := run()
	if err == nil {
		t.Fatal("want an error for an unknown command, got nil")
	}
	if !strings.Contains(err.Error(), "publish") {
		t.Errorf("error = %q, want it to name the offending command", err)
	}
}
