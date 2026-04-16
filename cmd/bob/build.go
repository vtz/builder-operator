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

	buildv1alpha1 "github.com/centos-automotive-suite/bob/api/v1alpha1"
	"github.com/centos-automotive-suite/bob/internal/buildapi"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func newBuildCmd() *cobra.Command {
	var file string
	var branch string

	cmd := &cobra.Command{
		Use:   "build [name]",
		Short: "Create or re-trigger a BuildJob",
		Long: `Create a BuildJob from a YAML file or re-trigger an existing one.

  bob build -f buildjob.yaml                      # create from file
  bob build -f buildjob.yaml --branch my-feature   # override git revision
  bob build body-ecu-nucleo                        # re-trigger existing BuildJob`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if file != "" {
				return createFromFile(cmd.Context(), file, branch)
			}
			if len(args) == 0 {
				return fmt.Errorf("provide a BuildJob name or use -f <file>")
			}
			return retrigger(cmd.Context(), args[0])
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to BuildJob YAML file")
	cmd.Flags().StringVar(&branch, "branch", "", "Override git revision/branch")
	return cmd
}

func createFromFile(ctx context.Context, path string, branch string) error {
	data, err := os.ReadFile(path)
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
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}

	var result buildapi.BuildJobSummary
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	fmt.Printf("BuildJob created: %s\n", result.Name)
	fmt.Printf("  Namespace: %s\n", result.Namespace)
	fmt.Printf("  Board:     %s\n", result.Board)
	fmt.Printf("  Platform:  %s\n", result.Platform)
	fmt.Printf("  Image:     %s\n", result.Image)
	fmt.Printf("\nWatch progress: bob list\n")
	fmt.Printf("Stream logs:   bob logs %s\n", result.Name)
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
