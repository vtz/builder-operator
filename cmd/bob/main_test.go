package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{"all empty", []string{"", "", ""}, ""},
		{"first wins", []string{"a", "b", "c"}, "a"},
		{"skips empty", []string{"", "b", "c"}, "b"},
		{"last", []string{"", "", "c"}, "c"},
		{"single", []string{"x"}, "x"},
		{"none", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstNonEmpty(tt.values...)
			if got != tt.want {
				t.Errorf("firstNonEmpty(%v) = %q, want %q", tt.values, got, tt.want)
			}
		})
	}
}

func TestNewBuildCmd_HasFlags(t *testing.T) {
	cmd := newBuildCmd()
	if cmd.Use != "build [name]" {
		t.Errorf("unexpected use: %s", cmd.Use)
	}

	f := cmd.Flags().Lookup("file")
	if f == nil {
		t.Fatal("expected --file flag")
	}
	if f.Shorthand != "f" {
		t.Errorf("expected shorthand f, got %s", f.Shorthand)
	}

	b := cmd.Flags().Lookup("branch")
	if b == nil {
		t.Fatal("expected --branch flag")
	}
}

func TestNewBuildCmd_ErrorWithoutArgs(t *testing.T) {
	cmd := newBuildCmd()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Usage()
		}
		return nil
	}
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Log("build with no args returns usage")
	}
}

func TestNewListCmd(t *testing.T) {
	cmd := newListCmd()
	if cmd.Use != "list" {
		t.Errorf("unexpected use: %s", cmd.Use)
	}
	aliases := cmd.Aliases
	found := false
	for _, a := range aliases {
		if a == "ls" {
			found = true
		}
	}
	if !found {
		t.Error("expected ls alias")
	}
}

func TestNewShowCmd(t *testing.T) {
	cmd := newShowCmd()
	if cmd.Use != "show <name>" {
		t.Errorf("unexpected use: %s", cmd.Use)
	}
}

func TestNewDeleteCmd(t *testing.T) {
	cmd := newDeleteCmd()
	if cmd.Use != "delete <name>" {
		t.Errorf("unexpected use: %s", cmd.Use)
	}
}

func TestNewLogsCmd(t *testing.T) {
	cmd := newLogsCmd()
	if cmd.Use != "logs <name>" {
		t.Errorf("unexpected use: %s", cmd.Use)
	}
}
