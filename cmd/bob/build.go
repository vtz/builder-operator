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
	"fmt"

	"github.com/spf13/cobra"
)

func newBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build <name>",
		Short: "Trigger a build for a BuildJob",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			result, err := c.Run(cmd.Context(), args[0])
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
		},
	}
	return cmd
}
