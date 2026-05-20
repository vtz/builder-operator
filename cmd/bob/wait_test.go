package main

import (
	"testing"
)

func TestNewBuildCmd_HasDownloadFlag(t *testing.T) {
	cmd := newBuildCmd()

	d := cmd.Flags().Lookup("download")
	if d == nil {
		t.Fatal("expected --download flag")
	}
	if d.Shorthand != "d" {
		t.Errorf("expected shorthand d, got %s", d.Shorthand)
	}

	sv := cmd.Flags().Lookup("skip-verify")
	if sv == nil {
		t.Fatal("expected --skip-verify flag")
	}
}

func TestBuildPollInterval(t *testing.T) {
	if buildPollInterval <= 0 {
		t.Fatal("poll interval should be positive")
	}
}
