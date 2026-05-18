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
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newVerifyCmd() *cobra.Command {
	var pubKeyPath string

	cmd := &cobra.Command{
		Use:   "verify [name]",
		Short: "Verify the cosign signature of a build's OCI artifact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			name := args[0]
			ctx := context.Background()

			build, err := c.Get(ctx, name)
			if err != nil {
				return fmt.Errorf("getting build: %w", err)
			}

			if build.OCIArtifactRef == "" {
				return fmt.Errorf("build %q has no OCI artifact to verify", name)
			}

			if pubKeyPath != "" {
				if err := os.Setenv("COSIGN_PUB_KEY", pubKeyPath); err != nil {
					return err
				}
			}

			if err := verifyCosignSignature(build.OCIArtifactRef, build.OCIArtifactDigest); err != nil {
				return err
			}

			fmt.Printf("\nBuild %q: signature verified OK\n", name)
			if build.OCISignatureVerified {
				fmt.Printf("  Pipeline also recorded successful signing\n")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&pubKeyPath, "key", "", "Path to cosign public key (default: cosign.pub or COSIGN_PUB_KEY)")
	return cmd
}
