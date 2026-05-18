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
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show details of a BuildJob",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			bj, err := c.Get(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("getting build job: %w", err)
			}

			fmt.Printf("Name:        %s\n", bj.Name)
			fmt.Printf("Namespace:   %s\n", bj.Namespace)
			fmt.Printf("Phase:       %s\n", bj.Phase)
			fmt.Printf("PipelineRun: %s\n", bj.PipelineRun)

			if bj.Source != nil {
				fmt.Printf("Source:      %s %s", bj.Source.Type, bj.Source.URL)
				if bj.Source.Revision != "" {
					fmt.Printf("@%s", bj.Source.Revision)
				}
				fmt.Println()
			}

			if bj.CommitSHA != "" {
				fmt.Printf("Commit:      %s\n", bj.CommitSHA)
			}

			fmt.Printf("Image:       %s\n", bj.Image)

			if bj.Board != "" {
				fmt.Printf("Board:       %s\n", bj.Board)
			}
			if bj.Platform != "" {
				fmt.Printf("Platform:    %s\n", bj.Platform)
			}
			if bj.Architecture != "" {
				fmt.Printf("Arch:        %s\n", bj.Architecture)
			}
			if bj.ArtifactURI != "" {
				fmt.Printf("Artifact:    %s\n", bj.ArtifactURI)
			}
			if bj.OCIArtifactRef != "" {
				fmt.Printf("OCI Ref:     %s\n", bj.OCIArtifactRef)
				if bj.OCIArtifactDigest != "" {
					fmt.Printf("OCI Digest:  %s\n", bj.OCIArtifactDigest)
				}
				if bj.OCISignatureVerified {
					fmt.Printf("Signed:      yes (cosign)\n")
				}
			}

			if len(bj.Stages) > 0 {
				fmt.Println("\nStages:")
				w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
				for _, s := range bj.Stages {
					fmt.Fprintf(w, "  %s\t%s\n", s.Name, s.State)
				}
				w.Flush()
			}

			return nil
		},
	}
}
