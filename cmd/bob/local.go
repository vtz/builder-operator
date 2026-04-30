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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func detectContainerRuntime() string {
	if p, err := exec.LookPath("podman"); err == nil {
		return p
	}
	if p, err := exec.LookPath("docker"); err == nil {
		return p
	}
	return ""
}

func runLocalBuild(file, branch, sourceDir, outputDir string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	var bj buildv1alpha1.BuildJob
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	if err := decoder.Decode(&bj); err != nil {
		return fmt.Errorf("parsing YAML: %w", err)
	}

	if branch != "" && bj.Spec.Source.Git != nil {
		bj.Spec.Source.Git.Revision = branch
	}

	runtime := detectContainerRuntime()
	if runtime == "" {
		return fmt.Errorf("no container runtime found (install podman or docker)")
	}

	image := bj.Spec.Toolchain.Image
	if image == "" {
		return fmt.Errorf("no toolchain image specified in BuildJob")
	}

	absSource, err := filepath.Abs(sourceDir)
	if err != nil {
		return fmt.Errorf("resolving source directory: %w", err)
	}
	if _, err := os.Stat(absSource); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", absSource)
	}

	absOutput, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("resolving output directory: %w", err)
	}
	if err := os.MkdirAll(absOutput, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	fmt.Printf("Local build: %s\n", bj.Name)
	fmt.Printf("  Image:   %s\n", image)
	fmt.Printf("  Source:  %s\n", absSource)
	fmt.Printf("  Output:  %s\n", absOutput)
	fmt.Printf("  Runtime: %s\n", filepath.Base(runtime))
	fmt.Println()

	var scriptParts []string
	scriptParts = append(scriptParts, "set -euo pipefail")
	scriptParts = append(scriptParts, "cd /workspace")

	for i, stage := range bj.Spec.Stages {
		scriptParts = append(scriptParts,
			fmt.Sprintf("echo '> [%d/%d] %s'", i+1, len(bj.Spec.Stages), stage.Name),
			stage.Command,
			fmt.Sprintf("echo '  [OK] %s completed'", stage.Name),
		)
	}

	scriptParts = append(scriptParts,
		"echo ''",
		"echo '=== Artifacts ==='",
		"ls -la /workspace/artifacts/ 2>/dev/null || echo '  (none)'",
	)

	script := strings.Join(scriptParts, "\n")

	args := []string{
		"run", "--rm",
		"-v", absSource + ":/workspace/source:z",
		"-v", absOutput + ":/workspace/artifacts:z",
		"-w", "/workspace",
	}

	if bj.Spec.Target.Board != "" {
		args = append(args, "-e", "BOB_BOARD="+bj.Spec.Target.Board)
	}
	if bj.Spec.Target.Platform != "" {
		args = append(args, "-e", "BOB_PLATFORM="+bj.Spec.Target.Platform)
	}
	if bj.Spec.Target.Architecture != "" {
		args = append(args, "-e", "BOB_ARCH="+bj.Spec.Target.Architecture)
	}
	args = append(args, "-e", "BOB_NAME="+bj.Name)

	args = append(args, image, "bash", "-c", script)

	cmd := exec.Command(runtime, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Printf("\nBuild complete. Artifacts in: %s\n", absOutput)
	entries, _ := os.ReadDir(absOutput)
	for _, e := range entries {
		fmt.Printf("  %s\n", e.Name())
	}
	return nil
}
