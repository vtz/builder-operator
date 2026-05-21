package main

import (
	"testing"
	"time"
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

func TestNewBuildCmd_HasForceFlag(t *testing.T) {
	cmd := newBuildCmd()

	f := cmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("expected --force flag")
	}
	if f.DefValue != "false" {
		t.Errorf("expected default false, got %s", f.DefValue)
	}
}

func TestBuildPollInterval(t *testing.T) {
	if buildPollInterval <= 0 {
		t.Fatal("poll interval should be positive")
	}
}

func TestBuildStartupTimeout(t *testing.T) {
	if buildStartupTimeout < 1*time.Minute {
		t.Fatal("startup timeout should be at least 1 minute")
	}
	if buildStartupTimeout > 5*time.Minute {
		t.Fatal("startup timeout should not exceed 5 minutes")
	}
}
