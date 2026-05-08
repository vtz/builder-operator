// Copyright 2026 Red Hat Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"compress/gzip"
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func TestTarDirectory_CreatesArchive(t *testing.T) {
	dir := t.TempDir()
	if err := writeFile(filepath.Join(dir, "hello.txt"), []byte("hello")); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(subDir, "world.txt"), []byte("world")); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	count, err := tarDirectory(dir, &buf)
	if err != nil {
		t.Fatalf("tarDirectory failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 files, got %d", count)
	}

	gr, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	names := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reading tar: %v", err)
		}
		names[hdr.Name] = true
	}
	if !names["hello.txt"] {
		t.Error("hello.txt not found in archive")
	}
	if !names[filepath.Join("subdir", "world.txt")] {
		t.Error("subdir/world.txt not found in archive")
	}
}

func TestTarDirectory_ExcludesGitDir(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main")); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(dir, "main.go"), []byte("package main")); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	count, err := tarDirectory(dir, &buf)
	if err != nil {
		t.Fatalf("tarDirectory failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 file (main.go only), got %d", count)
	}
}

func TestTarDirectory_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	count, err := tarDirectory(dir, &buf)
	if err != nil {
		t.Fatalf("tarDirectory failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 files, got %d", count)
	}
}

func TestShouldSkipPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{".git", true},
		{"node_modules", true},
		{"__pycache__", true},
		{".tox", true},
		{".venv", true},
		{"build", true},
		{".bob-output", true},
		{"src", false},
		{"main.go", false},
		{"Makefile", false},
		{"README.md", false},
		{filepath.Join("src", ".git"), true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := shouldSkipPath(tt.path)
			if got != tt.want {
				t.Errorf("shouldSkipPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDetectKubeClient(t *testing.T) {
	result := detectKubeClient()
	// Can't assert specific value since it depends on the test environment.
	t.Logf("detectKubeClient() = %q", result)
}
