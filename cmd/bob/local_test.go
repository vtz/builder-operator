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
	"testing"
)

func TestDetectContainerRuntime_ReturnsPath(t *testing.T) {
	rt := detectContainerRuntime()
	// We can't assert a specific path since it depends on the test environment,
	// but on most dev machines at least one of podman/docker is present.
	// This test primarily validates the function doesn't panic.
	t.Logf("detectContainerRuntime() = %q", rt)
}

func TestRunLocalBuild_MissingFile(t *testing.T) {
	err := runLocalBuild("/nonexistent/file.yaml", "", ".", t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRunLocalBuild_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	yamlPath := dir + "/bad.yaml"
	if err := writeTestFile(yamlPath, "not: valid: yaml: [[["); err != nil {
		t.Fatal(err)
	}
	err := runLocalBuild(yamlPath, "", ".", dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestRunLocalBuild_NoToolchainImage(t *testing.T) {
	dir := t.TempDir()
	yamlPath := dir + "/no-image.yaml"
	content := `apiVersion: builder.sdv.cloud.redhat.com/v1alpha1
kind: BuildJob
metadata:
  name: test
spec:
  source:
    type: git
  stages:
    - name: build
      command: make
`
	if err := writeTestFile(yamlPath, content); err != nil {
		t.Fatal(err)
	}
	err := runLocalBuild(yamlPath, "", ".", dir)
	if err == nil {
		t.Fatal("expected error for missing toolchain image")
	}
}

func TestRunLocalBuild_SourceDirNotExist(t *testing.T) {
	dir := t.TempDir()
	yamlPath := dir + "/bj.yaml"
	content := `apiVersion: builder.sdv.cloud.redhat.com/v1alpha1
kind: BuildJob
metadata:
  name: test
spec:
  toolchain:
    image: ubuntu:24.04
  source:
    type: git
  stages:
    - name: build
      command: make
`
	if err := writeTestFile(yamlPath, content); err != nil {
		t.Fatal(err)
	}
	err := runLocalBuild(yamlPath, "", "/nonexistent/source/dir", dir)
	if err == nil {
		t.Fatal("expected error for nonexistent source directory")
	}
}

func writeTestFile(path, content string) error {
	return writeFile(path, []byte(content))
}
