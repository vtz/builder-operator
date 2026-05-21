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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	"github.com/centos-automotive-suite/bob/internal/buildapi"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func newBuildCmd() *cobra.Command {
	var file string
	var branch string
	var local bool
	var sourceDir string
	var outputDir string
	var pvcName string
	var downloadDir string
	var skipVerify bool
	var force bool

	cmd := &cobra.Command{
		Use:   "build [name]",
		Short: "Create or re-trigger a BuildJob",
		Long: `Create a BuildJob from a YAML file or re-trigger an existing one.

  bob build -f buildjob.yaml                      # create from file on cluster
  bob build -f buildjob.yaml --branch my-feature   # override git revision
  bob build body-ecu-nucleo                        # re-trigger existing BuildJob
  bob build body-ecu-mpu-hostlike --local                  # sync . and build
  bob build body-ecu-mpu-hostlike --local --source ~/code  # sync specific dir
  bob build body-ecu-nucleo -d ./out              # download artifacts (rebuild only if needed)
  bob build body-ecu-nucleo -d ./out --force      # force rebuild and download

When --local is used, the BuildJob is temporarily switched to use your local
source. The next build without --local automatically restores git source.

When -d is used, the CLI checks if the build already succeeded. If so, it
downloads the existing artifacts without retriggering. Use --force to rebuild
regardless.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ns := firstNonEmpty(bobNamespace, os.Getenv("BOB_NAMESPACE"), "bob-builds")
			if local {
				if len(args) == 0 && file == "" {
					return fmt.Errorf("--local requires a BuildJob name or -f <file>")
				}
				if file != "" {
					return runLocalBuild(file, branch, sourceDir, outputDir)
				}
				kubecli := detectKubeClient()
				if kubecli == "" {
					return fmt.Errorf("no Kubernetes CLI found (install oc or kubectl)")
				}
				arch := buildJobArch(kubecli, ns, args[0])
				if err := runSyncWithArch(sourceDir, pvcName, ns, "/", kubecli, arch); err != nil {
					return err
				}
				cleanupSyncPod(kubecli, ns, pvcName)
				if err := switchToPVCAndTrigger(kubecli, ns, args[0], pvcName, "/"); err != nil {
					return err
				}
				if downloadDir != "" {
					return waitAndDownload(cmd.Context(), args[0], downloadDir, skipVerify)
				}
				return nil
			}
			if file != "" {
				name, err := createFromFile(cmd.Context(), file, branch)
				if err != nil {
					return err
				}
				if downloadDir != "" {
					return waitAndDownload(cmd.Context(), name, downloadDir, skipVerify)
				}
				return nil
			}
			if len(args) == 0 {
				return fmt.Errorf("provide a BuildJob name or use -f <file>")
			}
			if err := autoRestoreIfLocal(ns, args[0]); err != nil {
				return fmt.Errorf("restoring git source: %w", err)
			}
			if downloadDir != "" && !force {
				if skipped, err := downloadIfUpToDate(cmd.Context(), args[0], downloadDir, skipVerify); err != nil {
					return err
				} else if skipped {
					return nil
				}
			}
			if err := retrigger(cmd.Context(), args[0]); err != nil {
				return err
			}
			if downloadDir != "" {
				return waitAndDownload(cmd.Context(), args[0], downloadDir, skipVerify)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to BuildJob YAML file")
	cmd.Flags().StringVar(&branch, "branch", "", "Override git revision/branch")
	cmd.Flags().BoolVar(&local, "local", false, "Build on the cluster using local source code")
	cmd.Flags().StringVar(&sourceDir, "source", ".", "Local source directory (used with --local)")
	cmd.Flags().StringVar(&outputDir, "output", "./bob-output", "Local output directory for artifacts (used with --local -f)")
	cmd.Flags().StringVar(&pvcName, "pvc", "source-code", "PVC name for local source upload")
	cmd.Flags().StringVarP(&downloadDir, "download", "d", "", "Wait for build to finish and download artifacts to this directory")
	cmd.Flags().BoolVar(&skipVerify, "skip-verify", false, "Skip cosign signature verification on download")
	cmd.Flags().BoolVar(&force, "force", false, "Force a new build even if the last one succeeded")
	return cmd
}

func createFromFile(ctx context.Context, path string, branch string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	var bj buildv1alpha1.BuildJob
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	if err := decoder.Decode(&bj); err != nil {
		return "", fmt.Errorf("parsing YAML: %w", err)
	}

	if branch != "" && bj.Spec.Source.Git != nil {
		bj.Spec.Source.Git.Revision = branch
	}

	c := newClient()
	ns := c.Namespace
	if bj.Namespace != "" {
		ns = bj.Namespace
	}

	reqBody := buildapi.CreateBuildJobRequest{
		Name: bj.Name,
		Spec: bj.Spec,
	}
	payload, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/v1/namespaces/%s/buildjobs", c.BaseURL, ns)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}

	var result buildapi.BuildJobSummary
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	fmt.Printf("BuildJob created: %s\n", result.Name)
	fmt.Printf("  Namespace: %s\n", result.Namespace)
	fmt.Printf("  Board:     %s\n", result.Board)
	fmt.Printf("  Platform:  %s\n", result.Platform)
	fmt.Printf("  Image:     %s\n", result.Image)
	fmt.Printf("\nWatch progress: bob list\n")
	fmt.Printf("Stream logs:   bob logs %s\n", result.Name)
	return result.Name, nil
}

func autoRestoreIfLocal(namespace, bjName string) error {
	kubecli := detectKubeClient()
	if kubecli == "" {
		return nil
	}
	out, _ := exec.Command(kubecli, "get", "buildjob", bjName, "-n", namespace,
		"-o", `jsonpath={.metadata.annotations.builder\.sdv\.cloud\.redhat\.com/original-source}`).CombinedOutput()
	if len(out) > 0 && string(out) != "" {
		return restoreGitSource(kubecli, namespace, bjName)
	}
	return nil
}

func retrigger(ctx context.Context, name string) error {
	c := newClient()
	result, err := c.Run(ctx, name)
	if err != nil {
		return fmt.Errorf("triggering build: %w", err)
	}

	fmt.Printf("Build triggered: %s\n", result.Name)
	fmt.Printf("  Phase:       %s\n", result.Phase)
	fmt.Printf("  PipelineRun: %s\n", result.PipelineRun)
	if result.Board != "" {
		fmt.Printf("  Board:       %s\n", result.Board)
	}
	if result.Platform != "" {
		fmt.Printf("  Platform:    %s\n", result.Platform)
	}
	fmt.Printf("\nStream logs with: bob logs %s\n", result.Name)
	return nil
}
